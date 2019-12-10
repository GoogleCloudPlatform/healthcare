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
)

// CloudBuildTrigger represents a Terraform Cloudbuild Trigger.
type CloudBuildTrigger struct {
	Name          string                 `json:"name"`
	Project       string                 `json:"project"`
	GitHub        *GitHub                `json:"github"`
	Description   string                 `json:"description,omitempty"`
	Disabled      bool                   `json:"disabled,omitempty"`
	Substitutions map[string]interface{} `json:"substitutions,omitempty"`
	Filename      string                 `json:"filename,omitempty"`
	IgnoredFiles  []string               `json:"ignored_files,omitempty"`
	IncludedFiles []string               `json:"included_files,omitempty"`

	raw json.RawMessage
}

// GitHub represents a github block in Terraform Cloudbuild Trigger.
type GitHub struct {
	Owner       string       `json:"owner"`
	Name        string       `json:"name"`
	PullRequest *PullRequest `json:"pull_request,omitempty"`
	Push        *Push        `json:"push,omitempty"`
}

// PullRequest represents a pull_request block in github block.
type PullRequest struct {
	Branch         string `json:"branch"`
	CommentControl string `json:"comment_control,omitempty"`
}

// Push represents a push block used in github block.
type Push struct {
	Branch string `json:"branch,omitempty"`
	Tag    string `json:"tag,omitempty"`
}

// Init initializes the resource.
func (t *CloudBuildTrigger) Init(projectID string) error {
	if t.Name == "" {
		return errors.New("name must be set")
	}
	g := t.GitHub
	if g == nil {
		return errors.New("github must be set")
	}
	if g.Owner == "" {
		return errors.New("owner in github must be set")
	}
	if g.Name == "" {
		return errors.New("name in github must be set")
	}
	if (g.PullRequest != nil) == (g.Push != nil) {
		return errors.New("exactly one of pull_request or push in github must be set")
	}
	if g.PullRequest != nil {
		if g.PullRequest.Branch == "" {
			return errors.New("branch in pull_request must be set")
		}
		if cc := g.PullRequest.CommentControl; cc != "" && cc != "COMMENTS_DISABLED" && cc != "COMMENTS_ENABLED" {
			return errors.New("value of comment_control in pull_request can only be one of [COMMENTS_DISABLED, COMMENTS_ENABLED]")
		}
	}
	if g.Push != nil && (g.Push.Branch == "") == (g.Push.Tag == "") {
		return errors.New("exactly one of branch or tag in push must be set")
	}
	if t.Project != "" {
		return fmt.Errorf("project must be unset: %v", t.Project)
	}
	t.Project = projectID
	return nil
}

// ID returns the resource unique identifier.
func (t *CloudBuildTrigger) ID() string {
	return t.Name
}

// ResourceType returns the resource terraform provider type.
func (t *CloudBuildTrigger) ResourceType() string {
	return "google_cloudbuild_trigger"
}

// // ImportID returns the ID to use for terraform imports.
// func (t *CloudBuildTrigger) ImportID(runner.Runner) (string, error) {
// 	TODO: obtain triggerID
// 	return fmt.Sprintf("projects/%s/triggers/%s", t.Project, triggerID), nil
// }

// aliasCloudBuildTrigger is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
type aliasCloudBuildTrigger CloudBuildTrigger

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (t *CloudBuildTrigger) UnmarshalJSON(data []byte) error {
	var alias aliasCloudBuildTrigger
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return fmt.Errorf("failed to unmarshal to parsed alias: %v", err)
	}
	*t = CloudBuildTrigger(alias)
	return nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (t *CloudBuildTrigger) MarshalJSON() ([]byte, error) {
	return interfacePair{t.raw, aliasCloudBuildTrigger(*t)}.MarshalJSON()
}
