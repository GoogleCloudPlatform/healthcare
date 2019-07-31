/*
 * Copyright 2019 Google LLC.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config_test

import (
	"testing"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/google/go-cmp/cmp"
	"github.com/ghodss/yaml"
)

func TestCHCDataset(t *testing.T) {
	datasetYAML := `
properties:
  location: us-central1
  datasetId: test_chc_dataset
  dicomStores:
  - dicomStoreId: test_chc_dicom_store
  fhirStores:
  - fhirStoreId: test_chc_fhir_store
  hl7V2Stores:
  - hl7V2StoreId: test_chc_hl7v2_store
`

	wantdatasetYAML := `
properties:
  location: us-central1
  datasetId: test_chc_dataset
  dicomStores:
  - dicomStoreId: test_chc_dicom_store
  fhirStores:
  - fhirStoreId: test_chc_fhir_store
  hl7V2Stores:
  - hl7V2StoreId: test_chc_hl7v2_store
`

	d := new(config.CHCDataset)
	if err := yaml.Unmarshal([]byte(datasetYAML), d); err != nil {
		t.Fatalf("yaml unmarshal: %v", err)
	}

	if err := d.Init(); err != nil {
		t.Fatalf("d.Init: %v", err)
	}

	got := make(map[string]interface{})
	want := make(map[string]interface{})
	b, err := yaml.Marshal(d)
	if err != nil {
		t.Fatalf("yaml.Marshal dataset: %v", err)
	}
	if err := yaml.Unmarshal(b, &got); err != nil {
		t.Fatalf("yaml.Unmarshal got config: %v", err)
	}
	if err := yaml.Unmarshal([]byte(wantdatasetYAML), &want); err != nil {
		t.Fatalf("yaml.Unmarshal want deployment config: %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("deployment yaml differs (-got +want):\n%v", diff)
	}

	if gotName, wantName := d.Name(), "test_chc_dataset"; gotName != wantName {
		t.Errorf("d.ResourceName() = %v, want %v", gotName, wantName)
	}
}
