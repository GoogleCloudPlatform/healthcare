package config

import (
	"encoding/json"
	"errors"
	"fmt"
)

// DefaultResource represents a resource supported by CFT
type DefaultResource struct {
	DefaultResourceProperties `json:"properties"`
	OuterName                 string `json:"name,omitempty"`

	TmplPath string `json:"-"` // must be set in code
	raw      json.RawMessage
}

// DefaultResourceProperties represents a partial CFT resource implementation.
type DefaultResourceProperties struct {
	InnerName string `json:"name,omitempty"`
}

// Init initializes a new default resource with the given project.
func (dr *DefaultResource) Init(*Project) error {
	if dr.OuterName != "" && dr.InnerName != "" {
		return fmt.Errorf("name can only be defined once in outer or inner object: got %q, %q", dr.OuterName, dr.InnerName)
	}
	if dr.Name() == "" {
		return errors.New("name must be set")
	}
	if dr.TmplPath == "" {
		return errors.New("templatePath must be set")
	}
	return nil
}

// Name returns the name of this resource.
func (dr *DefaultResource) Name() string {
	if dr.OuterName != "" {
		return dr.OuterName
	}
	return dr.InnerName
}

// TemplatePath returns the name of the template to use for this resource.
func (dr *DefaultResource) TemplatePath() string {
	return dr.TmplPath
}

// aliasDefaultResource is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
// TODO: avoid doing this once generics support is implemented in go 2
type aliasDefaultResource DefaultResource

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (dr *DefaultResource) UnmarshalJSON(data []byte) error {
	var alias aliasDefaultResource
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return fmt.Errorf("failed to unmarshal to parsed alias: %v", err)
	}
	*dr = DefaultResource(alias)
	return nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (dr *DefaultResource) MarshalJSON() ([]byte, error) {
	return interfacePair{dr.raw, aliasDefaultResource(*dr)}.MarshalJSON()
}
