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
	Subscriptions []*subscription `json:"subscriptions"`
}

type subscription struct {
	Bindings []Binding `json:"accessControl,omitempty"`
	raw      json.RawMessage
}

// Init initializes a new pubsub with the given project.
func (p *Pubsub) Init(project *Project) error {
	if p.Name() == "" {
		return errors.New("topic must be set")
	}

	appendGroupPrefix := func(ss ...string) []string {
		res := make([]string, 0, len(ss))
		for _, s := range ss {
			res = append(res, "group:"+s)
		}
		return res
	}

	defaultBindings := []Binding{
		{"roles/pubsub.editor", appendGroupPrefix(project.DataReadWriteGroups...)},
		{"roles/pubsub.viewer", appendGroupPrefix(project.DataReadOnlyGroups...)},
	}

	for _, s := range p.Subscriptions {
		s.Bindings = MergeBindings(append(defaultBindings, s.Bindings...)...)
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
type aliasSubscription subscription

func (s *subscription) UnmarshalJSON(data []byte) error {
	var alias aliasSubscription
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return fmt.Errorf("failed to unmarshal to parsed alias: %v", err)
	}
	*s = subscription(alias)
	return nil
}

func (s *subscription) MarshalJSON() ([]byte, error) {
	return interfacePair{s.raw, aliasSubscription(*s)}.MarshalJSON()
}
