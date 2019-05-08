package cft

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Pubsub represents a GCP pubsub channel resource.
type Pubsub struct {
	PubsubProperties `json:"properties"`
}

// PubsubProperties represents a partial CFT pubsub implementation.
type PubsubProperties struct {
	TopicName         string              `json:"topic"`
	SubscriptionPairs []*subscriptionPair `json:"subscriptions"`
}

// subscriptionPair is used to retain fields not defined by the parsed subscription.
// mergo.Merge can only override or append slices, not merge them,
// so this merges the raw and parsed subscriptions before the pubsub merge happens.
type subscriptionPair struct {
	raw    json.RawMessage
	parsed subscription
}

func (p *subscriptionPair) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &p.raw); err != nil {
		return fmt.Errorf("failed to unmarshal subscription into raw form: %v", err)
	}
	if err := json.Unmarshal(data, &p.parsed); err != nil {
		return fmt.Errorf("failed to unmarshal subscription into parsed form: %v", err)
	}
	return nil
}

func (p subscriptionPair) MarshalJSON() ([]byte, error) {
	return interfacePair{p.raw, p.parsed}.MarshalJSON()
}

type subscription struct {
	Bindings []Binding `json:"accessControl,omitempty"`
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

	for _, subp := range p.SubscriptionPairs {
		subp.parsed.Bindings = MergeBindings(append(defaultBindings, subp.parsed.Bindings...)...)
	}

	return nil
}

// Name returns the name of this pubsub.
func (p *Pubsub) Name() string {
	return p.TopicName
}

// TemplatePath returns the name of the template to use for this pubsub.
func (p *Pubsub) TemplatePath() string {
	return "deploy/cft/templates/pubsub/pubsub.py"
}
