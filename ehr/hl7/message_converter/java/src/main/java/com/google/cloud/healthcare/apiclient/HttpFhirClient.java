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

import com.google.api.client.http.ByteArrayContent;
import com.google.api.client.http.GenericUrl;
import com.google.api.client.http.HttpHeaders;
import com.google.api.client.http.HttpResponse;
import com.google.api.client.json.JsonObjectParser;
import com.google.cloud.healthcare.apiclient.HttpHl7Client.Hl7Message;
import com.google.common.io.CharStreams;
import java.io.IOException;
import java.io.InputStreamReader;
import java.net.URI;
import java.util.logging.Level;
import java.util.logging.Logger;

public class HttpFhirClient extends HttpClient implements FhirClient {
  private static final Logger LOGGER = Logger.getLogger(HttpFhirClient.class.getCanonicalName());

  private static final String FHIR_JSON_CONTENT_TYPE = "application/fhir+json;charset=utf-8";

  private final String apiEndpoint;
  private final String projectId;
  private final String locationId;
  private final String datasetId;
  private final String fhirStoreId;

  public HttpFhirClient(
      String apiEndpoint,
      String projectId,
      String locationId,
      String datasetId,
      String fhirStoreId) {
    super();
    this.apiEndpoint = apiEndpoint;
    this.projectId = projectId;
    this.locationId = locationId;
    this.datasetId = datasetId;
    this.fhirStoreId = fhirStoreId;
  }

  @Override
  public String executeBundle(String bundle) {
    GenericUrl url = generateFhirStoreUrl();
    try {
      HttpResponse response =
          httpRequestFactory
              .buildPostRequest(
                  url, new ByteArrayContent(FHIR_JSON_CONTENT_TYPE, bundle.getBytes()))
              .setHeaders(new HttpHeaders().setAccept(FHIR_JSON_CONTENT_TYPE))
              .execute();
      return CharStreams.toString(new InputStreamReader(response.getContent(), "UTF-8"));
    } catch (IOException e) {
      LOGGER.log(Level.SEVERE, "Post resources failed.", e);
      throw new RuntimeException(e);
    }
  }

  @Override
  public FhirStore getFhirStore() {
    GenericUrl url = generateFhirStoreUrl();
    try {
      HttpResponse response =
          httpRequestFactory
              .buildGetRequest(url)
              .setParser(new JsonObjectParser(JSON_FACTORY))
              .execute();
      return response.parseAs(FhirStore.class);
    } catch (IOException e) {
      LOGGER.log(Level.SEVERE, "Unable to get FHIR store information");
      throw new RuntimeException(e);
    }
  }

  private GenericUrl generateFhirStoreUrl() {
    return new GenericUrl(
            URI.create(
                String.format(
                    "%s/projects/%s/locations/%s/datasets/%s/fhirStores/%s",
                    apiEndpoint, projectId, locationId, datasetId, fhirStoreId)));
  }
}
