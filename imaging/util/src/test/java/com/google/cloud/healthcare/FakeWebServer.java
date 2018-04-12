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

import com.google.api.client.http.ByteArrayContent;
import com.google.api.client.http.LowLevelHttpRequest;
import com.google.api.client.http.LowLevelHttpResponse;
import com.google.api.client.http.MultipartContent;
import com.google.api.client.testing.http.MockHttpTransport;
import com.google.api.client.testing.http.MockLowLevelHttpRequest;
import com.google.api.client.testing.http.MockLowLevelHttpResponse;
import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.util.LinkedList;
import java.util.Queue;
import java.util.Vector;

/** A fake in-memory implementation of a web server. */
public class FakeWebServer extends MockHttpTransport {
  public static class Request {
    public Request(String method, MockLowLevelHttpRequest req) {
      this.method = method;
      this.request = req;
    }

    public String method;
    public MockLowLevelHttpRequest request;
  }

  private Queue<LowLevelHttpResponse> responses = new LinkedList<LowLevelHttpResponse>();
  private Vector<Request> requests = new Vector<Request>();

  public FakeWebServer() {}

  @Override
  public LowLevelHttpRequest buildRequest(String method, String url) throws IOException {
    MockLowLevelHttpRequest request = new MockLowLevelHttpRequest(url) {
      @Override
      public LowLevelHttpResponse execute() throws IOException {
        MockLowLevelHttpResponse response = new MockLowLevelHttpResponse();
        if (responses.isEmpty()) {
          throw new IOException(
              "Unexpected call to execute(), no injected responses left to return.");
        }
        return responses.remove();
      }
    };
    requests.add(new Request(method, request));
    return request;
  }

  public void addWadoResponse(byte[] dicomInstance) throws IOException {
    MultipartContent mpc = new MultipartContent();
    mpc.addPart(
        new MultipartContent.Part(new ByteArrayContent("application/dicom", dicomInstance)));

    ByteArrayOutputStream contentOutputStream = new ByteArrayOutputStream();
    mpc.writeTo(contentOutputStream);

    MockLowLevelHttpResponse response = new MockLowLevelHttpResponse();
    response.setStatusCode(200);
    response.setContentType(String.format("multipart/related; boundary=\"%s\"", mpc.getBoundary()));
    response.setContent(contentOutputStream.toByteArray());
    responses.add(response);
  }

  public void addJsonResponse(String jsonResponse) {
    MockLowLevelHttpResponse response = new MockLowLevelHttpResponse();
    response.setStatusCode(200);
    response.setContentType("application/json");
    response.setContent(new ByteArrayInputStream(jsonResponse.getBytes()));
    responses.add(response);
  }

  public void addResponseWithStatusCode(int statusCode) {
    MockLowLevelHttpResponse response = new MockLowLevelHttpResponse();
    response.setStatusCode(statusCode);
    responses.add(response);
  }

  public Vector<Request> getRequests() {
    return requests;
  }

}
