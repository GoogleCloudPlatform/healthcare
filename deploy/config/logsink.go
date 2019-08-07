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

// LogSink wraps a deployment manager Log Sink.
// Note: log sinks cannot be created by users, so do not implement custom json marshallers.
// TODO: see if we can use the CFT log sink template.
type LogSink struct {
	LogSinkProperties `json:"properties"`
}

// LogSinkProperties represents a partial DM log sink resource.
type LogSinkProperties struct {
	Sink                 string `json:"sink"`
	Destination          string `json:"destination"`
	Filter               string `json:"filter"`
	UniqueWriterIdentity bool   `json:"uniqueWriterIdentity"`
}

// Init initializes the instance.
func (l *LogSink) Init() error {
	return nil
}

// Name returns the name of this log sink.
func (l *LogSink) Name() string {
	return l.Sink
}

// DeploymentManagerType returns the type to use for deployment manager.
func (*LogSink) DeploymentManagerType() string {
	return "logging.v2.sink"
}
