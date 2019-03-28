package cft

import (
	"errors"
)

// Metric wraps a logging metric.
type Metric struct {
	MetricProperties `yaml:"properties"`
}

// MetricProperties wraps the metric template properties.
type MetricProperties struct {
	MetricName      string            `yaml:"metric"`
	Description     string            `yaml:"description"`
	Filter          string            `yaml:"filter"`
	Descriptor      descriptor        `yaml:"metricDescriptor"`
	LabelExtractors map[string]string `yaml:"labelExtractors"`
}

type descriptor struct {
	MetricKind string  `yaml:"metricKind"`
	ValueType  string  `yaml:"valueType"`
	Unit       string  `yaml:"unit"`
	Labels     []label `yaml:"labels"`
}

type label struct {
	Key         string `yaml:"key"`
	ValueType   string `yaml:"valueType"`
	Description string `yaml:"description"`
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
