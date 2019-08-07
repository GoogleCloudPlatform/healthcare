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

// Pubsub represents a GCP pubsub channel resource.
type Pubsub struct {
	PubsubProperties `json:"properties"`
	raw              json.RawMessage
}

// PubsubProperties represents a partial CFT pubsub implementation.
type PubsubProperties struct {
	TopicName     string          `json:"topic"`
	Subscriptions []*Subscription `json:"subscriptions"`
}

// Subscription represents a partial subscription impementation.
type Subscription struct {
	Bindings []Binding `json:"accessControl,omitempty"`
	raw      json.RawMessage
}

// Init initializes a new pubsub with the given project.
func (p *Pubsub) Init() error {
	if p.Name() == "" {
		return errors.New("topic must be set")
	}

	return nil
}

// Name returns the name of this pubsub.
func (p *Pubsub) Name() string {
	return p.TopicName
}

// TemplatePath returns the name of the template to use for this pubsub.
func (p *Pubsub) TemplatePath() string {
	return "deploy/config/templates/pubsub/pubsub.py"
}

// aliasPubsub is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
type aliasPubsub Pubsub

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (p *Pubsub) UnmarshalJSON(data []byte) error {
	var alias aliasPubsub
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return fmt.Errorf("failed to unmarshal to parsed alias: %v", err)
	}
	*p = Pubsub(alias)
	return nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (p *Pubsub) MarshalJSON() ([]byte, error) {
	return interfacePair{p.raw, aliasPubsub(*p)}.MarshalJSON()
}

// aliasSubscription is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
type aliasSubscription Subscription

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (s *Subscription) UnmarshalJSON(data []byte) error {
	var alias aliasSubscription
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return fmt.Errorf("failed to unmarshal to parsed alias: %v", err)
	}
	*s = Subscription(alias)
	return nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (s *Subscription) MarshalJSON() ([]byte, error) {
	return interfacePair{s.raw, aliasSubscription(*s)}.MarshalJSON()
}
