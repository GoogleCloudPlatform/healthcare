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

package com.google.cloud.healthcare.apiclient;

import com.google.api.client.auth.oauth2.BearerToken;
import com.google.api.client.auth.oauth2.Credential;
import com.google.api.client.http.HttpRequestFactory;
import com.google.api.client.http.HttpTransport;
import com.google.api.client.http.javanet.NetHttpTransport;
import com.google.api.client.json.JsonFactory;
import com.google.api.client.json.gson.GsonFactory;
import com.google.auth.oauth2.AccessToken;
import com.google.auth.oauth2.GoogleCredentials;
import java.io.IOException;
import java.util.Collections;
import java.util.logging.Level;
import java.util.logging.Logger;

/** A base HTTP client with common setup, e.g. credentials. */
abstract class HttpClient {
  private static final Logger LOGGER = Logger.getLogger(HttpClient.class.getCanonicalName());

  private static final String HEALTHCARE_SCOPE = "https://www.googleapis.com/auth/cloud-healthcare";

  protected static final HttpTransport HTTP_TRANSPORT = new NetHttpTransport();
  protected static final JsonFactory JSON_FACTORY = new GsonFactory();

  protected HttpRequestFactory httpRequestFactory;
  protected final GoogleCredentials credentials;

  HttpClient() {
    try {
      credentials =
          GoogleCredentials.getApplicationDefault()
              .createScoped(Collections.singleton(HEALTHCARE_SCOPE));
      refreshCredential();
    } catch (IOException e) {
      LOGGER.log(Level.SEVERE, "Unable to obtain credentials.", e);
      throw new RuntimeException(e);
    }
  }

  protected void refreshCredential() {
    try {
      initHttpRequestFactory(credentials.refreshAccessToken());
    } catch (IOException e) {
      LOGGER.log(Level.SEVERE, "Unable to refresh credentials.", e);
      throw new RuntimeException(e);
    }
  }

  private void initHttpRequestFactory(AccessToken accessToken) {
    Credential credential =
        new Credential(BearerToken.authorizationHeaderAccessMethod())
            .setAccessToken(accessToken.getTokenValue());
    httpRequestFactory = HTTP_TRANSPORT.createRequestFactory(credential);
  }
}
