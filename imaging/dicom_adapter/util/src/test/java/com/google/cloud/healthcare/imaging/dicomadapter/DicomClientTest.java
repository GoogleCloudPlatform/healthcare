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

import com.google.cloud.healthcare.TestUtils;
import org.dcm4che3.data.Attributes;
import org.dcm4che3.data.Tag;
import org.dcm4che3.data.UID;
import org.dcm4che3.io.DicomInputStream;
import org.dcm4che3.net.ApplicationEntity;
import org.dcm4che3.net.Association;
import org.dcm4che3.net.Connection;
import org.dcm4che3.net.Device;
import org.dcm4che3.net.DimseRSPHandler;
import org.dcm4che3.net.NoPresentationContextException;
import org.dcm4che3.net.Status;
import org.dcm4che3.net.TransferCapability;
import org.dcm4che3.net.pdu.PresentationContext;
import org.dcm4che3.net.service.BasicCEchoSCP;
import org.dcm4che3.net.service.DicomServiceRegistry;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.junit.runners.JUnit4;

@RunWith(JUnit4.class)
public final class DicomClientTest {
  private final String serverAET = "SERVER";
  private final String serverHost = "localhost";

  private final String clientAET = "CLIENT";
  private ApplicationEntity clientAE;

  private String sopInstanceUID = "1.0.0.0";

  private DicomInputStream dicom;

  @Before
  public void setUp() throws Exception {
    clientAE = new ApplicationEntity(clientAET);
    Connection conn = new Connection();
    DeviceUtil.createClientDevice(clientAE, conn);
    clientAE.addConnection(conn);

    dicom = new DicomInputStream(TestUtils.streamTestFile(TestUtils.TEST_MR_FILE));
  }

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

  @Test
  public void testDicomClient_successStatus() throws Exception {
    int serverPort = createServerDevice("*", "*", Status.Success);
    String sopClass = UID.MRImageStorage;
    String transferSyntax = UID.ExplicitVRLittleEndian;
    PresentationContext pc = new PresentationContext(1, sopClass, transferSyntax);
    DicomClient client = DicomClient.associatePeer(clientAE, serverAET, serverHost, serverPort, pc);
    Association association = client.getAssociation();
    client.cstore(
        sopClass,
        sopInstanceUID,
        dicom,
        transferSyntax,
        createDimseRSPHandler(association, Status.Success));
    association.waitForOutstandingRSP();
    association.release();
    association.waitForSocketClose();
  }

  @Test
  public void testDicomClient_failureStatus() throws Exception {
    int serverPort = createServerDevice("*", "*", Status.ProcessingFailure);
    String sopClass = UID.MRImageStorage;
    String transferSyntax = UID.ExplicitVRLittleEndian;
    PresentationContext pc = new PresentationContext(1, sopClass, transferSyntax);
    DicomClient client = DicomClient.associatePeer(clientAE, serverAET, serverHost, serverPort, pc);
    Association association = client.getAssociation();
    client.cstore(
        sopClass,
        sopInstanceUID,
        dicom,
        transferSyntax,
        createDimseRSPHandler(association, Status.ProcessingFailure));
    association.waitForOutstandingRSP();
    association.release();
    association.waitForSocketClose();
  }

  @Test(expected = NoPresentationContextException.class)
  public void testDicomClient_abstractSyntaxNotNegotiated() throws Exception {
    // CT SOP class is not negotiated - cstore is rejected.
    int serverPort = createServerDevice(UID.MRImageStorage, "*", Status.Success);
    String sopClass = UID.CTImageStorage;
    String transferSyntax = UID.ExplicitVRLittleEndian;
    PresentationContext pc = new PresentationContext(1, sopClass, transferSyntax);
    DicomClient client = DicomClient.associatePeer(clientAE, serverAET, serverHost, serverPort, pc);
    Association association = client.getAssociation();
    client.cstore(
        sopClass,
        sopInstanceUID,
        dicom,
        transferSyntax,
        createDimseRSPHandler(association, Status.Success));
  }

  @Test(expected = NoPresentationContextException.class)
  public void testDicomClient_transferSyntaxNotNegotiated() throws Exception {
    // Explicit VR Little Endian transfer syntax is not negotiated - cstore is rejected.
    int serverPort = createServerDevice("*", UID.ImplicitVRLittleEndian, Status.Success);
    String sopClass = UID.MRImageStorage;
    String transferSyntax = UID.ExplicitVRLittleEndian;
    PresentationContext pc = new PresentationContext(1, sopClass, transferSyntax);
    DicomClient client = DicomClient.associatePeer(clientAE, serverAET, serverHost, serverPort, pc);
    Association association = client.getAssociation();
    client.cstore(
        sopClass,
        sopInstanceUID,
        dicom,
        transferSyntax,
        createDimseRSPHandler(association, Status.Success));
  }
}
