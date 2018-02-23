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

package com.google.cloud.healthcare.imaging.dicomadapter;

import static com.google.common.truth.Truth.assertThat;

import com.google.api.client.http.HttpRequestFactory;
import com.google.api.client.http.HttpStatusCodes;
import com.google.api.client.http.LowLevelHttpRequest;
import com.google.api.client.http.LowLevelHttpResponse;
import com.google.api.client.testing.http.MockHttpTransport;
import com.google.api.client.testing.http.MockLowLevelHttpRequest;
import com.google.api.client.testing.http.MockLowLevelHttpResponse;
import com.google.cloud.healthcare.TestUtils;
import java.io.IOException;
import java.io.InputStream;
import java.util.concurrent.Executors;
import org.dcm4che3.data.Attributes;
import org.dcm4che3.data.Tag;
import org.dcm4che3.data.UID;
import org.dcm4che3.io.DicomInputStream;
import org.dcm4che3.net.ApplicationEntity;
import org.dcm4che3.net.Association;
import org.dcm4che3.net.Connection;
import org.dcm4che3.net.Device;
import org.dcm4che3.net.DimseRSPHandler;
import org.dcm4che3.net.InputStreamDataWriter;
import org.dcm4che3.net.Status;
import org.dcm4che3.net.pdu.AAssociateRQ;
import org.dcm4che3.net.pdu.PresentationContext;
import org.dcm4che3.net.service.BasicCEchoSCP;
import org.dcm4che3.net.service.DicomServiceRegistry;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.junit.runners.JUnit4;

@RunWith(JUnit4.class)
public final class CStoreServiceTest {
  // Server properties.
  String serverAET = "SERVER";
  String serverHostname = "localhost";

  // Client properties.
  ApplicationEntity clientAE;

  // Creates the client-side C-STORE response handler.
  private static DimseRSPHandler createDimseRSPHandler(Association association, int wantStatus) {
    return new DimseRSPHandler(association.nextMessageID()) {
      @Override
      public void onDimseRSP(Association association, Attributes cmd, Attributes data) {
        super.onDimseRSP(association, cmd, data);
        int gotStatus = cmd.getInt(Tag.Status, /* default status */ -1);
        assertThat(gotStatus).isEqualTo(wantStatus);
      }
    };
  }

  // Creates a HTTP request factory that returns given response code.
  private HttpRequestFactory createHttpRequestFactory(boolean connectError, int responseCode) {
    return new MockHttpTransport() {
      @Override
      public LowLevelHttpRequest buildRequest(String method, String url) {
        return new MockLowLevelHttpRequest() {
          @Override
          public LowLevelHttpResponse execute() throws IOException {
            if (connectError) {
              throw new IOException("connect error");
            }
            MockLowLevelHttpResponse response = new MockLowLevelHttpResponse();
            response.setStatusCode(responseCode);
            return response;
          }
        };
      }
    }.createRequestFactory();
  }

  private Association associate(
      String serverHostname, int serverPort, String sopClass, String syntax) throws Exception {
    AAssociateRQ rq = new AAssociateRQ();
    rq.addPresentationContext(new PresentationContext(1, sopClass, syntax));
    rq.setCalledAET(serverAET);
    Connection remoteConn = new Connection();
    remoteConn.setHostname(serverHostname);
    remoteConn.setPort(serverPort);
    return clientAE.connect(remoteConn, rq);
  }

  // Creates a DICOM service and returns the port it is listening on.
  private int createDicomServer(boolean connectError, int responseCode) throws Exception {
    int serverPort = PortUtil.getFreePort();
    DicomServiceRegistry serviceRegistry = new DicomServiceRegistry();
    serviceRegistry.addDicomService(new BasicCEchoSCP());
    HttpRequestFactory httpRequestFactory = createHttpRequestFactory(connectError, responseCode);
    CStoreService cStoreService =
        new CStoreService("https://localhost:443", "/studies", httpRequestFactory);
    serviceRegistry.addDicomService(cStoreService);
    Device serverDevice = DeviceUtil.createServerDevice(serverAET, serverPort, serviceRegistry);
    serverDevice.bindConnections();
    return serverPort;
  }

  @Before
  public void setUp() throws Exception {
    // Create the C-STORE client.
    String clientAET = "CSTORECLIENT";
    Device device = new Device(clientAET);
    Connection conn = new Connection();
    device.addConnection(conn);
    clientAE = new ApplicationEntity(clientAET);
    device.addApplicationEntity(clientAE);
    clientAE.addConnection(conn);
    device.setExecutor(Executors.newSingleThreadExecutor());
    device.setScheduledExecutor(Executors.newSingleThreadScheduledExecutor());
  }

  @Test
  public void testCStoreService_basicCStore() throws Exception {
    InputStream in =
        new DicomInputStream(TestUtils.streamDICOMStripHeaders(TestUtils.TEST_MR_FILE));
    InputStreamDataWriter data = new InputStreamDataWriter(in);

    // Properties of the DICOM image.
    String sopClassUID = UID.MRImageStorage;
    String sopInstanceUID = "1.0.0.0";

    // Create C-STORE DICOM server.
    int serverPort = createDicomServer(/*connect error*/ false, HttpStatusCodes.STATUS_CODE_OK);

    // Associate with peer AE.
    Association association =
        associate(serverHostname, serverPort, sopClassUID, UID.ExplicitVRLittleEndian);

    // Send the DICOM file.
    int wantStatus = Status.Success;
    association.cstore(
        sopClassUID,
        sopInstanceUID,
        1,
        data,
        UID.ExplicitVRLittleEndian,
        createDimseRSPHandler(association, wantStatus));
    association.waitForOutstandingRSP();

    // Close the association.
    association.release();
    association.waitForSocketClose();
  }

  @Test
  public void testCStoreService_connectionError() throws Exception {
    InputStream in =
        new DicomInputStream(TestUtils.streamDICOMStripHeaders(TestUtils.TEST_MR_FILE));
    InputStreamDataWriter data = new InputStreamDataWriter(in);

    // Properties of the DICOM image.
    String sopClassUID = UID.MRImageStorage;
    String sopInstanceUID = "1.0.0.0";

    // Create C-STORE DICOM server.
    // The server is unable to get the response for the STOW-RS post (e.g. connection error).
    int serverPort =
        createDicomServer(/*connect error*/ true, HttpStatusCodes.STATUS_CODE_UNAUTHORIZED);

    // Associate with peer AE.
    Association association =
        associate(serverHostname, serverPort, sopClassUID, UID.ExplicitVRLittleEndian);

    // Send the DICOM file.
    int wantStatus = Status.ProcessingFailure;
    association.cstore(
        sopClassUID,
        sopInstanceUID,
        1,
        data,
        UID.ExplicitVRLittleEndian,
        createDimseRSPHandler(association, wantStatus));
    association.waitForOutstandingRSP();

    // Close the association.
    association.release();
    association.waitForSocketClose();
  }

  @Test
  public void testCStoreService_basicCStoreUnauthorized() throws Exception {
    InputStream in =
        new DicomInputStream(TestUtils.streamDICOMStripHeaders(TestUtils.TEST_MR_FILE));
    InputStreamDataWriter data = new InputStreamDataWriter(in);

    // Properties of the DICOM image.
    String sopClassUID = UID.MRImageStorage;
    String sopInstanceUID = "1.0.0.0";

    // Create C-STORE DICOM server.
    // The server gets an unauthorized error upon invoking STOW-RS POST (which for now causes a
    // client error).
    int serverPort =
        createDicomServer(/*connect error*/ false, HttpStatusCodes.STATUS_CODE_UNAUTHORIZED);

    // Associate with peer AE.
    Association association =
        associate(serverHostname, serverPort, sopClassUID, UID.ExplicitVRLittleEndian);

    // Send the DICOM file.
    int wantStatus = Status.ProcessingFailure;
    association.cstore(
        sopClassUID,
        sopInstanceUID,
        1,
        data,
        UID.ExplicitVRLittleEndian,
        createDimseRSPHandler(association, wantStatus));
    association.waitForOutstandingRSP();

    // Close the association.
    association.release();
    association.waitForSocketClose();
  }

  // TODO(b/73252285): increase test coverage.
}
