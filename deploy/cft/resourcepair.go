package cft

import (
	"errors"
	"fmt"

	"github.com/imdario/mergo"
)

// resourcePair groups the raw resource map with its parsed version.
type resourcePair struct {
	// parsed represents the parsed version of raw. This is a struct that contains a subset
	// of the fields defined in raw.
	parsed parsedResource

	// raw stores the original user input. Its purpose is to preserve fields not handled by parsed. This map can be empty or nil if there is no user input (e.g. resource created internally).
	raw interface{}
}

// MergedPropertiesMap merges the raw and parsed resources into a single map.
// The raw resource map can contain extra fields that were not parsed by its concrete equivalent.
// On the other hand, the parsed resource may set some defaults or change values.
// Thus, the merged map contains the union of all fields.
// For keys in the intersection, the parsed value is given precedence.
func (r resourcePair) MergedPropertiesMap() (map[string]interface{}, error) {
	if r.parsed == nil {
		return nil, errors.New("parsed must not be nil")
	}
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
