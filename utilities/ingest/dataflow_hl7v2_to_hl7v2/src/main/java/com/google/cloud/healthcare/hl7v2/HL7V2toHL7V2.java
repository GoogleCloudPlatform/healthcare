// Copyright 2024 Google LLC
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

package com.google.cloud.healthcare.hl7v2;

import com.google.cloud.healthcare.hl7v2.common.HL7V2toHL7V2Options;
import com.google.common.base.Strings;
import org.apache.beam.sdk.Pipeline;
import org.apache.beam.sdk.io.gcp.healthcare.HL7v2IO;
import org.apache.beam.sdk.io.gcp.pubsub.PubsubIO;
import org.apache.beam.sdk.options.PipelineOptionsFactory;

public class HL7V2toHL7V2 {

  static void runHL7V2Pipeline(HL7V2toHL7V2Options options) {
    boolean hasSourceSubscription = !Strings.isNullOrEmpty(options.getSourceSubscription());
    boolean hasDestinationStore = !Strings.isNullOrEmpty(options.getDestinationHL7V2Store());

    if (!hasSourceSubscription && !hasDestinationStore) {
      throw new IllegalArgumentException(
          "Must specify both a source subscription and a destination HL7V2 store");
    }

    Pipeline p = Pipeline.create(options);
    HL7v2IO.Read.Result v2Messages =
        p.apply(
                "Read from Subscription",
                PubsubIO.readStrings().fromSubscription(options.getSourceSubscription()))
            .apply("Read from HL7V2Store", HL7v2IO.getAll());
    // TODO: Handle errors in the result
    v2Messages
        .getMessages()
        .apply("Write to HL7V2Store", HL7v2IO.ingestMessages(options.getDestinationHL7V2Store()));

    p.run();
  }

  public static void main(String[] args) {
    HL7V2toHL7V2Options options =
        PipelineOptionsFactory.fromArgs(args).withValidation().as(HL7V2toHL7V2Options.class);
    runHL7V2Pipeline(options);
  }
}
