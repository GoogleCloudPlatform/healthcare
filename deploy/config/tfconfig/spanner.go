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

// SpannerInstance represents a Terraform spanner instance.
type SpannerInstance struct {
	Name    string `json:"name"`
	Project string `json:"project"`

	IAMMembers []*SpannerInstanceIAMMember `json:"_iam_members"`
	Databases  []*SpannerDatabase          `json:"_databases"`

	raw json.RawMessage
}

// Init initializes the resource.
func (i *SpannerInstance) Init(projectID string) error {
	if i.Name == "" {
		return errors.New("name must be set")
	}
	if i.Project != "" {
		return fmt.Errorf("project must be unset: %v", i.Project)
	}
	i.Project = projectID

	for _, d := range i.Databases {
		d.Instance = fmt.Sprintf("${google_spanner_instance.%s.name}", i.ID())
		d.instanceLiteral = i.ID()
		if err := d.Init(projectID); err != nil {
			return fmt.Errorf("failed to init database %q: %v", d.Name, err)
		}
	}
	return nil
}

// ID returns the resource unique identifier.
func (i *SpannerInstance) ID() string {
	return i.Name
}

// ResourceType returns the resource terraform provider type.
func (*SpannerInstance) ResourceType() string {
	return "google_spanner_instance"
}

// DependentResources returns the child resources of this resource.
func (i *SpannerInstance) DependentResources() []Resource {
	var res []Resource

	for _, d := range i.Databases {
		res = append(res, d)
	}

	if len(i.IAMMembers) == 0 {
		return nil
	}

	forEach := make(map[string]*SpannerInstanceIAMMember)
	for _, m := range i.IAMMembers {
		key := fmt.Sprintf("%s %s", m.Role, m.Member)
		forEach[key] = m
	}
	res = append(res, &SpannerInstanceIAMMember{
		ForEach:  forEach,
		Instance: fmt.Sprintf("${google_spanner_instance.%s.name}", i.ID()),
		Role:     "${each.value.role}",
		Member:   "${each.value.member}",
		id:       i.Name,
	})

	return res
}

// ImportID returns the ID to use for terraform imports.
func (i *SpannerInstance) ImportID(runner.Runner) (string, error) {
	return fmt.Sprintf("projects/%s/instances/%s", i.Project, i.ID()), nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (i *SpannerInstance) MarshalJSON() ([]byte, error) {
	return interfacePair{i.raw, aliasSpannerInstance(*i)}.MarshalJSON()
}

// aliasSpannerInstance is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
type aliasSpannerInstance SpannerInstance

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (i *SpannerInstance) UnmarshalJSON(data []byte) error {
	var alias aliasSpannerInstance
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return fmt.Errorf("failed to unmarshal to parsed alias: %v", err)
	}
	*i = SpannerInstance(alias)
	return nil
}

// SpannerInstanceIAMMember represents a Terraform spanner instance iam member.
type SpannerInstanceIAMMember struct {
	Role   string `json:"role"`
	Member string `json:"member"`

	// ForEach is used to let a single iam member expand to reference multiple iam members
	// through the use of terraform's for_each iterator.
	ForEach map[string]*SpannerInstanceIAMMember `json:"for_each,omitempty"`

	// Instance should be written as a terraform reference to an instance name so that it is created after the instance.
	// e.g. ${google_spanner_instance.instance.name}
	Instance string `json:"instance,omitempty"`

	// id should be the instance's literal name.
	id string
}

// Init initializes the resource.
func (m *SpannerInstanceIAMMember) Init(string) error {
	return nil
}

// ID returns the unique identifier.
func (m *SpannerInstanceIAMMember) ID() string {
	return m.id
}

// ResourceType returns the terraform provider type.
func (m *SpannerInstanceIAMMember) ResourceType() string {
	return "google_spanner_instance_iam_member"
}

// SpannerDatabase represents a Terraform GCS bucket.
type SpannerDatabase struct {
	Name    string `json:"name"`
	Project string `json:"project"`

	Instance string `json:"instance"`

	IAMMembers []*SpannerDatabaseIAMMember `json:"_iam_members"`

	instanceLiteral string
	raw             json.RawMessage
}

// Init initializes the resource.
func (d *SpannerDatabase) Init(projectID string) error {
	if d.Name == "" {
		return errors.New("name must be set")
	}
	if d.Instance == "" {
		return errors.New("instance must be set")
	}
	if d.Project != "" {
		return fmt.Errorf("project must be unset: %v", d.Project)
	}
	d.Project = projectID
	return nil
}

// ID returns the resource unique identifier.
func (d *SpannerDatabase) ID() string {
	return d.Name
}

// ResourceType returns the resource terraform provider type.
func (*SpannerDatabase) ResourceType() string {
	return "google_spanner_database"
}

// DependentResources returns the child resources of this resource.
func (d *SpannerDatabase) DependentResources() []Resource {
	if len(d.IAMMembers) == 0 {
		return nil
	}

	forEach := make(map[string]*SpannerDatabaseIAMMember)
	for _, m := range d.IAMMembers {
		key := fmt.Sprintf("%s %s", m.Role, m.Member)
		forEach[key] = m
	}
	return []Resource{&SpannerDatabaseIAMMember{
		ForEach:  forEach,
		Database: fmt.Sprintf("${google_spanner_database.%s.name}", d.ID()),
		Instance: d.instanceLiteral,
		Role:     "${each.value.role}",
		Member:   "${each.value.member}",
		id:       d.Name,
	}}
}

// ImportID returns the ID to use for terraform imports.
func (d *SpannerDatabase) ImportID(runner.Runner) (string, error) {
	return fmt.Sprintf("projects/%s/instances/%s/databases/%s", d.Project, d.instanceLiteral, d.ID()), nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (d *SpannerDatabase) MarshalJSON() ([]byte, error) {
	return interfacePair{d.raw, aliasSpannerDatabase(*d)}.MarshalJSON()
}

// aliasSpannerDatabase is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
type aliasSpannerDatabase SpannerDatabase

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (d *SpannerDatabase) UnmarshalJSON(data []byte) error {
	var alias aliasSpannerDatabase
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return fmt.Errorf("failed to unmarshal to parsed alias: %v", err)
	}
	*d = SpannerDatabase(alias)
	return nil
}

// SpannerDatabaseIAMMember represents a Terraform spanner database iam member.
type SpannerDatabaseIAMMember struct {
	Role   string `json:"role"`
	Member string `json:"member"`

	// ForEach is used to let a single iam member expand to reference multiple iam members
	// through the use of terraform's for_each iterator.
	ForEach map[string]*SpannerDatabaseIAMMember `json:"for_each,omitempty"`

	// Database should be written as a terraform reference to an database name so that it is created after the database.
	// e.g. ${google_spanner_database.database.name}
	Database string `json:"database,omitempty"`

	// Instance should be a literal instance id.
	Instance string `json:"instance,omitempty"`

	// id should be the database's literal name.
	id string
}

// Init initializes the resource.
func (m *SpannerDatabaseIAMMember) Init(string) error {
	return nil
}

// ID returns the unique identifier.
func (m *SpannerDatabaseIAMMember) ID() string {
	return fmt.Sprintf("%s_%s", m.Instance, m.id)
}

// ResourceType returns the terraform provider type.
func (m *SpannerDatabaseIAMMember) ResourceType() string {
	return "google_spanner_database_iam_member"
}
