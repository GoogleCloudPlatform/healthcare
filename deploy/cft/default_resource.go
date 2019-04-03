package cft

import "errors"

// DefaultResource represents a resource supported by CFT
type DefaultResource struct {
	DefaultResourceProperties `json:"properties"`
	templatePath              string
}

// DefaultResourceProperties represents a partial CFT resource implementation.
type DefaultResourceProperties struct {
	ResourceName string `json:"name"`
}

// Init initializes a new default resource with the given project.
func (dr *DefaultResource) Init(*Project) error {
	if dr.Name() == "" {
		return errors.New("name must be set")
	}
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
