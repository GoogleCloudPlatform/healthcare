package cft

// DefaultResource represents a resource supported by CFT
type DefaultResource struct {
	DefaultResourceProperties `yaml:"properties"`
	templatePath              string
}

// DefaultResourceProperties represents a partial CFT resource implementation.
type DefaultResourceProperties struct {
	ResourceName string `yaml:"name"`
}

// Init initializes a new default resource with the given project.
func (dr *DefaultResource) Init(*Project) error {
	return nil
}

// Name returns the name of this resource.
func (dr *DefaultResource) Name() string {
	return dr.ResourceName
}

// TemplatePath returns the name of the template to use for this resource.
func (dr *DefaultResource) TemplatePath() string {
	return dr.templatePath
}
