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

package config

import (
	"errors"
)

var (
	unexpectedUserDescriptor = descriptor{
		MetricKind: "DELTA",
		ValueType:  "INT64",
		Unit:       "1",
		Labels: []label{{
			Key:         "user",
			ValueType:   "STRING",
			Description: "Unexpected user",
		}},
	}
	principalEmailLabelExtractor = map[string]string{
		"user": "EXTRACT(protoPayload.authenticationInfo.principalEmail)",
	}
)

// Metric wraps a logging metric.
// Note: metrics cannot be created by users, so do not implement custom json marshallers.
type Metric struct {
	MetricProperties `json:"properties"`
	dependencies     []string
}

// MetricProperties wraps the metric template properties.
type MetricProperties struct {
	MetricName      string            `json:"metric"`
	Description     string            `json:"description"`
	Filter          string            `json:"filter"`
	Descriptor      descriptor        `json:"metricDescriptor"`
	LabelExtractors map[string]string `json:"labelExtractors"`
}

type descriptor struct {
	MetricKind string  `json:"metricKind"`
	ValueType  string  `json:"valueType"`
	Unit       string  `json:"unit"`
	Labels     []label `json:"labels"`
}

type label struct {
	Key         string `json:"key"`
	ValueType   string `json:"valueType"`
	Description string `json:"description"`
}

// Init initializes the metric.
func (m *Metric) Init() error {
	if m.MetricName == "" {
		return errors.New("metric name must be set")
	}
	return nil
}

// Name returns the name of the metric.
func (m *Metric) Name() string {
	return m.MetricName
}

// DeploymentManagerType returns the type to use for deployment manager.
func (m *Metric) DeploymentManagerType() string {
	return "logging.v2.metric"
}

// Dependencies gets the dependencies of this metric.
func (m *Metric) Dependencies() []string {
	return m.dependencies
}
