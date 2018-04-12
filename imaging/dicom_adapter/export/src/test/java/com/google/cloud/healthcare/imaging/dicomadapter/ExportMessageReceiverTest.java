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

import com.google.api.client.testing.http.HttpTesting;
import com.google.cloud.healthcare.DicomWebClient;
import com.google.cloud.healthcare.FakeWebServer;
import com.google.cloud.healthcare.TestUtils;
import com.google.cloud.pubsub.v1.AckReplyConsumer;
import com.google.protobuf.ByteString;
import com.google.pubsub.v1.PubsubMessage;
import org.dcm4che3.data.UID;
import org.dcm4che3.io.DicomInputStream;
import org.dcm4che3.net.ApplicationEntity;
import org.dcm4che3.net.Connection;
import org.dcm4che3.net.Device;
import org.dcm4che3.net.Status;
import org.dcm4che3.net.TransferCapability;
import org.dcm4che3.net.service.BasicCEchoSCP;
import org.dcm4che3.net.service.DicomServiceRegistry;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.junit.runners.JUnit4;

@RunWith(JUnit4.class)
public final class ExportMessageReceiverTest {
  private final String serverAET = "SERVER";
  private final String serverHost = "localhost";
  private final String clientAET = "CLIENT";
  private final String testPubsubPath = "/studies/1/series/2/instances/3";
  private final String testQidoResponse =
      "[{\"00020010\":{\"vr\":\"UI\",\"Value\":[\"1\"]},"
          + "\"00080016\":{\"vr\":\"UI\",\"Value\":[\"2\"]},"
          + "\"00080018\":{\"vr\":\"UI\",\"Value\":[\"3\"]}}]";
  private ApplicationEntity clientAE;
  private DicomInputStream dicom;

  @Before
  public void setUp() throws Exception {
    clientAE = new ApplicationEntity(clientAET);
    Connection conn = new Connection();
    DeviceUtil.createClientDevice(clientAE, conn);
    clientAE.addConnection(conn);

    dicom = new DicomInputStream(TestUtils.streamTestFile(TestUtils.TEST_MR_FILE));
  }

  private int createServerDevice(String sopClass, String transferSyntax, int wantResponseStatus)
      throws Exception {
    int serverPort = PortUtil.getFreePort();
    DicomServiceRegistry serviceRegistry = new DicomServiceRegistry();
    serviceRegistry.addDicomService(new BasicCEchoSCP());
    serviceRegistry.addDicomService(new StubCStoreService(wantResponseStatus));
    TransferCapability transferCapability =
        new TransferCapability(
            null /* commonName */, sopClass, TransferCapability.Role.SCP, transferSyntax);
    Device serverDevice =
        DeviceUtil.createServerDevice(serverAET, serverPort, serviceRegistry, transferCapability);
    serverDevice.bindConnections();
    return serverPort;
  }

  // StubAckReplyConsumer is a test helper used to check whether Pub/Sub message was ACK-ed
  // (representing a successful export).
  private class StubAckReplyConsumer implements AckReplyConsumer {
    private boolean isAck = false;

    @Override
    public void ack() {
      isAck = true;
    }

    @Override
    public void nack() {}

    public boolean isAck() {
      return isAck;
    }
  }

  // Creates an ExportMessageReceiver that uses C-STORE for use in unit-tests.
  ExportMessageReceiver testCStoreExportMessageReceiver(
      int serverPort, FakeWebServer fakeWebServer, StubAckReplyConsumer replyConsumer)
      throws Exception {
    ByteString pubsubMessageBytes = ByteString.copyFromUtf8(testPubsubPath);
    PubsubMessage pubsubMessage = PubsubMessage.newBuilder().setData(pubsubMessageBytes).build();
    DicomWebClient dicomWebClient =
        new DicomWebClient(fakeWebServer.createRequestFactory(), HttpTesting.SIMPLE_URL);
    DicomSender dicomSender =
        new CStoreSender(clientAE, serverAET, serverHost, serverPort, dicomWebClient);
    ExportMessageReceiver receiver = new ExportMessageReceiver(dicomSender);
    receiver.receiveMessage(pubsubMessage, replyConsumer);
    return receiver;
  }

  // Creates an ExportMessageReceiver that uses STOW-RS for use in unit-tests.
  ExportMessageReceiver testStowRsExportMessageReceiver(
      FakeWebServer fakeSourceDicomWebServer,
      FakeWebServer fakeSinkDicomWebServer,
      StubAckReplyConsumer replyConsumer)
      throws Exception {
    ByteString pubsubMessageBytes = ByteString.copyFromUtf8(testPubsubPath);
    PubsubMessage pubsubMessage = PubsubMessage.newBuilder().setData(pubsubMessageBytes).build();
    DicomWebClient sourceDicomWebClient =
        new DicomWebClient(fakeSourceDicomWebServer.createRequestFactory(), HttpTesting.SIMPLE_URL);
    DicomWebClient sinkDicomWebClient =
        new DicomWebClient(fakeSinkDicomWebServer.createRequestFactory(), HttpTesting.SIMPLE_URL);
    DicomSender dicomSender = new StowRsSender(sourceDicomWebClient, sinkDicomWebClient, "studies");
    ExportMessageReceiver receiver = new ExportMessageReceiver(dicomSender);
    receiver.receiveMessage(pubsubMessage, replyConsumer);
    return receiver;
  }

  @Test
  public void ExportMessageReceiverTest_CStoreSuccess() throws Exception {
    int serverPort = createServerDevice("*", "*", Status.Success);
    FakeWebServer fakeWebServer = new FakeWebServer();
    fakeWebServer.addJsonResponse(testQidoResponse);
    fakeWebServer.addWadoResponse(TestUtils.readTestFile(TestUtils.TEST_MR_FILE));

    StubAckReplyConsumer replyConsumer = new StubAckReplyConsumer();
    ExportMessageReceiver receiver =
        testCStoreExportMessageReceiver(serverPort, fakeWebServer, replyConsumer);
    assertThat(replyConsumer.isAck()).isTrue();
  }

  @Test
  public void ExportMessageReceiverTest_CStoreInvalidQidoResponse() throws Exception {
    int serverPort = createServerDevice("*", "*", Status.Success);
    FakeWebServer fakeWebServer = new FakeWebServer();
    fakeWebServer.addJsonResponse("[{}]");

    StubAckReplyConsumer replyConsumer = new StubAckReplyConsumer();
    ExportMessageReceiver receiver =
        testCStoreExportMessageReceiver(serverPort, fakeWebServer, replyConsumer);
    assertThat(replyConsumer.isAck()).isFalse();
  }

  @Test
  public void ExportMessageReceiverTest_CStoreDicomWebConnectionError() throws Exception {
    int serverPort = createServerDevice("*", "*", Status.Success);
    FakeWebServer fakeWebServer = new FakeWebServer();
    fakeWebServer.addResponseWithStatusCode(403);

    StubAckReplyConsumer replyConsumer = new StubAckReplyConsumer();
    ExportMessageReceiver receiver =
        testCStoreExportMessageReceiver(serverPort, fakeWebServer, replyConsumer);
    assertThat(replyConsumer.isAck()).isFalse();
  }

  @Test
  public void ExportMessageReceiverTest_CStoreAssociationRejected() throws Exception {
    int serverPort = createServerDevice(UID.MRImageStorage, "*", Status.Success);
    FakeWebServer fakeWebServer = new FakeWebServer();
    fakeWebServer.addJsonResponse(testQidoResponse);
    fakeWebServer.addWadoResponse(TestUtils.readTestFile(TestUtils.TEST_MR_FILE));

    StubAckReplyConsumer replyConsumer = new StubAckReplyConsumer();
    ExportMessageReceiver receiver =
        testCStoreExportMessageReceiver(serverPort, fakeWebServer, replyConsumer);
    assertThat(replyConsumer.isAck()).isFalse();
  }

  @Test
  public void ExportMessageReceiverTest_CStoreError() throws Exception {
    int serverPort = createServerDevice("*", "*", Status.ProcessingFailure);
    FakeWebServer fakeWebServer = new FakeWebServer();
    fakeWebServer.addJsonResponse(testQidoResponse);
    fakeWebServer.addWadoResponse(TestUtils.readTestFile(TestUtils.TEST_MR_FILE));

    StubAckReplyConsumer replyConsumer = new StubAckReplyConsumer();
    ExportMessageReceiver receiver =
        testCStoreExportMessageReceiver(serverPort, fakeWebServer, replyConsumer);
    assertThat(replyConsumer.isAck()).isFalse();
  }

  @Test
  public void ExportMessageReceiverTest_StowRsSuccess() throws Exception {
    // Service that sources DICOM.
    FakeWebServer fakeSourceDicomWebServer = new FakeWebServer();
    fakeSourceDicomWebServer.addWadoResponse(TestUtils.readTestFile(TestUtils.TEST_MR_FILE));

    // Service that sinks DICOM.
    FakeWebServer fakeSinkDicomWebServer = new FakeWebServer();
    fakeSinkDicomWebServer.addResponseWithStatusCode(200);

    StubAckReplyConsumer replyConsumer = new StubAckReplyConsumer();
    ExportMessageReceiver receiver =
        testStowRsExportMessageReceiver(
            fakeSourceDicomWebServer, fakeSinkDicomWebServer, replyConsumer);
    assertThat(replyConsumer.isAck()).isTrue();
  }

  @Test
  public void ExportMessageReceiverTest_StowRsSourceWadoRequestError() throws Exception {
    // Service that sources DICOM.
    FakeWebServer fakeSourceDicomWebServer = new FakeWebServer();
    fakeSourceDicomWebServer.addResponseWithStatusCode(404);

    // Service that sinks DICOM.
    FakeWebServer fakeSinkDicomWebServer = new FakeWebServer();

    StubAckReplyConsumer replyConsumer = new StubAckReplyConsumer();
    ExportMessageReceiver receiver =
        testStowRsExportMessageReceiver(
            fakeSourceDicomWebServer, fakeSinkDicomWebServer, replyConsumer);
    assertThat(replyConsumer.isAck()).isFalse();
  }

  @Test
  public void ExportMessageReceiverTest_StowRsSinkStowRequestError() throws Exception {
    // Service that sources DICOM.
    FakeWebServer fakeSourceDicomWebServer = new FakeWebServer();
    fakeSourceDicomWebServer.addWadoResponse(TestUtils.readTestFile(TestUtils.TEST_MR_FILE));

    // Service that sinks DICOM.
    FakeWebServer fakeSinkDicomWebServer = new FakeWebServer();
    fakeSourceDicomWebServer.addResponseWithStatusCode(400);

    StubAckReplyConsumer replyConsumer = new StubAckReplyConsumer();
    ExportMessageReceiver receiver =
        testStowRsExportMessageReceiver(
            fakeSourceDicomWebServer, fakeSinkDicomWebServer, replyConsumer);
    assertThat(replyConsumer.isAck()).isFalse();
  }
}
