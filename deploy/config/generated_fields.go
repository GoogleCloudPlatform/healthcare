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
	"fmt"
)

// AllGeneratedFields defines the generated_fields block.
// AllGeneratedFields contains resource information when the resources are deployed.
// See field_generation_test for examples.
type AllGeneratedFields struct {
	Projects map[string]*GeneratedFields `json:"projects,omitempty"`
	Forseti  *ForsetiServiceInfo         `json:"forseti,omitempty"`
}

// GeneratedFields defines the generated_fields of a single project.
type GeneratedFields struct {
	ProjectNumber         string            `json:"project_number,omitempty"`
	LogSinkServiceAccount string            `json:"log_sink_service_account,omitempty"`
	GCEInstanceInfoList   []GCEInstanceInfo `json:"gce_instance_info,omitempty"`
	FailedStep            int               `json:"failed_step,omitempty"`
}

// GCEInstanceInfo defines the generated fields for instances in a project.
type GCEInstanceInfo struct {
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty"`
}

// ForsetiServiceInfo defines the generated_fields of the forseti service.
type ForsetiServiceInfo struct {
	ServiceAccount string `json:"service_account,omitempty"`
	ServiceBucket  string `json:"server_bucket,omitempty"`
}

// InstanceID returns the ID of the instance with the given name.
func (g *GeneratedFields) InstanceID(name string) (string, error) {
	for _, info := range g.GCEInstanceInfoList {
		if info.Name == name {
			return info.ID, nil
		}
	}
	return "", fmt.Errorf("info for instance %q not found in generated_fields", name)
}
