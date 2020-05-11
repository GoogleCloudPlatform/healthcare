# Copyright 2018 Google LLC. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Inference via AutoML or Cloud ML engine.

The inference module allows medical imaging ML models to be more easily
integrated into clinical workflows. PACS systems (and other imaging archives)
can push DICOM instances into Cloud Healthcare DICOM stores, allowing ML models
to be triggered for inference. This inference results can then be stored as
DICOM structured reports in the Cloud Healthcare API, which can be retrieved by
the customer. This script does the following actions:

1) Listens to new DICOM to be inserted into a DICOM Store via Cloud Pubsub.

   When new DICOM instances are stored in Cloud Healthcare API, a Cloud Pubsub
   message is generated for the new instance. This module will listen to all
   incoming messages for the Cloud Pubsub subscription specified in
   --subscription_path.

2) Retrieves the instance from Cloud Healthcare API.

   The DICOMWeb WADO-RS request will be invoked to retrieve the DICOM instance.
   Since the model is trained on JPEG images, this script will retrieve the
   instance in that format. It will utilize the Cloud Healthcare API's ability
   to transcode instances in comsumer format (e.g. JPEG).

3) Invokes Cloud Machine Learning Engine for Inference.

   The image in JPEG format will be sent to the pre-trained model (specified
   by --model_path) served by Cloud ML Engine. Cloud ML Engine will return both
   the predicted class and the score for the image.

4) Store the inference results in Cloud Healthcare API.

   This step is optional and is only enabled by setting the
   --dicom_store_path flag.

   The inference results can be stored as a DICOM structured report. This is
   a format that can store free text format (amongst other things). In this
   case, the script is going to store the predicted class and score from step
   (3). This DICOM strucuted report will be stored back to the Cloud Healthcare
   API in the given DICOM store (--dicom_store_path). This instance
   can be then be retrieved by the client.
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import abc
import argparse
import base64
import logging
import posixpath
import sys
import traceback

from dicomweb_client import session_utils
from dicomweb_client.api import DICOMwebClient
from dicomweb_client.api import load_json_dataset
import googleapiclient.discovery
from hcls_imaging_ml_toolkit import dicom_path
from hcls_imaging_ml_toolkit import exception
from hcls_imaging_ml_toolkit import pubsub_format
from hcls_imaging_ml_toolkit import tag_values
from oauth2client.client import GoogleCredentials
import pydicom

from google.api_core.exceptions import InvalidArgument
from google.api_core.exceptions import PermissionDenied
from google.cloud import automl_v1beta1
from google.cloud import pubsub_v1

# Output this module's logs (INFO and above) to stdout.
_logger = logging.getLogger(__name__)
_logger.addHandler(logging.StreamHandler(sys.stdout))
_logger.setLevel(logging.INFO)

FLAGS = None

# Prefix for Cloud Healthcare API.
_HEALTHCARE_API_URL_PREFIX = 'https://healthcare.googleapis.com/v1'

# Credentails used to access Cloud Healthcare API.
_CREDENTIALS = GoogleCredentials.get_application_default().create_scoped(
    ['https://www.googleapis.com/auth/cloud-platform'])

# Number of times to retry failing CMLE predictions.
_NUM_RETRIES_CMLE = 5

# Accepted character set.
_CHARACTER_SET = 'ISO_IR 192'

# Little Endian Transfer Syntax.
_IMPLICIT_VR_LITTLE_ENDIAN = '1.2.840.10008.1.2'

_MODALITY_KEY_WORD = 'Modality'


class Predictor(object):
  """Abstract base class for ML Predictor."""
  __metaclass__ = abc.ABCMeta

  @abc.abstractmethod
  def Predict(self, image_jpeg_bytes):
    # type: str -> (str, str)
    """Runs inference and returns predicted class and score."""
    raise NotImplementedError


class CMLEPredictor(Predictor):
  """Handler for CMLE predictor.

  Attributes:
    model_path: Path to model.
  """

  def __init__(self, model_path):
    self._model_path = model_path

  def Predict(self, image_jpeg_bytes):
    # type: str -> (str, str)
    """Runs inference on image using Cloud ML Engine.

    This will invoke the model with the given image and will return the
    predicted class and score.

    Args:
      image_jpeg_bytes: Bytes of JPEG image.

    Returns:
      (class, score) tuple.

      class is the predicted class.
      score is the predicted score.

    Raises:
      RuntimeError: if failed to get inference results.
    """

    # CMLE requires images to be encoded in this format:
    # https://cloud.google.com/ml-engine/docs/v1/predict-request
    input_data = {
        'instances': [{
            'inputs': [{
                'b64': base64.b64encode(image_jpeg_bytes).decode()
            }]
        }]
    }
    # Disable cache discovery due to following issue:
    # https://github.com/google/google-api-python-client/issues/299
    service = googleapiclient.discovery.build('ml', 'v1', cache_discovery=False)
    response = service.projects().predict(
        name=self._model_path,
        body=input_data).execute(num_retries=_NUM_RETRIES_CMLE)

    # Propagate the error.
    if 'error' in response:
      raise RuntimeError(response['error'])

    # Return the predictions.
    prediction = response['predictions'][0]
    return prediction['classes'], prediction['scores']


class AutoMLPredictor(Predictor):
  """Handler for AutoML Vision predictor.

  Attributes:
    model_path: Path to model.
  """

  def __init__(self, model_path):
    self._model_path = model_path

  def Predict(self, image_jpeg_bytes):
    # type: str -> (str, str)
    """Runs inference on image using AutoML.

    This will invoke the model with the given image and will return the
    predicted class and score.

    Args:
      image_jpeg_bytes: Bytes of JPEG image.

    Returns:
      (class, score) tuple.

      class is the predicted class.
      score is the predicted score.

    Raises:
      RuntimeError: if failed to get valid inference results.
    """
    payload = {'image': {'image_bytes': image_jpeg_bytes}}
    params = {}
    prediction_client = automl_v1beta1.PredictionServiceClient()
    response = prediction_client.predict(self._model_path, payload, params)
    if len(response.payload) != 1:
      raise RuntimeError('AutoML response payload size should be of size 1')
    result = response.payload[0]
    return result.display_name, result.classification.score


def _BuildSR(study_json, text, series_uid, instance_uid):
  # type: (Dict, str, str, str) -> pydicom.dataset.Dataset
  """Builds and returns a Basic Text DICOM Structured Report instance.

  Args:
    study_json: Dict of study level information to populate the SR.
    text: Text string to use for the Basic Text DICOM SR.
    series_uid: UID of the series to use for the SR.
    instance_uid: UID of the instance to use for the SR.

  Returns:
    pydicom.dataset.Dataset representing the Structured Report.
  """
  dataset = load_json_dataset(study_json)
  dataset.SOPClassUID = tag_values.BASIC_TEXT_SR_CUID
  dataset.SeriesInstanceUID = series_uid
  dataset.SOPInstanceUID = instance_uid
  dataset.Modality = tag_values.SR_MODALITY

  content_dataset = pydicom.dataset.Dataset()
  dataset.ContentSequence = pydicom.sequence.Sequence([content_dataset])
  content_dataset.RelationshipType = 'CONTAINS'
  content_dataset.ValueType = 'TEXT'
  content_dataset.TextValue = text

  dataset.fix_meta_info(enforce_standard=True)
  # Must be set but is overwritten later during `dcmwrite()`.
  dataset.file_meta.FileMetaInformationGroupLength = 0
  return dataset


class PubsubMessageHandler(object):
  """Handler for incoming Pubsub messages.

  Attributes:
    publisher: PublisherClient used to publish pubsub messages.
  """

  def __init__(self, predictor, dicom_store_path):
    """Inits PubsubMessageHandler with args.

    Args:
      predictor: Object used to get prediction results.
      dicom_store_path: DICOM store used to store inference results.
    """
    self._predictor = predictor
    # May raise exception if the path is invalid.
    if dicom_store_path:
      self._dicom_store_path = dicom_path.FromString(dicom_store_path,
                                                     dicom_path.Type.STORE)
    else:
      self._dicom_store_path = None
    self._success_count = 0
    self.publisher = pubsub_v1.PublisherClient()

  def _IsMammoInstance(self, input_client, instance_path):
    # type: (dicomweb_client.DICOMwebClient, dicom_path.Path) -> bool
    """Returns whether the DICOM instance is of type MG modality.

    Args:
      input_client: dicomweb_client.DICOMwebClient used to perform search query.
      instance_path: dicom_path.Path of DICOM instance.

    Returns:
      ParsedMessage or None if the message is invalid.
    """
    instance_json_list = input_client.search_for_instances(
        instance_path.study_uid,
        instance_path.series_uid,
        fields=[_MODALITY_KEY_WORD],
        search_filters={'SOPInstanceUID': instance_path.instance_uid})
    if not instance_json_list:
      return False
    dataset = load_json_dataset(instance_json_list[0])
    return dataset.get(_MODALITY_KEY_WORD) == tag_values.MG_MODALITY

  def _PublishInferenceResultsReady(self, image_instance_path):
    # type: str -> None
    """Publishes a results ready notification to the supplied Pubsub channel.

    If the Inference Module is supplied FLAGS.publisher_topic_path flag, it will
    attempt to publish any inference results to that channel.

    Args:
      image_instance_path: Path of DICOM study. This should be formatted as
        follows
        projects/{PROJECT_ID}/locations/{LOCATION_ID}/datasets/{DATASET_ID}/
        dicomStores/{DICOM_STORE_ID}/dicomWeb/studies/{STUDY_ID}/series/
        ${SERIES_ID}/instances/{INSTANCE_ID}
    """
    if FLAGS.publisher_topic_path is None:
      return

    try:
      self.publisher.publish(
          FLAGS.publisher_topic_path, data=image_instance_path)
      _logger.info('Published inference results ready message')
    except TypeError as e:
      _logger.error('Invalid type sent to publish channel: %s', e)

  def PubsubCallback(self, message):
    # type: pubsub_v1.Message -> None
    """Processes a Pubsub message.

    This function handles the processing of a single Pub/Sub message. It will
    ACK any message that leads to an exception.

    Args:
      message: pubsub_v1.Message being processed.
    """
    try:
      self._PubsubCallback(message)
    except Exception as e:  # pylint: disable=broad-except
      # This is pretty broad exception handling, real deployments should try
      # to have more robust error handing.
      _logger.error(e)
      traceback.print_exc()
      _logger.error('Unexpected exception...acking message')
      message.ack()

  def _PubsubCallback(self, message):
    # type: pubsub_v1.Message -> None
    """Processes a Pubsub message.

    This function will retrieve the instance (specified in Pubsub message) from
    the Cloud Healthcare API. Then it will invoke CMLE to get the prediction
    results. Finally (and optionally), it will store the inference results back
    to the Cloud Healthcare API as a DICOM Structured Report. The instance URL
    to the Structured Report containing the prediction is then published to a
    pub/sub.

    Args:
      message: Incoming pubsub message.
    """
    image_instance_path = message.data.decode()
    _logger.debug('Received instance in pubsub feed: %s', image_instance_path)
    try:
      parsed_message = pubsub_format.ParseMessage(message,
                                                  dicom_path.Type.INSTANCE)
    except exception.CustomExceptionError:
      _logger.info('Invalid input path: %s', image_instance_path)
      message.ack()
      return
    input_path = parsed_message.input_path
    authed_session = session_utils.create_session_from_gcp_credentials()
    dicomweb_url = posixpath.join(_HEALTHCARE_API_URL_PREFIX,
                                  input_path.dicomweb_path_str)
    input_client = DICOMwebClient(dicomweb_url, authed_session)
    if not self._IsMammoInstance(input_client, input_path):
      _logger.info('Instance is not of type MG modality, ignoring message: %s',
                   image_instance_path)
      message.ack()
      return

    _logger.info('Processing instance: %s', image_instance_path)
    # Retrieve instance from DICOM API in JPEG format.
    image_jpeg_bytes = input_client.retrieve_instance_rendered(
        input_path.study_uid,
        input_path.series_uid,
        input_path.instance_uid,
        media_types=('image/jpeg',))
    # Retrieve study level information
    study_json = input_client.search_for_studies(
        fields=['all'],
        search_filters={'StudyInstanceUID': input_path.study_uid})[0]
    # Get the predicted score and class from the inference model in Cloud ML or
    # AutoML.
    try:
      predicted_class, predicted_score = self._predictor.Predict(
          image_jpeg_bytes)
    except PermissionDenied as e:
      _logger.error('Permission error running prediction service: %s', e)
      message.nack()
      return
    except InvalidArgument as e:
      _logger.error('Invalid arguments when running prediction service: %s', e)
      message.nack()
      return

    # Print the prediction.
    text = 'Base path: %s\nPredicted class: %s\nPredicted score: %s' % (
        image_instance_path, predicted_class, predicted_score)
    _logger.info(text)

    # If user requested destination DICOM store for inference, create a DICOM
    # structured report that stores the prediction.
    if self._dicom_store_path:
      # Create DICOMwebClient with output url.
      output_dicom_web_url = posixpath.join(
          _HEALTHCARE_API_URL_PREFIX, self._dicom_store_path.dicomweb_path_str)
      output_client = DICOMwebClient(output_dicom_web_url, authed_session)

      # Generate series uid and instance uid for the structured report.
      sr_instance_uid = pydicom.uid.generate_uid(prefix=None)
      sr_series_uid = pydicom.uid.generate_uid(prefix=None)
      sr_dataset = _BuildSR(study_json, text, sr_series_uid, sr_instance_uid)
      try:
        output_client.store_instances([sr_dataset])
      except RuntimeError as e:
        _logger.error('Error storing DICOM in API: %s', e)
        message.nack()
        return

      # If user requested that new structured reports be published to a channel,
      # publish the instance path of the Structured Report
      structured_report_path = dicom_path.FromPath(
          self._dicom_store_path,
          study_uid=input_path.study_uid,
          series_uid=sr_series_uid,
          instance_uid=sr_instance_uid)
      self._PublishInferenceResultsReady(str(structured_report_path))
      _logger.info('Published structured report with path: %s',
                   structured_report_path)
    # Ack the message (successful or invalid message).
    message.ack()
    self._success_count += 1

  def GetSuccessCount(self):
    # type: None -> int
    """Returns the number of Pubsub messages successfully processed."""
    return self._success_count


def main():
  if FLAGS.prediction_service == 'CMLE':
    predictor = CMLEPredictor(FLAGS.model_path)
  elif FLAGS.prediction_service == 'AutoML':
    predictor = AutoMLPredictor(FLAGS.model_path)
  else:
    raise ValueError('FLAGS.prediction_service must be CMLE or AutoML.')

  handler = PubsubMessageHandler(predictor, FLAGS.dicom_store_path)
  subscriber = pubsub_v1.SubscriberClient()
  future = subscriber.subscribe(FLAGS.subscription_path, handler.PubsubCallback)
  try:
    # If timeout is set, wait for FLAGS.pubsub_timeout seconds until messages
    # are processed on the pubsub channel.
    future.result(timeout=FLAGS.pubsub_timeout)
  except KeyboardInterrupt:
    # User exits the script early.
    future.cancel()
    _logger.info('Received keyboard interrupt, exiting...')
  except pubsub_v1.exceptions.TimeoutError:
    # No messages are processed in FLAGS.pubsub_timeout seconds.
    future.cancel()
    assert (handler.GetSuccessCount() >
            0), 'Timeout but no pubsub messages successfully processed'


if __name__ == '__main__':
  parser = argparse.ArgumentParser()
  parser.add_argument(
      '--subscription_path',
      type=str,
      required=True,
      help=
      'Pub/Sub Subscription ID associated with the topic for notifications of '
      'new DICOM instances. The Inference Module subscribes to notifications on'
      ' this channel and runs inference on new DICOM instances added to the '
      'store.')
  parser.add_argument(
      '--publisher_topic_path',
      type=str,
      required=False,
      help=
      'Pub/Sub topic id, which is used to indicate that inference results are '
      'ready. Each notification contains a path to the DICOM Structured Report '
      'containing predictions.')
  parser.add_argument(
      '--model_path',
      type=str,
      required=True,
      help='Path of Cloud ML Engine model used for inference.')
  parser.add_argument(
      '--dicom_store_path',
      type=str,
      default='',
      help=
      'DICOM store used to store inference results. If empty, this script will '
      'not attempt to store inference results back into Healthcare API.')
  parser.add_argument(
      '--prediction_service',
      type=str,
      default='CMLE',
      help='Service to call for prediction, either "CMLE" or "AutoML"')
  parser.add_argument(
      '--pubsub_timeout',
      type=int,
      default=None,
      help=
      'Number of seconds to wait for pubsub message to process until timeout. '
      'If set to None, it will wait indefinitely.')

  FLAGS, unparsed = parser.parse_known_args()
  if FLAGS.publisher_topic_path and not FLAGS.dicom_store_path:
    parser.error('--publisher_topic_path requires --dicom_store_path '
                 'to be set.')
  main()
