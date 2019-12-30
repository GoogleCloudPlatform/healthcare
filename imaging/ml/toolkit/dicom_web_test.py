# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Tests for dicom_web.py."""

from __future__ import absolute_import
from __future__ import division
from __future__ import google_type_annotations
from __future__ import print_function

from typing import Any, Dict, Text, Tuple
from absl.testing import absltest
from absl.testing import parameterized
import google_auth_httplib2
import httplib2
import mock
from six.moves import http_client

import google.auth.credentials
from toolkit import dicom_json
from toolkit import dicom_web
from toolkit import tags
_URI = 'http://healthcareapi.com/test'
_GET = 'GET'
_BODY = 'body'
_TEST_INSTANCES = 10


def _CreateMockInstanceMetadata() -> Dict[Text, Any]:
  instance_metadata = {}
  dicom_json.Insert(instance_metadata, tags.STUDY_INSTANCE_UID, 1)
  dicom_json.Insert(instance_metadata, tags.SERIES_INSTANCE_UID, 2)
  dicom_json.Insert(instance_metadata, tags.SOP_INSTANCE_UID, 3)
  return instance_metadata


_MOCK_CT_INSTANCE_METADATA = _CreateMockInstanceMetadata()


def FakeHttpResponse(error_code: Text) -> Tuple[httplib2.Response, Text]:
  return (httplib2.Response({'status': error_code}), _BODY)


class DicomWebTest(parameterized.TestCase):

  @mock.patch.object(
      google.auth, 'default', return_value=(mock.MagicMock(), mock.MagicMock()))
  def setUp(self, *_):
    super(DicomWebTest, self).setUp()
    self._dwc = dicom_web.DicomWebClientImpl(credentials=mock.MagicMock())

  @mock.patch.object(httplib2, 'Http')
  @mock.patch.object(google_auth_httplib2, 'AuthorizedHttp')
  def testInvokeHttpRequest(self, *_):
    http_mock = mock.MagicMock()
    httplib2.Http.return_value = http_mock
    http_mock.request.return_value = FakeHttpResponse(http_client.OK)
    google_auth_httplib2.AuthorizedHttp.return_value = http_mock
    resp, content = self._dwc._InvokeHttpRequest(_URI, _GET)
    self.assertEqual(resp.status, 200)
    self.assertEqual(content, _BODY)

  @parameterized.parameters(dicom_web._TOO_MANY_REQUESTS_ERROR,
                            http_client.REQUEST_TIMEOUT,
                            http_client.SERVICE_UNAVAILABLE,
                            http_client.GATEWAY_TIMEOUT)
  @mock.patch.object(httplib2, 'Http')
  @mock.patch.object(google_auth_httplib2, 'AuthorizedHttp')
  def testInvokeHttpRequestWithRetriedErrors(self, error_code, *_):
    http_mock = mock.MagicMock()
    httplib2.Http.return_value = http_mock
    http_mock.request.side_effect = [
        FakeHttpResponse(error_code),
        FakeHttpResponse(http_client.OK)
    ]
    google_auth_httplib2.AuthorizedHttp.return_value = http_mock
    resp, content = self._dwc._InvokeHttpRequest(_URI, _GET)
    self.assertEqual(resp.status, 200)
    self.assertEqual(content, _BODY)

  def testGetAllMetaData(self):
    expected_url = (
        'http://test/studies/1/instances/?includefield=%s&'
        'includefield=%s&includefield=%s&limit=%d' %
        (tags.STUDY_INSTANCE_UID.number, tags.SERIES_INSTANCE_UID.number,
         tags.SOP_INSTANCE_UID.number, _TEST_INSTANCES))
    mock_client = mock.create_autospec(dicom_web.DicomWebClientImpl)
    mock_client.QidoRs.return_value = [_MOCK_CT_INSTANCE_METADATA]
    dicomweb_url = 'http://test'
    study_uid = '1'
    tag_list = [
        tags.STUDY_INSTANCE_UID, tags.SERIES_INSTANCE_UID, tags.SOP_INSTANCE_UID
    ]

    all_meta_data = dicom_web.GetInstancesMetadata(mock_client, dicomweb_url,
                                                   study_uid, tag_list,
                                                   _TEST_INSTANCES)
    self.assertLen(all_meta_data, 1)
    self.assertEqual(all_meta_data[0], _MOCK_CT_INSTANCE_METADATA)
    mock_client.QidoRs.assert_called_once()
    call_args, _ = mock_client.QidoRs.call_args
    self.assertEqual(call_args[0], expected_url)

  def testStowRsJsonError(self):
    bulk_data = dicom_web.DicomBulkData(uri='', data=b'', content_type='a/b/c')
    with self.assertRaises(Exception):
      self._dwc.StowRsJson('', [{}], [bulk_data])


if __name__ == '__main__':
  absltest.main()
