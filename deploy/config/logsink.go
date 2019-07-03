package config

// LogSink wraps a deployment manager Log Sink.
// Note: log sinks cannot be created by users, so do not implement custom json marshallers.
// TODO: see if we can use the CFT log sink template.
type LogSink struct {
	LogSinkProperties `json:"properties"`
}

// LogSinkProperties represents a partial DM log sink resource.
type LogSinkProperties struct {
	Sink                 string `json:"sink"`
	Destination          string `json:"destination"`
	Filter               string `json:"filter"`
	UniqueWriterIdentity bool   `json:"uniqueWriterIdentity"`
}

// Init initializes the instance.
func (l *LogSink) Init(p *Project) error {
	return nil
}

// Name returns the name of this log sink.
func (l *LogSink) Name() string {
	return l.Sink
}

// DeploymentManagerType returns the type to use for deployment manager.
func (*LogSink) DeploymentManagerType() string {
	return "logging.v2.sink"
}
