package config

import "encoding/json"

// TerraformModule provides a terraform module config.
type TerraformModule struct {
	Source     string
	Properties interface{}
}

// MarshalJSON provides a custom marshaller that merges all fields of the struct into a single level.
func (t *TerraformModule) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	if err := convertJSON(t.Properties, &m); err != nil {
		return nil, err
	}
	m["source"] = t.Source
	return json.Marshal(m)
}
