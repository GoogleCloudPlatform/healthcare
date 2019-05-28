package cft

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ghodss/yaml"
)

type testResource struct {
	CFTResource struct {
		A     int `json:"a,omitempty"`
		Inner *struct {
			B int `json:"b"`
		} `json:"inner,omitempty"`
	} `json:"properties"`
}

func (r testResource) Init(*Project) error {
	return nil
}

func (r testResource) Name() string {
	return "foo-resource"
}

func (r testResource) TemplatePath() string {
	return "does_not_exist.py"
}

func TestMergedMap(t *testing.T) {
	tests := []struct {
		name, parsed, raw, want string
	}{
		{
			"empty_raw",
			"properties: {a: 1}",
			"{}",
			"a: 1",
		},
		{
			"empty_parsed",
			"",
			"properties: {a: 1}",
			"a: 1",
		},
		{
			"override",
			"properties: {a: 2}",
			"properties: {a: 1}",
			"a: 2",
		},
		{
			"inner_parsed",
			"properties: {inner: {b: 1}}",
			"{}",
			"inner: {b: 1}",
		},
		{
			"inner_raw",
			"",
			"properties: {inner: {b: 1}}",
			"inner: {b: 1}",
		},
		{
			"inner_override",
			"properties: {inner: {b: 2}}",
			"properties: {inner: {b: 1}}",
			"inner: {b: 2}",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parsed := new(testResource)
			var raw json.RawMessage
			want := make(map[string]interface{})

			if err := yaml.Unmarshal([]byte(tc.parsed), parsed); err != nil {
				t.Fatalf("yaml.Unmarshal parsed: %v", err)
			}
			if err := yaml.Unmarshal([]byte(tc.raw), &raw); err != nil {
				t.Fatalf("yaml.Unmarshal raw: %v", err)
			}
			if err := yaml.Unmarshal([]byte(tc.want), &want); err != nil {
				t.Fatalf("yaml.Unmarshal want: %v", err)
			}

			pair := ResourcePair{Parsed: parsed, Raw: raw}

			got, err := pair.MergedPropertiesMap()
			if err != nil {
				t.Fatalf("tc.resourceGroup.MergedPropertiesMap(): %v", err)
			}

			if diff := cmp.Diff(got, want); diff != "" {
				t.Errorf("resource merged map differs (-got +want):\n%v", diff)
			}
		})
	}
}
