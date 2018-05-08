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

import com.google.api.client.http.GenericUrl;
import com.google.api.client.http.HttpResponse;
import com.google.api.client.json.JsonObjectParser;
import com.google.api.client.util.Key;
import java.io.IOException;
import java.net.URI;
import java.util.Base64;
import java.util.logging.Level;
import java.util.logging.Logger;

public class HttpHl7Client extends HttpClient implements Hl7Client {
  private static final Logger LOGGER = Logger.getLogger(HttpHl7Client.class.getCanonicalName());

  /* Used for parsing HL7 messages. */
  public static class Hl7Message {
    @Key
    private String data;

    protected void setData(String data) {
      this.data = data;
    }

    // Returns base64 decoded content.
    protected byte[] getDecodedData() {
      return Base64.getDecoder().decode(data);
    }
  }

  protected final String apiEndpoint;

  public HttpHl7Client(String apiEndpoint) {
    super();
    this.apiEndpoint = apiEndpoint;
  }

  @Override
  public byte[] getMsg(String name) {
    GenericUrl url = new GenericUrl(URI.create(String.format("%s/%s", apiEndpoint, name)));
    try {
      HttpResponse response =
          httpRequestFactory
              .buildGetRequest(url)
              .setParser(new JsonObjectParser(JSON_FACTORY))
              .execute();
      Hl7Message msg = response.parseAs(Hl7Message.class);
      return msg.getDecodedData();
    } catch (IOException e) {
      LOGGER.log(Level.SEVERE, "Get message failed: " + name, e);
      throw new RuntimeException(e);
    }
  }
}
