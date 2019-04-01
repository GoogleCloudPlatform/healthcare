package cft

import (
	"errors"
)

// Metric wraps a logging metric.
type Metric struct {
	MetricProperties `json:"properties"`
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
func (m *Metric) Init(*Project) error {
	if m.MetricName == "" {
		return errors.New("metric name must be set")
	}
	return nil
}

// Name returns the name of the metric.
func (m *Metric) Name() string {
	return m.MetricName
}

// TemplatePath returns the name of the template to use for the metric.
func (m *Metric) TemplatePath() string {
	return "deploy/templates/metric.py"
}
