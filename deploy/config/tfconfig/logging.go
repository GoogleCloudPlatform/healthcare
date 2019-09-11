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
	"errors"
	"fmt"
)

// LoggingSink represents a logging sink.
type LoggingSink struct {
	Name                 string `json:"name"`
	Project              string `json:"project"`
	Destination          string `json:"destination"`
	Filter               string `json:"filter"`
	UniqueWriterIdentity bool   `json:"unique_writer_identity"`
}

// Init initializes the resource.
func (s *LoggingSink) Init(projectID string) error {
	s.Project = projectID
	return nil
}

// ID returns the resource unique identifier.
func (s *LoggingSink) ID() string {
	return s.Name
}

// ResourceType returns the resource terraform provider type.
func (*LoggingSink) ResourceType() string {
	return "google_logging_project_sink"
}

// ImportID returns the ID to use for terraform imports.
func (s *LoggingSink) ImportID() (string, error) {
	return fmt.Sprintf("projects/%s/sinks/%s", s.Project, s.Name), nil
}

// LoggingMetric represents a Terraform logging metric.
type LoggingMetric struct {
	Name             string            `json:"name"`
	Project          string            `json:"project"`
	Description      string            `json:"description"`
	Filter           string            `json:"filter,omitempty"`
	MetricDescriptor *MetricDescriptor `json:"metric_descriptor,omitempty"`
	ValueExtractor   string            `json:"value_extractor,omitempty"`
	LabelExtractors  map[string]string `json:"label_extractors,omitempty"`
}

// MetricDescriptor is the metric descriptor associated with the logs-based metric.
type MetricDescriptor struct {
	MetricKind string   `json:"metric_kind,omitempty"`
	ValueType  string   `json:"value_type,omitempty"`
	Unit       string   `json:"unit,omitempty"`
	Labels     []*Label `json:"labels,omitempty"`
}

// Label can be used to describe a specific instance of this metric type.
type Label struct {
	Key         string `json:"key,omitempty"`
	ValueType   string `json:"value_type,omitempty"`
	Description string `json:"description,omitempty"`
}

// Init initializes the resource.
func (m *LoggingMetric) Init(projectID string) error {
	if m.Name == "" {
		return errors.New("name must be set")
	}
	if m.Project != "" {
		return fmt.Errorf("project must not be set: %v", m.Project)
	}
	m.Project = projectID
	return nil
}

// ID returns the resource unique identifier.
func (m *LoggingMetric) ID() string {
	return m.Name
}

// ResourceType returns the resource terraform provider type.
func (m *LoggingMetric) ResourceType() string {
	return "google_logging_metric"
}

// ImportID returns the ID to use for terraform imports
func (m *LoggingMetric) ImportID() (string, error) {
	return m.Name, nil
}
