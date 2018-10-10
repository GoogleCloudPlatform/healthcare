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
from email import encoders
from email.mime import application
from email.mime import multipart
import json
import logging
import mimetools
import os
import re
import sys

import googleapiclient.discovery
import httplib2
from oauth2client.client import GoogleCredentials
import pydicom
from requests_toolbelt.multipart import decoder

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
_HEALTHCARE_API_URL_PREFIX = 'https://healthcare.googleapis.com/v1alpha'

# Credentails used to access Cloud Healthcare API.
_CREDENTIALS = GoogleCredentials.get_application_default().create_scoped(
    ['https://www.googleapis.com/auth/cloud-platform'])

# SOP Class UID for Basic Text Structured Reports.
_BASIC_TEXT_SR_CUID = '1.2.840.10008.5.1.4.1.1.88.11'

# Structured Report SOP Class UID.
_STRUCTURED_REPORT_ID = '1.2.840.10008.5.1.4.1.1.88.11'

# DICOM Tags.
_SOP_INSTANCE_UID_TAG = '00080018'
_SOP_CLASS_UID_TAG = '00080016'

_VALUE_TYPE = 'Value'

# Indices for study and series UID component of path contained in Pubsub message
_STUDY_UID_INDEX = 10
_SERIES_UID_INDEX = 12

# Number of times to retry failing CMLE predictions.
_NUM_RETRIES_CMLE = 5


# TODO(b/111960222): Potentially add to ML toolkit.
def _WadoRS(instance_path):
  # type: str -> str
  """Receives instance in JPEG format using WADO-RS protocol.

  WADO-RS is one of the standard protocols specified by DICOMWeb protocol. It
  allows clients to retrieve instances in various formats. In this case we will
  retrieve the instance in JPEG format, from the Cloud Healthcare API specified
  by _HEALTHCARE_API_URL_PREFIX.

  Args:
    instance_path: Path of DICOM instance. This is found in the contents of the
      Pubsub message. This should be formatted as follows:
        projects/{PROJECT_ID}/locations/{LOCATION_ID}/datasets/{DATASET_ID}/
        dicomStores/{DICOM_STORE_ID}/dicomWeb/studies/{STUDY_UID}/series/
        {SERIES_UID}/instances/{INSTANCE_UID}

  Returns:
    content: The bytes for the JPEG image.

  Raises:
    RuntimeError: If failed to retrieve or process instance.
  """
  wado_url = os.path.join(_HEALTHCARE_API_URL_PREFIX, instance_path)
  http = httplib2.Http()
  http = _CREDENTIALS.authorize(http)

  # Headers for receiving DICOM in JPEG Baseline format.
  headers = {
      'Accept': 'multipart/related; type="image/jpeg"; '
                'transfer-syntax=1.2.840.10008.1.2.4.50'
  }
  resp, content = http.request(wado_url, 'GET', headers=headers)
  if resp.status != 200:
    raise RuntimeError(
        'Failed to retrieve DICOM instance: (%s, %s)' % (resp, content))

  multipart_data = decoder.MultipartDecoder(content, resp['content-type'])
  if len(multipart_data.parts) != 1:
    raise RuntimeError('Invalid number of WADO-RS response parts (need 1): %s' %
                       (str(len(multipart_data.parts))))
  return multipart_data.parts[0].content


# TODO(b/111960222): Potentially add to ML toolkit.
def _StowRS(study_path, instance_bytes):
  # type: (str, str) -> None
  """Stores instance in Cloud Healthcare API using STOW-RS protocol.

  STOW-RS is one of the standard protocols specified by DICOMWeb protocol. It
  allows clients to store DICOM instances. In this case we will
  store the instance in in the Cloud Healthcare API specified by
  _HEALTHCARE_API_URL_PREFIX.

  Args:
    study_path: Path of DICOM study. This should be formatted as follows:
      projects/{PROJECT_ID}/locations/{LOCATION_ID}/datasets/{DATASET_ID}/
      dicomStores/{DICOM_STORE_ID}/dicomWeb/studies
    instance_bytes: Bytes for instances.

  Raises:
    RuntimeError: If failed to store instance.
  """
  stow_url = os.path.join(_HEALTHCARE_API_URL_PREFIX, study_path)
  http = httplib2.Http()
  http = _CREDENTIALS.authorize(http)

  root = multipart.MIMEMultipart(
      subtype='related', boundary=mimetools.choose_boundary())
  # root should not write out its own headers
  setattr(root, '_write_headers', lambda self: None)
  part = application.MIMEApplication(
      instance_bytes, 'dicom', _encoder=encoders.encode_noop)
  root.attach(part)

  boundary = root.get_boundary()
  content_type = (
      'multipart/related; type="application/dicom"; boundary="%s"') % boundary
  headers = {'content-type': content_type}

  resp, content = http.request(
      stow_url, method='POST', body=root.as_string(), headers=headers)
  if resp.status != 200:
    raise RuntimeError(
        'Failed to store DICOM instance in Healthcare API: (%s, %s)' %
        (resp.status, content))


# TODO(b/111960222): Potentially add to ML toolkit.
def _QidoRs(qido_url):
  # type: str -> List
  """Performs the request, and returns the parsed JSON response.

  QIDO-RS is one of the standard protocols specified by DICOMWeb protocol. It
  allows clients to query metadata for DICOM instances. In this case we will
  store the instance in in the Cloud Healthcare API specified by
  _HEALTHCARE_API_URL_PREFIX.

  Args:
    qido_url: URL for the QIDO request.

  Returns:
    The parsed JSON response content.

  Raises:
    RuntimeError: if the response status was not 200.
  """
  http = httplib2.Http()
  http = _CREDENTIALS.authorize(http)
  resp, content = http.request(qido_url, 'GET')
  if resp.status != 200:
    raise RuntimeError(
        'QidoRs error. Response Status: %d,\nURL: %s,\nContent: %s.' %
        (resp.status, qido_url, content))
  return json.loads(content)


class Predictor(object):
  """ Abstract base class for ML Predictor."""
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
            'inputs': {
                'b64': base64.b64encode(image_jpeg_bytes)
            }
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


# TODO(b/111960222): Potentially add to ML toolkit.
def _BuildSR(text, study_uid, series_uid, instance_uid):
  # (str, str, str, str) -> str
  """Builds and returns a Basic Text DICOM Structured Report instance.

  Args:
    text: Text string to use for the Basic Text DICOM SR.
    study_uid: UID of the study to use for the SR.
    series_uid: UID of the series to use for the SR.
    instance_uid: UID of the instance to use for the SR.

  Returns:
    The serialized form of the DICOM SR instance.
  """
  dataset = pydicom.dataset.Dataset()
  dataset.SOPClassUID = _BASIC_TEXT_SR_CUID
  dataset.StudyInstanceUID = study_uid
  dataset.SeriesInstanceUID = series_uid
  dataset.SOPInstanceUID = instance_uid

  content_dataset = pydicom.dataset.Dataset()
  dataset.ContentSequence = pydicom.sequence.Sequence([content_dataset])
  content_dataset.RelationshipType = 'CONTAINS'
  content_dataset.ValueType = 'TEXT'
  content_dataset.TextValue = text

  dataset.is_little_endian = True
  dataset.is_implicit_VR = True

  fp = pydicom.filebase.DicomBytesIO()
  pydicom.filewriter.dcmwrite(fp, dataset, write_like_original=False)
  bytestring = fp.parent.getvalue()
  fp.close()
  return bytestring


class PubsubMessageHandler(object):
  """Handler for incoming Pubsub messages.

  Attributes:
    predictor: Object used to get prediction results.
    dicom_store_path: DICOM store used to store inference results.
  """

  def __init__(self, predictor, dicom_store_path):
    self._predictor = predictor
    self._dicom_store_path = dicom_store_path
    self._success_count = 0
    self.publisher = pubsub_v1.PublisherClient()

  def _ShouldFilterMessage(self, instance_path):
    # type: pubsub_v1.Message -> bool
    """Returns whether to filter the given pubsub message.

    # The DICOM store publishing messages may be used for storage of both the
    original DICOM instances and the Structured Reports. We need to filter out
    SRs such that only DICOM instances are processed.

    Args:
      instance_path: Path of DICOM instances.

    Returns:
      Whether the instance notification should be filtered.
    """
    split_instance_path = instance_path.split('/')
    removed_instance_uid_path = os.path.join(*split_instance_path[:-1])
    instance_uid = split_instance_path[-1]
    qido_url = (
        '%s/%s?%s=%s' % (_HEALTHCARE_API_URL_PREFIX, removed_instance_uid_path,
                         _SOP_INSTANCE_UID_TAG, instance_uid))

    # The content is JSON containing a list of DICOM instances
    parsed_content = _QidoRs(qido_url)
    # TODO(jonluca) find a better filtering algorithm
    sop_class_uid = parsed_content[0][_SOP_CLASS_UID_TAG][_VALUE_TYPE][0]
    return sop_class_uid == _STRUCTURED_REPORT_ID

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
    # TODO(b/114201645) when converting to Python 3, change image_instance_path
    # to bytes
    try:
      self.publisher.publish(
          FLAGS.publisher_topic_path, data=image_instance_path)
      _logger.info('Published inference results ready message')
    except TypeError as e:
      _logger.error('Invalid type sent to publish channel: %s', e.message)


  def PubsubCallback(self, message):
    # type: pubsub_v1.Message -> None
    """Processess a Pubsub message.

    This function will retrieve the instance (specified in Pubsub message) from
    the Cloud Healthcare API. Then it will invoke CMLE to get the prediction
    results. Finally (and optionally), it will store the inference results back
    to the Cloud Healthcare API as a DICOM Structured Report. The instance URL
    to the Structured Report containing the prediction is then published to a
    pub/sub.

    Args:
      message: Incoming pubsub message.
    """
    image_instance_path = message.data
    _logger.debug('Received instance in pubsub feed: %s', image_instance_path)
    # Filter messages that correspond to inference results or that don't match
    # the type of image we are expecting
    if self._ShouldFilterMessage(image_instance_path):
      message.ack()
      return

    _logger.info('Processing instance: %s', image_instance_path)

    # Retrieve instance from DICOM API in JPEG format.
    image_jpeg_bytes = _WadoRS(image_instance_path)

    # Get the predicted score and class from the inference model in Cloud ML or
    # AutoML.
    try:
      predicted_class, predicted_score = self._predictor.Predict(
          image_jpeg_bytes)
    except PermissionDenied as e:
      _logger.error('Permission error running prediction service: %s',
                    e.message)
      message.nack()
      return
    except InvalidArgument as e:
      _logger.error('Invalid arguments when running prediction service: %s',
                    e.message)
      message.nack()
      return

    # Print the prediction.
    text = 'Base path: %s\nPredicted class: %s\nPredicted score: %s' % (
        image_instance_path, predicted_class, predicted_score)
    _logger.info(text)

    # If user requested destination DICOM store for inference, create a DICOM
    # structured report that stores the prediction.
    if self._dicom_store_path:
      # Generate (study_uid, series_uid, instance_uid) triplet for presentation
      # state. The study_uid and series_uid will be the same as the original
      # instance and extracted from Pubsub path:
      # projects/{PROJECT_ID}/locations/{LOCATION_ID}/datasets/{DATASET_ID}/
      # dicomStores/{DICOM_STORE_ID}/dicomWeb/studies/{STUDY_UID}/series/
      # {SERIES_UID}/instances/{INSTANCE_UID}
      split_image_instance_path = image_instance_path.split('/')
      sr_study_uid = split_image_instance_path[_STUDY_UID_INDEX]
      sr_series_uid = split_image_instance_path[_SERIES_UID_INDEX]
      # Set instance_uid to random number.
      sr_instance_uid = pydicom.uid.generate_uid()

      # Store the DICOM structured report in same series using Healthcare API.
      dicom_sr = _BuildSR(text, sr_study_uid, sr_series_uid, sr_instance_uid)
      study_path = os.path.join(self._dicom_store_path, 'dicomWeb', 'studies')
      try:
        _StowRS(study_path, dicom_sr)
      except RuntimeError as e:
        _logger.error('Error storing DICOM in API: %s', e.message)
        message.nack()
        return

      # If user requested that new structured reports be published to a channel,
      # publish the instance path of the Structured Report
      structured_report_path = os.path.join(study_path, sr_study_uid, 'series',
                                            sr_series_uid, 'instances',
                                            sr_instance_uid)
      self._PublishInferenceResultsReady(structured_report_path)

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
