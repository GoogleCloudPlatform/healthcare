package cft

// GCEInstance wraps a CFT GCE Instance.
type GCEInstance struct {
	GCEInstanceProperties `json:"properties"`
}

// GCEInstanceProperties represents a partial CFT instance implementation.
type GCEInstanceProperties struct {
	GCEInstanceName string `json:"name"`
	Zone            string `json:"zone"`
}

// Init initializes the instance.
func (i *GCEInstance) Init(*Project) error {
	return nil
}

// Name returns the name of this instance.
func (i *GCEInstance) Name() string {
	return i.GCEInstanceName
}

// TemplatePath returns the name of the template to use for this instance.
func (i *GCEInstance) TemplatePath() string {
	return "deploy/cft/templates/instance.py"
}
