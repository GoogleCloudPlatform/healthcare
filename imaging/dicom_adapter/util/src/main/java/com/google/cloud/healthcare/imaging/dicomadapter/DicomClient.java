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

import java.io.IOException;
import java.io.InputStream;
import org.dcm4che3.io.DicomInputStream;
import org.dcm4che3.net.ApplicationEntity;
import org.dcm4che3.net.Association;
import org.dcm4che3.net.Connection;
import org.dcm4che3.net.DimseRSPHandler;
import org.dcm4che3.net.InputStreamDataWriter;
import org.dcm4che3.net.pdu.AAssociateRQ;
import org.dcm4che3.net.pdu.PresentationContext;

/** DicomClient is used to handle client-side legacy DICOM requests on given association. */
public class DicomClient {
  private Association association;

  public DicomClient(Association association) {
    this.association = association;
  }

  /** Creates a new DicomClient by creating an association to the given peer. */
  public static DicomClient associatePeer(
      ApplicationEntity clientAE,
      String peerAET,
      String peerHostname,
      int peerPort,
      PresentationContext pc)
      throws Exception {
    AAssociateRQ rq = new AAssociateRQ();
    rq.addPresentationContext(pc);
    rq.setCalledAET(peerAET);
    Connection remoteConn = new Connection();
    remoteConn.setHostname(peerHostname);
    remoteConn.setPort(peerPort);
    Association association = clientAE.connect(remoteConn, rq);
    return new DicomClient(association);
  }

  public void cstore(
      String sopClassUID,
      String sopInstanceUID,
      InputStream in,
      String transferSyntaxUID,
      DimseRSPHandler responseHandler)
      throws IOException, InterruptedException {
    DicomInputStream din = new DicomInputStream(in);
    din.readFileMetaInformation();
    InputStreamDataWriter data = new InputStreamDataWriter(din);
    association.cstore(
        sopClassUID, sopInstanceUID, /* priority */ 1, data, transferSyntaxUID, responseHandler);
  }

  public Association getAssociation() {
    return association;
  }
}
