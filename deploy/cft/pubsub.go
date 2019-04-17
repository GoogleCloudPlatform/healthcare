package cft

import "errors"

// Pubsub represents a GCP pubsub channel resource.
type Pubsub struct {
	PubsubProperties `json:"properties"`
}

// PubsubProperties represents a partial CFT pubsub implementation.
type PubsubProperties struct {
	TopicName string `json:"topic"`

	// Note: mergo.Merge does not merge slices together, so subscription should implement all fields, even if it does not use it so that fields are not lost.
	// TODO: figure out a way to avoid doing this.
	Subscriptions []*subscription `json:"subscriptions"`
}

type subscription struct {
	Name               string    `json:"name"`
	Bindings           []binding `json:"accessControl,omitempty"`
	PushEndpoint       string    `json:"pushEndpoint,omitempty"`
	AckDeadlineSeconds int       `json:"ackDeadlineSeconds,omitempty"`
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

	defaultBindings := []binding{
		{"roles/pubsub.editor", appendGroupPrefix(project.DataReadWriteGroups...)},
		{"roles/pubsub.viewer", appendGroupPrefix(project.DataReadOnlyGroups...)},
	}

	for _, sub := range p.Subscriptions {
		sub.Bindings = mergeBindings(append(defaultBindings, sub.Bindings...)...)
	}

	return nil
}

// Name returns the name of this pubsub.
func (p *Pubsub) Name() string {
	return p.TopicName
}

// TemplatePath returns the name of the template to use for this pubsub.
func (p *Pubsub) TemplatePath() string {
	return "deploy/cft/templates/pubsub.py"
}
