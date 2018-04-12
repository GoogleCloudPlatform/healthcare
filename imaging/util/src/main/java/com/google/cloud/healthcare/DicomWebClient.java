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

import com.github.danieln.multipart.MultipartInput;
import com.google.api.client.http.GenericUrl;
import com.google.api.client.http.HttpRequest;
import com.google.api.client.http.HttpRequestFactory;
import com.google.api.client.http.HttpResponse;
import com.google.api.client.http.InputStreamContent;
import com.google.api.client.http.MultipartContent;
import com.google.common.io.CharStreams;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.util.UUID;
import javax.inject.Inject;
import org.json.JSONArray;

/** A client for communicating with the Cloud Healthcare API. */
public class DicomWebClient {
  /** An exception for errors returned by the DicomWeb server. */
  public static class DicomWebException extends Exception {
    public DicomWebException(String message) {
      super(message);
    }

    public DicomWebException(Throwable cause) {
      super(cause);
    }
  };

  // Factory to create HTTP requests with proper credentials.
  private final HttpRequestFactory requestFactory;

  // Service prefix all dicomWeb paths will be appended to.
  private final String serviceUrlPrefix;

  @Inject
  public DicomWebClient(
      HttpRequestFactory requestFactory,
      @Annotations.DicomwebAddr String serviceUrlPrefix) {
    this.requestFactory = requestFactory;
    this.serviceUrlPrefix = serviceUrlPrefix;
  }

  /** Makes a WADO-RS call and returns the multipart response. */
  public MultipartInput wadoRs(String path) throws DicomWebException {
    try {
      HttpRequest httpRequest =
          requestFactory.buildGetRequest(new GenericUrl(serviceUrlPrefix + "/" + path));
      HttpResponse httpResponse = httpRequest.execute();

      if (!httpResponse.isSuccessStatusCode()) {
        throw new DicomWebException(
            String.format(
                "WadoRs: %d, %s", httpResponse.getStatusCode(), httpResponse.getStatusMessage()));
      }
      return new MultipartInput(httpResponse.getContent(), httpResponse.getContentType());
    } catch (IOException | IllegalArgumentException e) {
      throw new DicomWebException(e);
    }
  }

  /** Makes a QIDO-RS call and returns a JSON array. */
  public JSONArray qidoRs(String path) throws DicomWebException {
    try {
      HttpRequest httpRequest =
          requestFactory.buildGetRequest(new GenericUrl(serviceUrlPrefix + "/" + path));
      HttpResponse httpResponse = httpRequest.execute();

      if (!httpResponse.isSuccessStatusCode()) {
        throw new DicomWebException(
            String.format(
                "QidoRs: %d, %s", httpResponse.getStatusCode(), httpResponse.getStatusMessage()));
      }
      return new JSONArray(
          CharStreams.toString(new InputStreamReader(httpResponse.getContent(), "UTF-8")));
    } catch (IOException | IllegalArgumentException e) {
      throw new DicomWebException(e);
    }
  }

  /**
   * Makes a STOW-RS call.
   *
   * @param path The resource path for the STOW-RS request.
   * @param in The DICOM input stream.
   */
  public void stowRs(String path, InputStream in) throws DicomWebException {
    GenericUrl url = new GenericUrl(serviceUrlPrefix + "/" + path);
    MultipartContent content = new MultipartContent();

    // DICOM "Type" parameter:
    // http://dicom.nema.org/medical/dicom/current/output/html/part18.html#sect_6.6.1.1.1
    content.setMediaType(content.getMediaType().setParameter("type", "\"application/dicom\""));
    content.setBoundary(UUID.randomUUID().toString());
    InputStreamContent dicomStream = new InputStreamContent("application/dicom", in);
    content.addPart(new MultipartContent.Part(dicomStream));

    HttpResponse httpResponse = null;
    try {
      HttpRequest httpRequest = requestFactory.buildPostRequest(url, content);
      httpResponse = httpRequest.execute();
    } catch (IOException e) {
      throw new DicomWebException(e);
    }

    if (!httpResponse.isSuccessStatusCode()) {
      throw new DicomWebException(
          String.format("StowRs: %d, %s.",
                        httpResponse.getStatusCode(), httpResponse.getStatusMessage()));
    }
  }
}
