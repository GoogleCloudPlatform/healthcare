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

package com.google.cloud.healthcare.hl7v2.common;

import org.apache.beam.runners.dataflow.options.DataflowPipelineOptions;
import org.apache.beam.sdk.options.Default;
import org.apache.beam.sdk.options.Description;

public interface HL7V2toHL7V2Options extends DataflowPipelineOptions {

  @Description("The subscription ID for the source hl7V2Store subscription")
  @Default.String("projects/example-project-id/subscriptions/source-subscription")
  String getSourceSubscription();

  void setSourceSubscription(String sourceTopic);

  @Description("Full name of the destination hl7V2Store")
  @Default.String(
      "projects/example-dest-project-id/locations/us-central1/datasets/destination-dataset/hl7V2Stores/destination-store")
  String getDestinationHL7V2Store();

  void setDestinationHL7V2Store(String destinationHL7V2Store);
}
