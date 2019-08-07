// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/imdario/mergo"
)

// interfacePair should be used by resources that need to preserve the original user config input.
// It provides functionality to merge the original user input (raw) with the parsed when the struct is marshalled.
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

func unmarshalJSONMany(data []byte, vs ...interface{}) error {
	for _, v := range vs {
		if err := json.Unmarshal(data, v); err != nil {
			return err
		}
	}
	return nil

}
