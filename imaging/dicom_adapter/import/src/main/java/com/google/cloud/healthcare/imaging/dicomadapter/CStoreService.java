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

import com.google.api.client.http.GenericUrl;
import com.google.api.client.http.HttpMediaType;
import com.google.api.client.http.HttpRequest;
import com.google.api.client.http.HttpRequestFactory;
import com.google.api.client.http.HttpResponse;
import com.google.api.client.http.InputStreamContent;
import com.google.api.client.http.MultipartContent;
import java.io.IOException;
import java.io.InputStream;
import java.util.UUID;
import org.dcm4che3.data.Attributes;
import org.dcm4che3.data.Tag;
import org.dcm4che3.data.VR;
import org.dcm4che3.net.Association;
import org.dcm4che3.net.PDVInputStream;
import org.dcm4che3.net.Status;
import org.dcm4che3.net.pdu.PresentationContext;
import org.dcm4che3.net.service.BasicCStoreSCP;
import org.dcm4che3.net.service.DicomServiceException;

/** Class to handle server-side C-STORE DICOM requests. */
public class CStoreService extends BasicCStoreSCP {
  private static final int C_STORE_SUCCESS_STATUS = 0;

  private String apiAddr;
  private String path;
  private HttpRequestFactory requestFactory;

  CStoreService(String apiAddr, String path, HttpRequestFactory requestFactory) {
    this.apiAddr = apiAddr;
    this.path = path;
    this.requestFactory = requestFactory;
  }

  @Override
  protected void store(
      Association association,
      PresentationContext presentationContext,
      Attributes request,
      PDVInputStream inDicomStream,
      Attributes response)
      throws DicomServiceException, IOException {
    String sopClassUID = request.getString(Tag.AffectedSOPClassUID);
    String sopInstanceUID = request.getString(Tag.AffectedSOPInstanceUID);
    String transferSyntax = presentationContext.getTransferSyntax();
    String remoteAeTitle = association.getCallingAET();

    InputStream inBuffer =
        DicomStreamUtil.dicomStreamWithFileMetaHeader(
            sopInstanceUID, sopClassUID, transferSyntax, inDicomStream);

    MultipartContent content = new MultipartContent();
    content.setMediaType(new HttpMediaType("multipart/related; type=\"application/dicom\""));
    content.setBoundary(UUID.randomUUID().toString());
    InputStreamContent dicomStream = new InputStreamContent("application/dicom", inBuffer);
    content.addPart(new MultipartContent.Part(dicomStream));

    HttpResponse httpResponse = null;
    GenericUrl url = new GenericUrl(apiAddr + path);
    try {
      HttpRequest httpRequest = requestFactory.buildPostRequest(url, content);
      httpResponse = httpRequest.execute();
    } catch (IOException e) {
      e.printStackTrace();
      throw new DicomServiceException(Status.ProcessingFailure, e);
    }

    if (!httpResponse.isSuccessStatusCode()) {
      String errorMessage = String.format(
              "Got error response code for STOW-RS POST: %d, %s",
              httpResponse.getStatusCode(), httpResponse.getStatusMessage());
      System.err.println(errorMessage);
      throw new DicomServiceException(Status.ProcessingFailure, errorMessage);
    }

    System.out.printf(
        "Received C-STORE for association %s, SOP class %s, TS %s, remote AE %s\n",
        association.toString(), sopClassUID, transferSyntax, remoteAeTitle);
    response.setInt(Tag.Status, VR.US, C_STORE_SUCCESS_STATUS);
  }

  @Override
  public void onClose(Association association) {
    // Handle any exceptions that may have been caused by aborts or C-Store request processing.
    String associationName = association.toString();
    if (association.getException() != null) {
      System.err.printf("Exception while handling association %s.\n", associationName);
      association.getException().printStackTrace();
    } else {
      System.out.printf("Association %s finished successfully.\n", associationName);
    }
  }
}
