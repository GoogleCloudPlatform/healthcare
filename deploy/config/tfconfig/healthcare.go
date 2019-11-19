// Copyright 2019 Google LLC
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

package tfconfig

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/GoogleCloudPlatform/healthcare/deploy/runner"
)

// HealthcareDataset represents a terraform healthcare dataset.
type HealthcareDataset struct {
	Name     string `json:"name"`
	Project  string `json:"project"`
	Provider string `json:"provider,omitempty"`
	Location string `json:"location"`

	IAMMembers []*HealthcareDatasetIAMMember `json:"_iam_members"`

	DICOMStores []*HealthcareDICOMStore `json:"_dicom_stores"`
	FHIRStores  []*HealthcareFHIRStore  `json:"_fhir_stores"`
	HL7V2Stores []*HealthcareHL7V2Store `json:"_hl7_v2_stores"`

	raw json.RawMessage
}

// Init initializes the resource.
func (d *HealthcareDataset) Init(projectID string) error {
	if d.Name == "" {
		return errors.New("name must be set")
	}
	if d.Location == "" {
		return errors.New("location must be set")
	}
	d.Project = projectID
	d.Provider = "google-beta"

	ref := fmt.Sprintf("${google_healthcare_dataset.%s.id}", d.ID())
	for _, s := range d.DICOMStores {
		if err := s.Init(projectID); err != nil {
			return fmt.Errorf("failed to init dicom store %q: %v", s.Name, err)
		}
		s.Dataset = ref
		s.id = fmt.Sprintf("%s_%s", d.Name, s.Name)
	}
	for _, s := range d.FHIRStores {
		if err := s.Init(projectID); err != nil {
			return fmt.Errorf("failed to init fhir store %q: %v", s.Name, err)
		}
		s.Dataset = ref
		s.id = fmt.Sprintf("%s_%s", d.Name, s.Name)
	}
	for _, s := range d.HL7V2Stores {
		if err := s.Init(projectID); err != nil {
			return fmt.Errorf("failed to init hl7 v2 store %q: %v", s.Name, err)
		}
		s.Dataset = ref
		s.id = fmt.Sprintf("%s_%s", d.Name, s.Name)
	}
	return nil
}

// ID returns the resource unique identifier.
func (d *HealthcareDataset) ID() string {
	return d.Name
}

// ResourceType returns the resource terraform provider type.
func (*HealthcareDataset) ResourceType() string {
	return "google_healthcare_dataset"
}

// ImportID returns the ID to use for terraform imports.
func (d *HealthcareDataset) ImportID(runner.Runner) (string, error) {
	return fmt.Sprintf("projects/%s/locations/%s/datasets/%s", d.Project, d.Location, d.Name), nil
}

// DependentResources returns the child resources of this resource.
func (d *HealthcareDataset) DependentResources() []Resource {
	var rs []Resource
	if len(d.IAMMembers) > 0 {
		forEach := make(map[string]*HealthcareDatasetIAMMember)
		for _, m := range d.IAMMembers {
			key := fmt.Sprintf("%s %s", m.Role, m.Member)
			forEach[key] = m
		}
		rs = append(rs, &HealthcareDatasetIAMMember{
			ForEach:   forEach,
			DatasetID: fmt.Sprintf("${google_healthcare_dataset.%s.id}", d.ID()),
			Role:      "${each.value.role}",
			Member:    "${each.value.member}",
			Provider:  "google-beta",
			id:        d.ID(),
		})
	}

	for _, s := range d.DICOMStores {
		rs = append(rs, s)
	}
	for _, s := range d.FHIRStores {
		rs = append(rs, s)
	}
	for _, s := range d.HL7V2Stores {
		rs = append(rs, s)
	}
	return rs
}

// aliasHealthcareDataset is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
type aliasHealthcareDataset HealthcareDataset

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (d *HealthcareDataset) UnmarshalJSON(data []byte) error {
	var alias aliasHealthcareDataset
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return fmt.Errorf("failed to unmarshal to parsed alias: %v", err)
	}
	*d = HealthcareDataset(alias)
	return nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (d *HealthcareDataset) MarshalJSON() ([]byte, error) {
	return interfacePair{d.raw, aliasHealthcareDataset(*d)}.MarshalJSON()
}

// HealthcareDatasetIAMMember represents a Terraform GCS bucket IAM member.
type HealthcareDatasetIAMMember struct {
	Role     string `json:"role"`
	Member   string `json:"member"`
	Provider string `json:"provider,omitempty"`

	// The following fields should not be set by users.

	// ForEach is used to let a single iam member expand to reference multiple iam members
	// through the use of terraform's for_each iterator.
	ForEach map[string]*HealthcareDatasetIAMMember `json:"for_each,omitempty"`

	// DatsetID should be written as a terraform reference to a dataset to create an implicit dependency.
	DatasetID string `json:"dataset_id,omitempty"`

	// id should be the dataset's literal name.
	id string
}

// Init initializes the resource.
func (m *HealthcareDatasetIAMMember) Init(string) error {
	return nil
}

// ID returns the unique identifier.
func (m *HealthcareDatasetIAMMember) ID() string {
	return m.id
}

// ResourceType returns the terraform provider type.
func (m *HealthcareDatasetIAMMember) ResourceType() string {
	return "google_healthcare_dataset_iam_member"
}

// HealthcareDICOMStore represents a terraform DICOM store.
type HealthcareDICOMStore struct {
	Name     string `json:"name"`
	Dataset  string `json:"dataset"`
	Provider string `json:"provider,omitempty"`

	IAMMembers []*HealthcareDICOMStoreIAMMember `json:"_iam_members"`

	// id should be a literal unique name to use as the terraform resource name.
	id  string
	raw json.RawMessage
}

// Init initializes the resource.
func (s *HealthcareDICOMStore) Init(string) error {
	if s.Name == "" {
		return errors.New("name must be set")
	}
	s.Provider = "google-beta"
	return nil
}

// ID returns the resource unique identifier.
func (s *HealthcareDICOMStore) ID() string {
	return s.id
}

// ResourceType returns the resource terraform provider type.
func (*HealthcareDICOMStore) ResourceType() string {
	return "google_healthcare_dicom_store"
}

// DependentResources returns the child resources of this resource.
func (s *HealthcareDICOMStore) DependentResources() []Resource {
	if len(s.IAMMembers) == 0 {
		return nil
	}

	forEach := make(map[string]*HealthcareDICOMStoreIAMMember)
	for _, m := range s.IAMMembers {
		key := fmt.Sprintf("%s %s", m.Role, m.Member)
		forEach[key] = m
	}
	return []Resource{&HealthcareDICOMStoreIAMMember{
		ForEach:      forEach,
		DICOMStoreID: fmt.Sprintf("${google_healthcare_dicom_store.%s.id}", s.ID()),
		Role:         "${each.value.role}",
		Member:       "${each.value.member}",
		Provider:     "google-beta",
		id:           s.ID(),
	}}
}

// aliasHealthcareDataset is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
type aliasHealthcareDICOMStore HealthcareDICOMStore

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (s *HealthcareDICOMStore) UnmarshalJSON(data []byte) error {
	var alias aliasHealthcareDICOMStore
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return fmt.Errorf("failed to unmarshal to parsed alias: %v", err)
	}
	*s = HealthcareDICOMStore(alias)
	return nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (s *HealthcareDICOMStore) MarshalJSON() ([]byte, error) {
	return interfacePair{s.raw, aliasHealthcareDICOMStore(*s)}.MarshalJSON()
}

// HealthcareDICOMStoreIAMMember represents a terraform DICOM store IAM member.
type HealthcareDICOMStoreIAMMember struct {
	Role     string `json:"role"`
	Member   string `json:"member"`
	Provider string `json:"provider,omitempty"`

	// The following fields should not be set by users.

	// ForEach is used to let a single iam member expand to reference multiple iam members
	// through the use of terraform's for_each iterator.
	ForEach map[string]*HealthcareDICOMStoreIAMMember `json:"for_each,omitempty"`

	// DICOMStoreID should be written as a terraform reference to a DICOM store to create an implicit dependency.
	DICOMStoreID string `json:"dicom_store_id,omitempty"`

	// id should be the dataset's literal name.
	id string
}

// Init initializes the resource.
func (m *HealthcareDICOMStoreIAMMember) Init(string) error {
	return nil
}

// ID returns the unique identifier.
func (m *HealthcareDICOMStoreIAMMember) ID() string {
	return m.id
}

// ResourceType returns the terraform provider type.
func (m *HealthcareDICOMStoreIAMMember) ResourceType() string {
	return "google_healthcare_dicom_store_iam_member"
}

// HealthcareFHIRStore represents a terraform FHIR store.
type HealthcareFHIRStore struct {
	Name     string `json:"name"`
	Dataset  string `json:"dataset"`
	Provider string `json:"provider,omitempty"`

	IAMMembers []*HealthcareFHIRStoreIAMMember `json:"_iam_members"`

	// id should be a literal unique name to use as the terraform resource name.
	id  string
	raw json.RawMessage
}

// Init initializes the resource.
func (s *HealthcareFHIRStore) Init(string) error {
	if s.Name == "" {
		return errors.New("name must be set")
	}
	s.Provider = "google-beta"
	return nil
}

// ID returns the resource unique identifier.
func (s *HealthcareFHIRStore) ID() string {
	return s.id
}

// ResourceType returns the resource terraform provider type.
func (*HealthcareFHIRStore) ResourceType() string {
	return "google_healthcare_fhir_store"
}

// DependentResources returns the child resources of this resource.
func (s *HealthcareFHIRStore) DependentResources() []Resource {
	if len(s.IAMMembers) == 0 {
		return nil
	}

	forEach := make(map[string]*HealthcareFHIRStoreIAMMember)
	for _, m := range s.IAMMembers {
		key := fmt.Sprintf("%s %s", m.Role, m.Member)
		forEach[key] = m
	}
	return []Resource{&HealthcareFHIRStoreIAMMember{
		ForEach:     forEach,
		FHIRStoreID: fmt.Sprintf("${google_healthcare_fhir_store.%s.id}", s.ID()),
		Role:        "${each.value.role}",
		Member:      "${each.value.member}",
		Provider:    "google-beta",
		id:          s.ID(),
	}}
}

// aliasHealthcareDataset is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
type aliasHealthcareFHIRStore HealthcareFHIRStore

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (s *HealthcareFHIRStore) UnmarshalJSON(data []byte) error {
	var alias aliasHealthcareFHIRStore
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return fmt.Errorf("failed to unmarshal to parsed alias: %v", err)
	}
	*s = HealthcareFHIRStore(alias)
	return nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (s *HealthcareFHIRStore) MarshalJSON() ([]byte, error) {
	return interfacePair{s.raw, aliasHealthcareFHIRStore(*s)}.MarshalJSON()
}

// HealthcareFHIRStoreIAMMember represents a terraform FHIR store IAM member.
type HealthcareFHIRStoreIAMMember struct {
	Role     string `json:"role"`
	Member   string `json:"member"`
	Provider string `json:"provider,omitempty"`

	// The following fields should not be set by users.

	// ForEach is used to let a single iam member expand to reference multiple iam members
	// through the use of terraform's for_each iterator.
	ForEach map[string]*HealthcareFHIRStoreIAMMember `json:"for_each,omitempty"`

	// FHIRStoreID should be written as a terraform reference to a FHIR store to create an implicit dependency.
	FHIRStoreID string `json:"fhir_store_id,omitempty"`

	// id should be a literal unique name to use as the terraform resource name.
	id string
}

// Init initializes the resource.
func (m *HealthcareFHIRStoreIAMMember) Init(string) error {
	return nil
}

// ID returns the unique identifier.
func (m *HealthcareFHIRStoreIAMMember) ID() string {
	return m.id
}

// ResourceType returns the terraform provider type.
func (m *HealthcareFHIRStoreIAMMember) ResourceType() string {
	return "google_healthcare_fhir_store_iam_member"
}

// HealthcareHL7V2Store represents a terraform HL7V2 store.
type HealthcareHL7V2Store struct {
	Name     string `json:"name"`
	Dataset  string `json:"dataset"`
	Provider string `json:"provider,omitempty"`

	IAMMembers []*HealthcareHL7V2StoreIAMMember `json:"_iam_members"`

	// id should be a literal unique name to use as the terraform resource name.
	id  string
	raw json.RawMessage
}

// Init initializes the resource.
func (s *HealthcareHL7V2Store) Init(string) error {
	if s.Name == "" {
		return errors.New("name must be set")
	}
	s.Provider = "google-beta"
	return nil
}

// ID returns the resource unique identifier.
func (s *HealthcareHL7V2Store) ID() string {
	return s.id
}

// ResourceType returns the resource terraform provider type.
func (*HealthcareHL7V2Store) ResourceType() string {
	return "google_healthcare_hl7_v2_store"
}

// DependentResources returns the child resources of this resource.
func (s *HealthcareHL7V2Store) DependentResources() []Resource {
	if len(s.IAMMembers) == 0 {
		return nil
	}

	forEach := make(map[string]*HealthcareHL7V2StoreIAMMember)
	for _, m := range s.IAMMembers {
		key := fmt.Sprintf("%s %s", m.Role, m.Member)
		forEach[key] = m
	}
	return []Resource{&HealthcareHL7V2StoreIAMMember{
		ForEach:      forEach,
		HL7V2StoreID: fmt.Sprintf("${google_healthcare_hl7_v2_store.%s.id}", s.ID()),
		Role:         "${each.value.role}",
		Member:       "${each.value.member}",
		Provider:     "google-beta",
		id:           s.ID(),
	}}
}

// aliasHealthcareDataset is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
type aliasHealthcareHL7V2Store HealthcareHL7V2Store

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (s *HealthcareHL7V2Store) UnmarshalJSON(data []byte) error {
	var alias aliasHealthcareHL7V2Store
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return fmt.Errorf("failed to unmarshal to parsed alias: %v", err)
	}
	*s = HealthcareHL7V2Store(alias)
	return nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (s *HealthcareHL7V2Store) MarshalJSON() ([]byte, error) {
	return interfacePair{s.raw, aliasHealthcareHL7V2Store(*s)}.MarshalJSON()
}

// HealthcareHL7V2StoreIAMMember represents a terraform HL7V2 store IAM member.
type HealthcareHL7V2StoreIAMMember struct {
	Role     string `json:"role"`
	Member   string `json:"member"`
	Provider string `json:"provider,omitempty"`

	// The following fields should not be set by users.

	// ForEach is used to let a single iam member expand to reference multiple iam members
	// through the use of terraform's for_each iterator.
	ForEach map[string]*HealthcareHL7V2StoreIAMMember `json:"for_each,omitempty"`

	// HL7V2StoreID should be written as a terraform reference to a HL7V2 store to create an implicit dependency.
	HL7V2StoreID string `json:"hl7_v2_store_id,omitempty"`

	// id should be a literal unique name to use as the terraform resource name.
	id string
}

// Init initializes the resource.
func (m *HealthcareHL7V2StoreIAMMember) Init(string) error {
	return nil
}

// ID returns the unique identifier.
func (m *HealthcareHL7V2StoreIAMMember) ID() string {
	return m.id
}

// ResourceType returns the terraform provider type.
func (m *HealthcareHL7V2StoreIAMMember) ResourceType() string {
	return "google_healthcare_hl7_v2_store_iam_member"
}
