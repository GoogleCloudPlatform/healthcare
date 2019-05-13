package cft

// LogSink wraps a deployment manager Log Sink.
// TODO: see if we can use the CFT log sink template.
type LogSink struct {
	LogSinkProperties `json:"properties"`
}

// LogSinkProperties represents a partial DM log sink resource.
type LogSinkProperties struct {
	LogSinkName          string `json:"name"`
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
	return l.LogSinkName
}

// DeploymentManagerType returns the type to use for deployment manager.
func (*LogSink) DeploymentManagerType() string {
	return "logging.v2.sink"
}
