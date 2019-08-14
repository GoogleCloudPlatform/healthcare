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
	"encoding/json"
	"errors"
	"fmt"
)

// GCEInstance wraps a CFT GCE Instance.
type GCEInstance struct {
	GCEInstanceProperties `json:"properties"`
	CustomBootImage       *struct {
		ImageName string `json:"image_name"`
		GCSPath   string `json:"gcs_path"`
	} `json:"custom_boot_image,omitempty"`
	raw json.RawMessage
}

// GCEInstanceProperties represents a partial CFT instance implementation.
type GCEInstanceProperties struct {
	GCEInstanceName string `json:"name"`
	Zone            string `json:"zone"`
	DiskImage       string `json:"diskImage,omitempty"`
}

// Init initializes the instance.
func (i *GCEInstance) Init() error {
	if i.CustomBootImage != nil {
		if i.DiskImage != "" {
			return errors.New("custom boot image and disk image cannot both be set")
		}
		i.DiskImage = "global/images/" + i.CustomBootImage.ImageName
	}
	return nil
}

// Name returns the name of this instance.
func (i *GCEInstance) Name() string {
	return i.GCEInstanceName
}

// TemplatePath returns the name of the template to use for this instance.
func (i *GCEInstance) TemplatePath() string {
	return "deploy/config/templates/instance/instance.py"
}

// aliasGCEInstance is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
type aliasGCEInstance GCEInstance

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (i *GCEInstance) UnmarshalJSON(data []byte) error {
	var alias aliasGCEInstance
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return fmt.Errorf("failed to unmarshal to parsed alias: %v", err)
	}
	*i = GCEInstance(alias)
	return nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (i *GCEInstance) MarshalJSON() ([]byte, error) {
	return interfacePair{i.raw, aliasGCEInstance(*i)}.MarshalJSON()
}
