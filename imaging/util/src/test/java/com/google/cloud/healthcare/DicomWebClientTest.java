// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package com.google.cloud.healthcare;

import static com.google.common.truth.Truth.assertThat;
import static org.junit.Assert.assertNotNull;
import static org.junit.Assert.assertNull;
import static org.junit.Assert.fail;
import static org.junit.jupiter.api.Assertions.assertThrows;

import com.github.danieln.multipart.MultipartInput;
import com.github.danieln.multipart.PartInput;
import com.google.api.client.testing.http.HttpTesting;
import com.google.common.io.ByteStreams;
import java.io.ByteArrayInputStream;
import java.util.Arrays;
import org.json.JSONArray;
import org.json.JSONObject;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.junit.runners.JUnit4;

@RunWith(JUnit4.class)
public final class DicomWebClientTest {

  private FakeWebServer fakeDicomWebServer;
  private DicomWebClient client;

  @Before
  public void setUp() {
    fakeDicomWebServer = new FakeWebServer();
    client = new DicomWebClient(fakeDicomWebServer.createRequestFactory(), HttpTesting.SIMPLE_URL);
  }

  @Test
  public void testDicomWebClient_Wado() throws Exception {
    byte[] dicomInstance = TestUtils.readTestFile(TestUtils.TEST_MR_FILE);

    fakeDicomWebServer.addWadoResponse(dicomInstance);
    MultipartInput resp = client.wadoRs("instanceName");

    PartInput part = resp.nextPart();
    assertNotNull(part);

    byte[] actual = ByteStreams.toByteArray(part.getInputStream());
    if (!Arrays.equals(dicomInstance, actual)) {
      fail("wadoRs returned unexpected DICOM bytes");
    }

    assertNull("WadoRs returned more than one part", resp.nextPart());
  }

  @Test
  public void testDicomWebClient_WadoError() throws Exception {
    fakeDicomWebServer.addResponseWithStatusCode(404);
    assertThrows(DicomWebClient.DicomWebException.class, () -> client.wadoRs("instanceName"));
  }

  @Test
  public void testDicomWebClient_Qido() throws Exception {
    String qidoResponse =
        "[{\"0020000E\":{\"vr\":\"UI\",\"Value\":[\"1.2.840.113619.2.176.3596.3364818.7819.1259708454.108\"]}}]";
    fakeDicomWebServer.addJsonResponse(qidoResponse);
    JSONArray jsonArray = client.qidoRs("query");
    assertThat(jsonArray.length()).isEqualTo(1);
    JSONObject tag = jsonArray.getJSONObject(0).getJSONObject("0020000E");
    JSONArray tagArray = tag.getJSONArray("Value");
    assertThat(tagArray.length()).isEqualTo(1);
    String tagValue = tagArray.getString(0);
    assertThat(tagValue).isEqualTo("1.2.840.113619.2.176.3596.3364818.7819.1259708454.108");
  }

  @Test
  public void testDicomWebClient_QidoError() throws Exception {
    fakeDicomWebServer.addResponseWithStatusCode(404);
    assertThrows(DicomWebClient.DicomWebException.class, () -> client.qidoRs("query"));
  }

  @Test
  public void testDicomWebClient_NothingInjected() throws Exception {
    assertThrows(DicomWebClient.DicomWebException.class, () -> client.qidoRs("query"));
  }

  @Test
  public void testDicomWebClient_StowError() throws Exception {
    fakeDicomWebServer.addResponseWithStatusCode(404);
    assertThrows(
        DicomWebClient.DicomWebException.class,
        () -> client.stowRs("stowPath", new ByteArrayInputStream(new byte[0])));
  }
}
