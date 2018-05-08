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

import com.google.cloud.healthcare.pubsub.MessageListener;
import com.google.devtools.common.options.Option;
import com.google.devtools.common.options.OptionsBase;
import com.google.devtools.common.options.OptionsParser;
import com.google.common.base.Strings;
import java.util.Collections;

/** Entrance class. */
public class Hl7ToFhirWorker {
  public static class WorkerOptions extends OptionsBase {
    @Option(name = "help", help = "Print this message.", defaultValue = "false")
    public boolean help;

    @Option(
        name = "apiAddrPrefix",
        help = "API endpoint URL prefix, including the URI scheme and API version.",
        abbrev = 'a',
        defaultValue = "")
    public String apiAddrPrefix;

    @Option(
        name = "fhirProjectId",
        help = "FHIR project ID, used to construct the FHIR URL.",
        abbrev = 'p',
        defaultValue = "")
    public String fhirProjectId;

    @Option(
        name = "fhirLocationId",
        help = "FHIR location ID, used to construct the FHIR URL. Defaults to us-central1.",
        abbrev = 'l',
        defaultValue = "us-central1")
    public String fhirLocationId;

    @Option(
        name = "fhirDatasetId",
        help = "FHIR dataset ID, used to construct the FHIR URL.",
        abbrev = 'd',
        defaultValue = "")
    public String fhirDatasetId;

    @Option(
        name = "fhirStoreId",
        help = "FHIR store ID, used to construct the FHIR URL.",
        abbrev = 's',
        defaultValue = "")
    public String fhirStoreId;

    @Option(
        name = "pubsubProjectId",
        help =
            "Project ID under which the pubsub is set up, this can be the same as HL7 or FHIR project ID.",
        abbrev = 'r',
        defaultValue = "")
    public String pubsubProjectId;

    @Option(
        name = "pubsubSubscription",
        help =
            "Pubsub subscription name without the leading prefix, e.g. test rather than project/my-project/subscriptions/test",
        abbrev = 'u',
        defaultValue = "")
    public String pubsubSubscription;

    public void printUsage(OptionsParser parser) {
      System.out.println("Usage: java -jar converter.jar OPTIONS");
      System.out.println(
          parser.describeOptions(
              Collections.<String, String>emptyMap(), OptionsParser.HelpVerbosity.LONG));
    }
  }

  public static void main(String[] args) {
    OptionsParser parser = OptionsParser.newOptionsParser(WorkerOptions.class);
    parser.parseAndExitUponError(args);
    WorkerOptions options = parser.getOptions(WorkerOptions.class);

    if (options.help) {
      options.printUsage(parser);
      return;
    }

    if (Strings.isNullOrEmpty(options.apiAddrPrefix)
        || Strings.isNullOrEmpty(options.fhirProjectId)
        || Strings.isNullOrEmpty(options.fhirDatasetId)
        || Strings.isNullOrEmpty(options.fhirStoreId)
        || Strings.isNullOrEmpty(options.pubsubProjectId)
        || Strings.isNullOrEmpty(options.pubsubSubscription)) {
      options.printUsage(parser);
      return;
    }

    new MessageListener(
            options.apiAddrPrefix,
            options.fhirProjectId,
            options.fhirLocationId,
            options.fhirDatasetId,
            options.fhirStoreId,
            options.pubsubProjectId,
            options.pubsubSubscription)
        .start();
  }
}
