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
	"os/exec"

	"github.com/GoogleCloudPlatform/healthcare/deploy/runner"
)

// MonitoringNotificationChannel represents a Terraform monitoring notification channel.
type MonitoringNotificationChannel struct {
	DisplayName string                 `json:"display_name"`
	Project     string                 `json:"project"`
	Type        string                 `json:"type,omitempty"`
	Labels      map[string]interface{} `json:"labels,omitempty"`

	Email string `json:"_email"`

	raw json.RawMessage
}

// Init initializes the resource.
func (c *MonitoringNotificationChannel) Init(projectID string) error {
	if c.DisplayName == "" {
		return errors.New("display_name must be set")
	}
	if c.Project != "" {
		return fmt.Errorf("project must not be set: %v", c.Project)
	}
	c.Project = projectID
	if c.Email != "" {
		if c.Type != "" {
			return fmt.Errorf("type must not be set when _email is set: %v", c.Type)
		}
		if len(c.Labels) > 0 {
			return fmt.Errorf("labels must be unset when _email is set: %v", c.Labels)
		}
		c.Type = "email"
		c.Labels = map[string]interface{}{
			"email_address": c.Email,
		}
	}
	return nil
}

// ID returns the resource unique identifier.
func (c *MonitoringNotificationChannel) ID() string {
	return standardizeID(c.DisplayName)
}

// ResourceType returns the resource terraform provider type.
func (c *MonitoringNotificationChannel) ResourceType() string {
	return "google_monitoring_notification_channel"
}

// ImportID returns the ID to use for terraform imports.
func (c *MonitoringNotificationChannel) ImportID(runner runner.Runner) (string, error) {
	// Check channel existence and create if not.
	cmd := exec.Command("gcloud", "--project", c.Project, "alpha", "monitoring", "channels", "list",
		"--format", "json")

	out, err := runner.CmdOutput(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to list existing monitoring channels: %v", err)
	}

	var channels []gcloudMonitoringResource
	if err := json.Unmarshal(out, &channels); err != nil {
		return "", fmt.Errorf("failed to unmarshal exist monitoring channels list output: %v", err)
	}

	var matched []string
	for _, gc := range channels {
		if c.DisplayName == gc.DisplayName {
			matched = append(matched, gc.Name)
		}
	}
	switch len(matched) {
	case 0:
		return "", nil
	case 1:
		return matched[0], nil
	default:
		return "", fmt.Errorf("multiple channels with display name %q found: %v", c.DisplayName, matched)
	}
}

// aliasBQDataset is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
type aliasMonitoringNotificationChannel MonitoringNotificationChannel

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (c *MonitoringNotificationChannel) UnmarshalJSON(data []byte) error {
	var alias aliasMonitoringNotificationChannel
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return fmt.Errorf("failed to unmarshal to parsed alias: %v", err)
	}
	*c = MonitoringNotificationChannel(alias)
	return nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (c *MonitoringNotificationChannel) MarshalJSON() ([]byte, error) {
	return interfacePair{c.raw, aliasMonitoringNotificationChannel(*c)}.MarshalJSON()
}

// MonitoringAlertPolicy represents a Terraform monitoring alert policy.
type MonitoringAlertPolicy struct {
	DisplayName          string         `json:"display_name"`
	Project              string         `json:"project"`
	Combiner             string         `json:"combiner"`
	Conditions           []*Condition   `json:"conditions,omitempty"`
	NotificationChannels []string       `json:"notification_channels,omitempty"`
	Documentation        *Documentation `json:"documentation,omitempty"`

	raw json.RawMessage
}

// Condition is a condition for the policy.
// If the combined conditions evaluate to true, then an incident is created.
type Condition struct {
	DisplayName        string              `json:"display_name,omitempty"`
	ConditionThreshold *ConditionThreshold `json:"condition_threshold,omitempty"`
}

// ConditionThreshold is a condition that compares a time series against a threshold.
type ConditionThreshold struct {
	Filter     string `json:"filter,omitempty"`
	Comparison string `json:"comparison"`
	Duration   string `json:"duration"`
}

// Documentation is a short name or phrase used to identify the policy in dashboards, notifications, and incidents.
type Documentation struct {
	Content  string `json:"content,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
}

// Init initializes the resource.
func (p *MonitoringAlertPolicy) Init(projectID string) error {
	if p.DisplayName == "" {
		return errors.New("display_name must be set")
	}
	if p.Project != "" {
		return fmt.Errorf("project must not be set: %v", p.Project)
	}
	p.Project = projectID
	return nil
}

// ID returns the resource unique identifier.
func (p *MonitoringAlertPolicy) ID() string {
	return standardizeID(p.DisplayName)
}

// ResourceType returns the resource terraform provider type.
func (p *MonitoringAlertPolicy) ResourceType() string {
	return "google_monitoring_alert_policy"
}

// ImportID returns the ID to use for terraform imports.
func (p *MonitoringAlertPolicy) ImportID(runner runner.Runner) (string, error) {
	// Get existing alerts.
	cmd := exec.Command("gcloud", "--project", p.Project, "alpha", "monitoring", "policies", "list", "--format", "json")

	out, err := runner.CmdOutput(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to list existing monitoring alert policies: %v", err)
	}

	var policies []gcloudMonitoringResource
	if err := json.Unmarshal(out, &policies); err != nil {
		return "", fmt.Errorf("failed to unmarshal exist monitoring channels list output: %v", err)
	}

	var matched []string
	for _, gp := range policies {
		if p.DisplayName == gp.DisplayName {
			matched = append(matched, gp.Name)
		}
	}
	switch len(matched) {
	case 0:
		return "", nil
	case 1:
		return matched[0], nil
	default:
		return "", fmt.Errorf("multiple policies with display name %q found: %v", p.DisplayName, matched)
	}
}

// aliasBQDataset is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
type aliasMonitoringAlertPolicy MonitoringAlertPolicy

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (p *MonitoringAlertPolicy) UnmarshalJSON(data []byte) error {
	var alias aliasMonitoringAlertPolicy
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return fmt.Errorf("failed to unmarshal to parsed alias: %v", err)
	}
	*p = MonitoringAlertPolicy(alias)
	return nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (p *MonitoringAlertPolicy) MarshalJSON() ([]byte, error) {
	return interfacePair{p.raw, aliasMonitoringAlertPolicy(*p)}.MarshalJSON()
}

type gcloudMonitoringResource struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}
