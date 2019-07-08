package config

import (
	"encoding/json"
	"fmt"
)

// Forseti wraps the CFT Forseti module.
type Forseti struct {
	Project    *Project           `json:"project"`
	Properties *ForsetiProperties `json:"properties"`
}

// ForsetiProperties represents a partial CFT Forseti implementation.
type ForsetiProperties struct {
	// The following vars should not directly be set by users.
	ProjectID              string   `json:"project_id"`
	Domain                 string   `json:"domain"`
	CompositeRootResources []string `json:"composite_root_resources"`

	raw json.RawMessage
}

// Init initializes Forseti properties.
func (p *ForsetiProperties) Init() error {
	if p.ProjectID != "" {
		return fmt.Errorf("project_id must be unset: %v", p.ProjectID)
	}
	if p.Domain != "" {
		return fmt.Errorf("domain must be unset: %v", p.Domain)
	}
	if len(p.CompositeRootResources) > 0 {
		return fmt.Errorf("composite_root_resources must be unset: %v", p.CompositeRootResources)
	}
	return nil
}

// aliasCHCDataset is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
type aliasForsetiProperties ForsetiProperties

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (p *ForsetiProperties) UnmarshalJSON(data []byte) error {
	var alias aliasForsetiProperties
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return fmt.Errorf("failed to unmarshal to parsed alias: %v", err)
	}
	*p = ForsetiProperties(alias)
	return nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (p *ForsetiProperties) MarshalJSON() ([]byte, error) {
	return interfacePair{p.raw, aliasForsetiProperties(*p)}.MarshalJSON()
}
