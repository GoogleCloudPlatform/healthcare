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

// GCSBucket wraps a CFT Cloud Storage Bucket.
type GCSBucket struct {
	GCSBucketProperties `json:"properties"`
	TTLDays             int      `json:"ttl_days,omitempty"`
	ExpectedUsers       []string `json:"expected_users,omitempty"`
	raw                 json.RawMessage
}

// GCSBucketProperties  represents a partial CFT bucket implementation.
type GCSBucketProperties struct {
	GCSBucketName              string     `json:"name"`
	Location                   string     `json:"location"`
	Bindings                   []Binding  `json:"bindings"`
	StorageClass               string     `json:"storageClass,omitempty"`
	Versioning                 versioning `json:"versioning"`
	Lifecycle                  *lifecycle `json:"lifecycle,omitempty"`
	PredefinedACL              string     `json:"predefinedAcl,omitempty"`
	PredefinedDefaultObjectACL string     `json:"predefinedDefaultObjectAcl,omitempty"`
	Logging                    *logging   `json:"logging,omitempty"`
}

type versioning struct {
	// Use pointer to differentiate between zero value and intentionally being set to false.
	Enabled *bool `json:"enabled"`
}

type logging struct {
	LogBucket string `json:"logBucket"`
}

type lifecycle struct {
	Rules []*LifecycleRule `json:"rule,omitempty"`
}

// LifecycleRule defines a partial bucket lifecycle rule implementation.
type LifecycleRule struct {
	Action    *action    `json:"action,omitempty"`
	Condition *condition `json:"condition,omitempty"`

	raw json.RawMessage
}

type action struct {
	Type string `json:"type,omitempty"`
}

type condition struct {
	Age    int  `json:"age,omitempty"`
	IsLive bool `json:"isLive,omitempty"`
}

// aliasGCSBucket is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
type aliasLifecycleRule LifecycleRule

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (r *LifecycleRule) UnmarshalJSON(data []byte) error {
	var alias aliasLifecycleRule
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return fmt.Errorf("failed to unmarshal to parsed alias: %v", err)
	}
	*r = LifecycleRule(alias)
	return nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (r *LifecycleRule) MarshalJSON() ([]byte, error) {
	return interfacePair{r.raw, aliasLifecycleRule(*r)}.MarshalJSON()
}

// Init initializes the bucket with the given project.
func (b *GCSBucket) Init() error {
	if b.GCSBucketName == "" {
		return errors.New("name must be set")
	}
	if b.Location == "" {
		return errors.New("location must be set")
	}
	if b.Versioning.Enabled != nil && !*b.Versioning.Enabled {
		return errors.New("versioning must not be disabled")
	}
	if b.PredefinedACL != "" || b.PredefinedDefaultObjectACL != "" {
		return errors.New("predefined ACLs must not be set")
	}

	t := true
	b.Versioning.Enabled = &t

	if b.TTLDays > 0 {
		if b.Lifecycle == nil {
			b.Lifecycle = &lifecycle{}
		}
		b.Lifecycle.Rules = append(b.Lifecycle.Rules, &LifecycleRule{
			Action:    &action{Type: "Delete"},
			Condition: &condition{Age: b.TTLDays, IsLive: true},
		})
	}
	return nil
}

// Name returns the name of the bucket.
func (b *GCSBucket) Name() string {
	return b.GCSBucketName
}

// TemplatePath returns the name of the template to use for the bucket.
func (b *GCSBucket) TemplatePath() string {
	return "config/templates/gcs_bucket/gcs_bucket.py"
}

// aliasGCSBucket is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
type aliasGCSBucket GCSBucket

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (b *GCSBucket) UnmarshalJSON(data []byte) error {
	var alias aliasGCSBucket
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return fmt.Errorf("failed to unmarshal to parsed alias: %v", err)
	}
	*b = GCSBucket(alias)
	return nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (b *GCSBucket) MarshalJSON() ([]byte, error) {
	return interfacePair{b.raw, aliasGCSBucket(*b)}.MarshalJSON()
}
