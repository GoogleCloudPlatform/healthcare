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
func (s *LoggingSink) ImportID() string {
	return fmt.Sprintf("projects/%s/sinks/%s", s.Project, s.Name)
}
