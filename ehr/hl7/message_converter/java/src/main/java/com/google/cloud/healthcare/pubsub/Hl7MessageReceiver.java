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

package com.google.cloud.healthcare.pubsub;

import com.google.cloud.healthcare.apiclient.FhirClient;
import com.google.cloud.healthcare.apiclient.Hl7Client;
import com.google.cloud.healthcare.transform.MessageTransformer;
import com.google.cloud.pubsub.v1.AckReplyConsumer;
import com.google.cloud.pubsub.v1.MessageReceiver;
import com.google.pubsub.v1.PubsubMessage;
import java.util.logging.Level;
import java.util.logging.Logger;

/**
 * A customized message receiver that handles fetching the message, transforming it, and then upload
 * the resources.
 */
public class Hl7MessageReceiver implements MessageReceiver {
  private static final Logger LOGGER =
      Logger.getLogger(Hl7MessageReceiver.class.getCanonicalName());

  private final MessageTransformer transformer;
  private final Hl7Client hl7Client;
  private final FhirClient fhirClient;

  Hl7MessageReceiver(MessageTransformer transformer, Hl7Client hl7Client, FhirClient fhirClient) {
    this.transformer = transformer;
    this.hl7Client = hl7Client;
    this.fhirClient = fhirClient;
  }

  @Override
  public void receiveMessage(PubsubMessage message, AckReplyConsumer consumer) {
    String msgName = message.getData().toStringUtf8();
    try {
      byte[] body = hl7Client.getMsg(msgName);
      String bundle = transformer.transform(body);
      fhirClient.executeBundle(bundle);
    } catch (RuntimeException e) {
      LOGGER.log(Level.SEVERE, "Failed to handle message: " + msgName, e);
      // Return so that we can retry later.
      return;
    }

    consumer.ack();
  }
}
