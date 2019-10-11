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

	"github.com/GoogleCloudPlatform/healthcare/deploy/runner"
)

// PubsubTopic represents a Terraform pubsub topic.
type PubsubTopic struct {
	Name    string `json:"name"`
	Project string `json:"project"`

	Subscriptions []*PubsubSubscription `json:"_subscriptions"`

	raw json.RawMessage
}

// Init initializes the resource.
func (t *PubsubTopic) Init(projectID string) error {
	if t.Name == "" {
		return errors.New("name must be set")
	}
	if t.Project != "" {
		return fmt.Errorf("project must be unset: %v", t.Project)
	}
	for _, s := range t.Subscriptions {
		if s.Topic != "" {
			return fmt.Errorf("subscription topic must be unset: %v", s.Topic)
		}
		if err := s.Init(projectID); err != nil {
			return fmt.Errorf("failed to init subscription %v: %v", s, err)
		}
	}
	t.Project = projectID
	return nil
}

// ID returns the resource unique identifier.
func (t *PubsubTopic) ID() string {
	return t.Name
}

// ResourceType returns the resource terraform provider type.
func (t *PubsubTopic) ResourceType() string {
	return "google_pubsub_topic"
}

// ImportID returns the ID to use for terraform imports.
func (t *PubsubTopic) ImportID(runner runner.Runner) (string, error) {
	return fmt.Sprintf("%s/%s", t.Project, t.Name), nil
}

// DependentResources returns the child resources of this resource (subscriptions).
func (t *PubsubTopic) DependentResources() []Resource {
	var rs []Resource
	for _, s := range t.Subscriptions {
		s.Topic = fmt.Sprintf("${google_pubsub_topic.%s.name}", t.ID())
		rs = append(rs, s)
	}
	return rs
}

// aliasPubsubTopic is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
type aliasPubsubTopic PubsubTopic

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (t *PubsubTopic) UnmarshalJSON(data []byte) error {
	var alias aliasPubsubTopic
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return fmt.Errorf("failed to unmarshal to parsed alias: %v", err)
	}
	*t = PubsubTopic(alias)
	return nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (t *PubsubTopic) MarshalJSON() ([]byte, error) {
	return interfacePair{t.raw, aliasPubsubTopic(*t)}.MarshalJSON()
}

// PubsubSubscription represents a Terraform pubsub topic.
type PubsubSubscription struct {
	Name    string `json:"name"`
	Project string `json:"project"`
	Topic   string `json:"topic"`

	IAMMembers []*SubscriptionIAMMember `json:"_iam_members"`

	raw json.RawMessage
}

// Init initializes the resource.
func (s *PubsubSubscription) Init(projectID string) error {
	if s.Name == "" {
		return errors.New("name must be set")
	}
	if s.Project != "" {
		return fmt.Errorf("project must be unset: %v", s.Project)
	}
	s.Project = projectID
	return nil
}

// ID returns the resource unique identifier.
func (s *PubsubSubscription) ID() string {
	return s.Name
}

// ResourceType returns the resource terraform provider type.
func (s *PubsubSubscription) ResourceType() string {
	return "google_pubsub_subscription"
}

// ImportID returns the ID to use for terraform imports.
func (s *PubsubSubscription) ImportID(runner runner.Runner) (string, error) {
	return fmt.Sprintf("%s/%s", s.Project, s.Name), nil
}

// DependentResources returns the child resources of this resource (subscriptions).
func (s *PubsubSubscription) DependentResources() []Resource {
	if len(s.IAMMembers) == 0 {
		return nil
	}

	forEach := make(map[string]*SubscriptionIAMMember)
	for _, m := range s.IAMMembers {
		key := fmt.Sprintf("%s %s", m.Role, m.Member)
		forEach[key] = m
	}
	return []Resource{&SubscriptionIAMMember{
		ForEach:      forEach,
		Subscription: fmt.Sprintf("${google_pubsub_subscription.%s.name}", s.ID()),
		Role:         "${each.value.role}",
		Member:       "${each.value.member}",
		id:           s.ID(),
	}}
}

// aliasPubsubSubscription is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
type aliasPubsubSubscription PubsubSubscription

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (s *PubsubSubscription) UnmarshalJSON(data []byte) error {
	var alias aliasPubsubSubscription
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return fmt.Errorf("failed to unmarshal to parsed alias: %v", err)
	}
	*s = PubsubSubscription(alias)
	return nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (s *PubsubSubscription) MarshalJSON() ([]byte, error) {
	return interfacePair{s.raw, aliasPubsubSubscription(*s)}.MarshalJSON()
}

// SubscriptionIAMMember represents a Terraform subscription IAM member.
type SubscriptionIAMMember struct {
	Role   string `json:"role"`
	Member string `json:"member"`

	// The following fields should not be set by users.

	// ForEach is used to let a single iam member expand to reference multiple iam members
	// through the use of terraform's for_each iterator.
	ForEach map[string]*SubscriptionIAMMember `json:"for_each,omitempty"`

	// Subscription should be written as a terraform reference to a subscription name so that it is created after the subscription.
	// e.g. ${google_storage_bucket.foo_bucket.name}
	Subscription string `json:"subscription,omitempty"`

	// id should be the subscription's literal name.
	id string
}

// Init initializes the resource.
func (m *SubscriptionIAMMember) Init(string) error {
	return nil
}

// ID returns the unique identifier.
func (m *SubscriptionIAMMember) ID() string {
	return m.id
}

// ResourceType returns the terraform provider type.
func (m *SubscriptionIAMMember) ResourceType() string {
	return "google_pubsub_subscription_iam_member"
}
