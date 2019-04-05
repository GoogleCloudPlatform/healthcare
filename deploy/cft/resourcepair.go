package cft

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/imdario/mergo"
)

// resourcePair groups the raw resource map with its parsed version.
type resourcePair struct {
	// raw stores the original user input. Its purpose is to preserve fields not handled by parsed.
	// It can be empty if there is no user input to save.
	raw json.RawMessage

	// parsed represents the parsed version of raw. This is a struct that contains a subset
	// of the fields defined in raw.
	parsed parsedResource
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
	if err := convertJSON(r.parsed, &parsedMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal parsed: %v\nparsed: %+v", err, r.parsed)
	}

	if len(r.raw) > 0 {
		rawMap := make(map[string]interface{})
		if err := json.Unmarshal(r.raw, &rawMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal raw: %v,\nraw: %v", err, string(r.raw))
		}
		if err := mergo.Merge(&parsedMap, &rawMap); err != nil {
			return nil, fmt.Errorf("failed to merge raw map with its parsed version: %v", err)
		}
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

// convertJSON converts one json supported type into another.
// Note: both in and out must support json marshalling.
// See json.Unmarshal details on supported types.
func convertJSON(in interface{}, out interface{}) error {
	b, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("failed to marshal %v: %v", in, err)
	}

	if err := json.Unmarshal(b, out); err != nil {
		return fmt.Errorf("failed to unmarshal %v: %v", string(b), err)
	}
	return nil
}
