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

import com.github.danieln.multipart.MultipartInput;
import com.github.danieln.multipart.PartInput;
import com.google.cloud.healthcare.DicomWebClient;
import com.google.pubsub.v1.PubsubMessage;
import java.nio.file.Path;
import java.nio.file.Paths;
import org.dcm4che3.data.Tag;
import org.dcm4che3.net.ApplicationEntity;
import org.dcm4che3.net.Association;
import org.dcm4che3.net.FutureDimseRSP;
import org.dcm4che3.net.pdu.PresentationContext;
import org.dcm4che3.util.TagUtils;
import org.json.JSONArray;
import org.json.JSONException;
import org.json.JSONObject;

// CStoreSender sends DICOM to peer using DIMSE C-STORE protocol.
public class CStoreSender implements DicomSender {
  private final ApplicationEntity applicationEntity;
  private final String dimsePeerAET;
  private final String dimsePeerIP;
  private final int dimsePeerPort;
  private final DicomWebClient dicomWebClient;

  private static final String TRANSFER_SYNTAX_UID_TAG = TagUtils.toHexString(Tag.TransferSyntaxUID);
  private static final String SOP_CLASS_UID_TAG = TagUtils.toHexString(Tag.SOPClassUID);
  private static final String SOP_INSTANCE_UID_TAG = TagUtils.toHexString(Tag.SOPInstanceUID);
  private static final int SUCCESS_DIMSE_STATUS_CODE = 0;

  CStoreSender(
      ApplicationEntity applicationEntity,
      String dimsePeerAET,
      String dimsePeerIP,
      int dimsePeerPort,
      DicomWebClient dicomWebClient) {
    this.applicationEntity = applicationEntity;
    this.dimsePeerAET = dimsePeerAET;
    this.dimsePeerIP = dimsePeerIP;
    this.dimsePeerPort = dimsePeerPort;
    this.dicomWebClient = dicomWebClient;
  }

  @Override
  public void send(PubsubMessage message) throws Exception {
    String wadoUri = message.getData().toStringUtf8();
    String qidoUri = qidoFromWadoUri(wadoUri);

    // Invoke QIDO-RS to get DICOM tags needed to invoke C-Store.
    JSONArray qidoResponse = dicomWebClient.qidoRs(qidoUri);
    if (qidoResponse.length() != 1) {
      throw new IllegalArgumentException(
          "Invalid QidoRS JSON array length for response: " + qidoResponse.toString());
    }
    String transferSyntaxUid = getTagValue(qidoResponse.getJSONObject(0), TRANSFER_SYNTAX_UID_TAG);
    String sopClassUid = getTagValue(qidoResponse.getJSONObject(0), SOP_CLASS_UID_TAG);
    String sopInstanceUid = getTagValue(qidoResponse.getJSONObject(0), SOP_INSTANCE_UID_TAG);

    // Invoke WADO-RS to get bulk DICOM.
    MultipartInput resp = dicomWebClient.wadoRs(wadoUri);
    PartInput part = resp.nextPart();
    if (part == null) {
      throw new IllegalArgumentException("WadoRS response has no parts");
    }

    // Invoke a C-Store to send DICOM.
    PresentationContext pc = new PresentationContext(1, sopClassUid, transferSyntaxUid);
    DicomClient dicomClient =
        DicomClient.associatePeer(applicationEntity, dimsePeerAET, dimsePeerIP, dimsePeerPort, pc);
    Association association = dicomClient.getAssociation();
    FutureDimseRSP handler = new FutureDimseRSP(association.nextMessageID());
    dicomClient.cstore(
        sopClassUid, sopInstanceUid, part.getInputStream(), transferSyntaxUid, handler);
    handler.next();
    int dimseStatus = handler.getCommand().getInt(Tag.Status, /* default status */ -1);
    if (dimseStatus != SUCCESS_DIMSE_STATUS_CODE) {
      throw new IllegalArgumentException("C-STORE failed with status code: " + dimseStatus);
    }
    try {
      association.release();
      association.waitForSocketClose();
    } catch (Exception e) {
      System.err.println("Send C-STORE successfully, but failed to close association");
      e.printStackTrace();
    }
  }

  // Derives a QIDO-RS path using a WADO-RS path.
  // TODO(b/72555677): Find an easier way to do this instead of string manipulation.
  private String qidoFromWadoUri(String wadoUri) {
    Path wadoPath = Paths.get(wadoUri);
    String instanceUid = wadoPath.getFileName().toString();
    Path wadoParentPath = wadoPath.getParent();
    String qidoUri =
        String.format(
            "%s?%s=%s&includefield=%s",
            wadoParentPath.toString(), SOP_INSTANCE_UID_TAG, instanceUid, TRANSFER_SYNTAX_UID_TAG);
    return qidoUri;
  }

  // Gets a DICOM tag from given JSONObject (a QIDO response).
  private String getTagValue(JSONObject json, String tag) throws JSONException {
    JSONObject jsonTag = json.getJSONObject(tag);
    JSONArray valueArray = jsonTag.getJSONArray("Value");
    if (valueArray.length() != 1) {
      throw new JSONException(
          "Expected one value in QIDO response for tag: "
              + tag
              + " in JSON:\n"
              + valueArray.toString());
    }
    return valueArray.getString(0);
  }
}
