package cft

import (
	"fmt"

	"github.com/imdario/mergo"
)

// resourcePair groups the raw resource map with its parsed version.
type resourcePair struct {
	parsed parsedResource
	raw    interface{}
}

// MergedPropertiesMap merges the raw and parsed resources into a single map.
// The raw resource map can contain extra fields that were not parsed by its concrete equivalent.
// On the other hand, the parsed resource may set some defaults or change values.
// Thus, the merged map contains the union of all fields.
// For keys in the intersection, the parsed value is given precedence.
func (r resourcePair) MergedPropertiesMap() (map[string]interface{}, error) {
	parsedMap := make(map[string]interface{})
	if err := unmarshal(r.parsed, &parsedMap); err != nil {
		return nil, err
	}

	if err := mergo.Merge(&parsedMap, r.raw); err != nil {
		return nil, fmt.Errorf("failed to merge raw map with its parsed version: %v", err)
	}

	props, ok := parsedMap["properties"]
	if !ok {
		return nil, fmt.Errorf("merged map is missing properties field: %v", parsedMap)
	}

	m, ok := props.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("properties is not a map: %v", props)
	}

	return m, nil
}
