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
import com.google.cloud.healthcare.apiclient.FhirClient.FhirStore;
import com.google.cloud.healthcare.apiclient.HttpFhirClient;
import com.google.cloud.healthcare.apiclient.HttpHl7V2Client;
import com.google.cloud.healthcare.transform.MessageTransformer;
import com.google.cloud.pubsub.v1.Subscriber;
import com.google.common.util.concurrent.MoreExecutors;
import com.google.pubsub.v1.ProjectSubscriptionName;
import java.util.logging.Level;
import java.util.logging.Logger;

/**
 * Listener for incoming HL7v2 messages notifications. It pulls HL7v2 messages, transforms them into FHIR
 * resources and uploads to a FHIR store.
 */
public class MessageListener {
  private static Logger LOGGER = Logger.getLogger(MessageListener.class.getCanonicalName());

  private final String apiAddrPrefix;
  private final String fhirProjectId;
  private final String fhirLocationId;
  private final String fhirDatasetId;
  private final String fhirStoreId;
  private final String pubsubProjectId;
  private final String pubsubSubscription;

  public MessageListener(
      String apiAddrPrefix,
      String fhirProjectId,
      String fhirLocationId,
      String fhirDatasetId,
      String fhirStoreId,
      String pubsubProjectId,
      String pubsubSubscription) {
    this.apiAddrPrefix = apiAddrPrefix;
    this.fhirProjectId = fhirProjectId;
    this.fhirLocationId = fhirLocationId;
    this.fhirDatasetId = fhirDatasetId;
    this.fhirStoreId = fhirStoreId;
    this.pubsubProjectId = pubsubProjectId;
    this.pubsubSubscription = pubsubSubscription;
  }

  public void start() {
    LOGGER.log(Level.INFO, "Starting listening for notifications...");

    ProjectSubscriptionName subscription =
        ProjectSubscriptionName.of(pubsubProjectId, pubsubSubscription);
    Subscriber subscriber = null;
    try {
      FhirClient fhirClient =
          new HttpFhirClient(
              apiAddrPrefix, fhirProjectId, fhirLocationId, fhirDatasetId, fhirStoreId);
      FhirStore fhirStore = fhirClient.getFhirStore();
      MessageTransformer transformer = new MessageTransformer(fhirStore.isUpdateCreateEnabled());

      subscriber =
          Subscriber.newBuilder(
                  subscription,
                  new Hl7V2MessageReceiver(
                      transformer,
                      new HttpHl7V2Client(apiAddrPrefix),
                      fhirClient))
              .build();
      subscriber.addListener(new SubscriberStateListener(), MoreExecutors.directExecutor());
      subscriber.startAsync().awaitRunning();

      try {
        // Prevent JVM from exiting.
        for (; ; ) {
          Thread.sleep(Long.MAX_VALUE);
        }
      } catch (InterruptedException e) {
      }
    } finally {
      if (subscriber != null) {
        subscriber.stopAsync().awaitTerminated();
      }
    }
  }
}
