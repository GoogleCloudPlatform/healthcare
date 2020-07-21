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
	"fmt"

	"github.com/GoogleCloudPlatform/healthcare/deploy/runner"
)

// ProjectIAMAuditConfig represents a terraform project iam audit config.
type ProjectIAMAuditConfig struct {
	Project         string            `json:"project"`
	Service         string            `json:"service"`
	AuditLogConfigs []*AuditLogConfig `json:"audit_log_config"`
}

// AuditLogConfig represents a terraform audit log config.
type AuditLogConfig struct {
	LogType string `json:"log_type"`
}

// Init initializes the resource.
func (c *ProjectIAMAuditConfig) Init(projectID string) error {
	if c.Project != "" {
		return fmt.Errorf("project must be unset: %v", c.Project)
	}
	c.Project = projectID
	return nil
}

// ID returns the resource unique identifier.
// It is hardcoded to return "project" as there is at most one of this resource in a deployment.
func (c *ProjectIAMAuditConfig) ID() string {
	return "project"
}

// ResourceType returns the resource terraform provider type.
func (c *ProjectIAMAuditConfig) ResourceType() string {
	return "google_project_iam_audit_config"
}

// ProjectIAMCustomRole represents a terraform project iam custom role.
type ProjectIAMCustomRole struct {
	RoleID  string `json:"role_id"`
	Project string `json:"project"`

	raw json.RawMessage
}

// Init initializes the resource.
func (r *ProjectIAMCustomRole) Init(projectID string) error {
	if r.Project != "" {
		return fmt.Errorf("project must be unset: %v", r.Project)
	}
	r.Project = projectID
	return nil
}

// ID returns the resource unique identifier.
// It is hardcoded to return "project" as there is at most one of this resource in a deployment.
func (r *ProjectIAMCustomRole) ID() string {
	return r.RoleID
}

// ResourceType returns the resource terraform provider type.
func (r *ProjectIAMCustomRole) ResourceType() string {
	return "google_project_iam_custom_role"
}

// ImportID returns the ID to use for terraform imports.
func (r *ProjectIAMCustomRole) ImportID(runner.Runner) (string, error) {
	return fmt.Sprintf("projects/%s/roles/%s", r.Project, r.RoleID), nil
}

// aliasProjectIAMCustomRole is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
type aliasProjectIAMCustomRole ProjectIAMCustomRole

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (r *ProjectIAMCustomRole) UnmarshalJSON(data []byte) error {
	var alias aliasProjectIAMCustomRole
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return fmt.Errorf("failed to unmarshal to parsed alias: %v", err)
	}
	*r = ProjectIAMCustomRole(alias)
	return nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (r *ProjectIAMCustomRole) MarshalJSON() ([]byte, error) {
	return interfacePair{r.raw, aliasProjectIAMCustomRole(*r)}.MarshalJSON()
}

// ProjectIAMMembers represents multiple Terraform project IAM members.
// It is used to wrap and merge multiple IAM members into a single IAM member when being marshalled to JSON.
type ProjectIAMMembers struct {
	Members   []*ProjectIAMMember
	DependsOn []string
	project   string
}

// ProjectIAMMember represents a Terraform project IAM member.
type ProjectIAMMember struct {
	Role   string `json:"role"`
	Member string `json:"member"`

	// The following fields should not be set by users.

	// ForEach is used to let a single iam member expand to reference multiple iam members
	// through the use of terraform's for_each iterator.
	ForEach   map[string]*ProjectIAMMember `json:"for_each,omitempty"`
	Project   string                       `json:"project,omitempty"`
	DependsOn []string                     `json:"depends_on,omitempty"`
}

// Init initializes the resource.
func (ms *ProjectIAMMembers) Init(projectID string) error {
	ms.project = projectID
	return nil
}

// ID returns the resource unique identifier.
// It is hardcoded to return "project" as there is at most one of this resource in a deployment.
func (ms *ProjectIAMMembers) ID() string {
	return "project"
}

// ResourceType returns the resource terraform provider type.
func (ms *ProjectIAMMembers) ResourceType() string {
	return "google_project_iam_member"
}

// MarshalJSON marshals the list of members into a single member.
// The single member will set a for_each block to expand to multiple iam members in the terraform call.
func (ms *ProjectIAMMembers) MarshalJSON() ([]byte, error) {
	forEach := make(map[string]*ProjectIAMMember)
	for _, m := range ms.Members {
		key := fmt.Sprintf("%s %s", m.Role, m.Member)
		forEach[key] = m
	}

	return json.Marshal(&ProjectIAMMember{
		ForEach:   forEach,
		Project:   ms.project,
		Role:      "${each.value.role}",
		Member:    "${each.value.member}",
		DependsOn: ms.DependsOn,
	})
}

// UnmarshalJSON unmarshals the bytes to a list of members.
func (ms *ProjectIAMMembers) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &ms.Members)
}

// ServiceAccount represents a Terraform service account.
type ServiceAccount struct {
	AccountID   string                     `json:"account_id"`
	Project     string                     `json:"project"`
	DisplayName string                     `json:"display_name"`
	IAMMembers  []*ServiceAccountIAMMember `json:"_iam_members"`

	raw json.RawMessage
}

// Init initializes the resource.
func (a *ServiceAccount) Init(projectID string) error {
	if a.Project != "" {
		return fmt.Errorf("project must not be set: %v", a.Project)
	}
	a.Project = projectID
	return nil
}

// ID returns the resource unique identifier.
func (a *ServiceAccount) ID() string {
	return a.AccountID
}

// ResourceType returns the resource terraform provider type.
func (a *ServiceAccount) ResourceType() string {
	return "google_service_account"
}

// ImportID returns the ID to use for terraform imports.
func (a *ServiceAccount) ImportID(runner.Runner) (string, error) {
	return fmt.Sprintf("projects/%s/serviceAccounts/%s@%s.iam.gserviceaccount.com", a.Project, a.AccountID, a.Project), nil
}

// DependentResources returns the child resources of this resource.
func (a *ServiceAccount) DependentResources() []Resource {
	if len(a.IAMMembers) == 0 {
		return nil
	}

	forEach := make(map[string]*ServiceAccountIAMMember)
	for _, m := range a.IAMMembers {
		key := fmt.Sprintf("%s %s", m.Role, m.Member)
		forEach[key] = m
	}
	return []Resource{&ServiceAccountIAMMember{
		ForEach:   forEach,
		AccountID: fmt.Sprintf("${google_service_account.%s.name}", a.ID()),
		Role:      "${each.value.role}",
		Member:    "${each.value.member}",
		id:        a.ID(),
	}}
}

// aliasStorageBucket is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
type aliasServiceAccount ServiceAccount

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (a *ServiceAccount) UnmarshalJSON(data []byte) error {
	var alias aliasServiceAccount
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return fmt.Errorf("failed to unmarshal to parsed alias: %v", err)
	}
	*a = ServiceAccount(alias)
	return nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (a *ServiceAccount) MarshalJSON() ([]byte, error) {
	return interfacePair{a.raw, aliasServiceAccount(*a)}.MarshalJSON()
}

// ServiceAccountIAMMember represents a Terraform GCS bucket IAM member.
type ServiceAccountIAMMember struct {
	Role   string `json:"role"`
	Member string `json:"member"`

	// The following fields should not be set by users.

	// ForEach is used to let a single iam member expand to reference multiple iam members
	// through the use of terraform's for_each iterator.
	ForEach map[string]*ServiceAccountIAMMember `json:"for_each,omitempty"`

	// Bucket should be written as a terraform reference to a bucket name so that it is created after the bucket.
	// e.g. ${google_ServiceAccount_bucket.foo_bucket.name}
	AccountID string `json:"service_account_id,omitempty"`

	// id should be the bucket's literal name.
	id string
}

// Init initializes the resource.
func (m *ServiceAccountIAMMember) Init(string) error {
	return nil
}

// ID returns the unique identifier.
func (m *ServiceAccountIAMMember) ID() string {
	return standardizeID(m.id)
}

// ResourceType returns the terraform provider type.
func (m *ServiceAccountIAMMember) ResourceType() string {
	return "google_service_account_iam_member"
}
