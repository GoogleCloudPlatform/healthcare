package cft

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/imdario/mergo"
)

// ResourcePair groups the raw resource with its parsed version.
type ResourcePair struct {
	Raw    json.RawMessage
	Parsed parsedResource
}

// MergedPropertiesMap merges the raw and parsed resources and extracts their properties map.
// See interfacePair.MergedMap for specifics on the merging.
func (p ResourcePair) MergedPropertiesMap() (map[string]interface{}, error) {
	merged, err := interfacePair{p.Raw, p.Parsed}.MergedMap()
	if err != nil {
		return nil, err
	}

	props, ok := merged["properties"]
	if !ok {
		return nil, fmt.Errorf("merged map is missing properties field: %v", merged)
	}

	m, ok := props.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("properties is not a map: %v", props)
	}

	return m, nil
}

// interfacePair should be used by lists of resources that are part of another resource's deployment.
// A concrete example of this is the pubsub resource. The resource defines a topic at the top level as well as a list of subscriptions.
// The subscriptions use interfacePair to merge the raw and parsed subscriptions together before mergo.Merge is called.
// This is required because mergo.Merge can only override or append slices, not merge them.
type interfacePair struct {
	// raw stores the original user input. Its purpose is to preserve fields not handled by parsed.
	// It can be empty if there is no user input to save.
	raw json.RawMessage

	// parsed represents the parsed version of raw. This is a struct that contains a subset
	// of the fields defined in raw.
	parsed interface{}
}

func (p interfacePair) MarshalJSON() ([]byte, error) {
	merged, err := p.MergedMap()
	if err != nil {
		return nil, err
	}
	return json.Marshal(merged)
}

// MergedMap merges raw and parsed into a single map.
// The raw resource can contain extra fields that were not parsed by its concrete equivalent.
// On the other hand, the parsed struct may set some defaults or change values.
// Thus, the merged map contains the union of all fields.
// For keys in the intersection, the parsed value is given precedence.
func (p interfacePair) MergedMap() (map[string]interface{}, error) {
	if p.parsed == nil {
		return nil, errors.New("parsed must not be nil")
	}

	merged := make(map[string]interface{})
	if err := convertJSON(p.parsed, &merged); err != nil {
		return nil, fmt.Errorf("failed to unmarshal parsed: %v\nparsed: %+v", err, p.parsed)
	}

	if len(p.raw) > 0 {
		rawMap := make(map[string]interface{})
		if err := json.Unmarshal(p.raw, &rawMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal raw: %v,\nraw: %v", err, string(p.raw))
		}
		if err := mergo.Merge(&merged, &rawMap); err != nil {
			return nil, fmt.Errorf("failed to merge raw map with its parsed version: %v", err)
		}
	}
	return merged, nil
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
