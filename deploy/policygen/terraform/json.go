/*
 * Copyright 2020 Google LLC.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package terraform provides definitions of Terraform resources in JSON format.
// This intentially does not define all fields in the plan JSON.
// https://www.terraform.io/docs/internals/json-format.html#plan-representation
// TODO: move to a common location that can be shared with Terraform Importer.
package terraform

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

const maxChildModuleLevel = 100

// https://www.terraform.io/docs/internals/json-format.html#plan-representation
type plan struct {
	Variables       map[string]variable `json:"variables"`
	PlannedValues   values              `json:"planned_values"`
	ResourceChanges []ResourceChange    `json:"resource_changes"`
	Configuration   Configuration       `json:"configuration"`
}

type variable struct {
	Value interface{} `json:"value"`
}

// https://www.terraform.io/docs/internals/json-format.html#state-representation
type state struct {
	Values values `json:"values"`
}

// https://www.terraform.io/docs/internals/json-format.html#values-representation
type values struct {
	RootModules struct {
		Resources    []Resource    `json:"resources"`
		ChildModules []childModule `json:"child_modules"`
	} `json:"root_module"`
}

type childModule struct {
	Address      string        `json:"address"`
	Resources    []Resource    `json:"resources"`
	ChildModules []childModule `json:"child_modules"`
}

// Resource represent single Terraform resource definition.
type Resource struct {
	Name    string                 `json:"name"`
	Address string                 `json:"address"`
	Kind    string                 `json:"type"`
	Mode    string                 `json:"mode"` // "managed" for resources, or "data" for data resources
	Values  map[string]interface{} `json:"values"`
}

// ResourceChange represents a Terraform resource change from a Terraform plan.
// See "resource_changes" at https://www.terraform.io/docs/internals/json-format.html#plan-representation
type ResourceChange struct {
	Address       string `json:"address"`
	ModuleAddress string `json:"module_address"`
	Mode          string `json:"mode"` // "managed" for resources, or "data" for data resources
	Kind          string `json:"type"`
	Name          string `json:"name"`
	Index         string `json:"index,omitempty"`
	Change        Change `json:"change"`
}

// Change represents the "Change" element of a Terraform resource change from a Terraform plan.
// https://www.terraform.io/docs/internals/json-format.html#change-representation
type Change struct {
	Actions []string `json:"actions"`
	// These are "value-representation", not "values-representation" and the keys are resource-specific.
	Before       map[string]interface{} `json:"before"`
	After        map[string]interface{} `json:"after"`
	AfterUnknown map[string]interface{} `json:"after_unknown"` // Undocumented :( See https://github.com/terraform-providers/terraform-provider-aws/issues/11823
}

// Configuration represents part of the configuration block of a plan.
// https://www.terraform.io/docs/internals/json-format.html#configuration-representation
type Configuration struct {
	ProviderConfig map[string]ProviderConfig `json:"provider_config"`
	RootModule     struct {
		// Note: This is not the same schema as the planned value resource above.
		Resources []struct {
			Address           string      `json:"address"`
			Kind              string      `json:"type"`
			Name              string      `json:"name"`
			ProviderConfigKey string      `json:"provider_config_key"`
			Expressions       expressions `json:"expressions"`
		} `json:"resources"`
	} `json:"root_module"`
}

// ProviderConfig represents a single provider configuration from the Configuration block of a Terraform plan.
type ProviderConfig struct {
	Name              string      `json:"name"`
	VersionConstraint string      `json:"version_constraint,omitempty"`
	Alias             string      `json:"alias,omitempty"`
	Expressions       expressions `json:"expressions"`
}

type expressions map[string]interface{}

// ReadPlanResources unmarshal json data to go struct and returns array of all Resource from root and child modules.
func ReadPlanResources(data []byte) ([]Resource, error) {
	p := new(plan)
	if err := json.Unmarshal(data, p); err != nil {
		return nil, err
	}
	var result []Resource
	for _, resource := range p.PlannedValues.RootModules.Resources {
		result = append(result, resource)
	}
	for _, module := range p.PlannedValues.RootModules.ChildModules {
		result = append(result, resourcesFromModule(&module, 0)...)
	}
	return result, nil
}

// ReadStateResources unmarshal json data to go struct and returns array of all Resource from root and child modules.
func ReadStateResources(data []byte) ([]Resource, error) {
	s := new(state)
	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}
	var result []Resource
	for _, resource := range s.Values.RootModules.Resources {
		result = append(result, resource)
	}
	for _, module := range s.Values.RootModules.ChildModules {
		result = append(result, resourcesFromModule(&module, 0)...)
	}
	return result, nil
}

// resourcesFromModule returns array of all Resource from the module recursively.
func resourcesFromModule(module *childModule, level int) []Resource {
	if level > maxChildModuleLevel {
		log.Printf("The configuration has more than %d level of modules. Modules with a depth more than %d will be ignored.", maxChildModuleLevel, maxChildModuleLevel)
		return nil
	}
	var result []Resource
	for _, resource := range module.Resources {
		result = append(result, resource)
	}
	for _, c := range module.ChildModules {
		result = append(result, resourcesFromModule(&c, level+1)...)
	}
	return result
}

// ReadPlanChanges unmarshals b into a jsonPlan and returns the array of ResourceChange from it.
// If actions is not "", will only return changes where one of the specified actions will be taken.
func ReadPlanChanges(data []byte, actions []string) ([]ResourceChange, error) {
	p := new(plan)
	if err := json.Unmarshal(data, p); err != nil {
		return nil, err
	}
	var result []ResourceChange
	for _, rc := range p.ResourceChanges {
		if len(actions) == 0 || slicesEqual(rc.Change.Actions, actions) {
			result = append(result, rc)
		}
	}

	return result, nil
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ReadProviderConfigValues returns the values from the expressions block from the provider config for the resource with the given kind and name.
// Variable references are resolved, as are constant_value.
func ReadProviderConfigValues(data []byte, kind, name string) (map[string]interface{}, error) {
	p := new(plan)
	if err := json.Unmarshal(data, p); err != nil {
		return nil, err
	}

	config, err := resourceProviderConfig(kind, name, p)
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})

	// Resolve expressions.
	for k, v := range config.Expressions {
		// Within a provider config, we expect expressions to be maps, but that's not guaranteed, so don't type assert.
		switch mv := v.(type) {
		case map[string]interface{}:
			result[k] = resolveExpression(mv, p)
		}
	}

	return result, nil
}

func resourceProviderConfig(kind string, name string, plan *plan) (ProviderConfig, error) {
	// Find the provider_config_key for this resource.
	for _, r := range plan.Configuration.RootModule.Resources {
		if r.Kind == kind && r.Name == name {
			// Find the right provider config based on the provider_config_key
			pc, ok := plan.Configuration.ProviderConfig[r.ProviderConfigKey]
			if !ok {
				return ProviderConfig{}, fmt.Errorf("Could not provider config for key %q", r.ProviderConfigKey)
			}
			return pc, nil
		}
	}

	return ProviderConfig{}, fmt.Errorf("Could not find provider config key for resource %v.%v", kind, name)
}

func resolveExpression(expr map[string]interface{}, plan *plan) interface{} {
	if expr == nil {
		return nil
	}

	if cv, ok := expr["constant_value"]; ok {
		return cv
	}
	if refs, ok := expr["references"]; ok {
		switch stringRefs := refs.(type) {
		case []interface{}:
			for _, ref := range stringRefs {
				if resolved := resolveReference(ref.(string), plan); resolved != nil {
					// Take the first one. Not sure if this is 100% correct.
					return resolved
				}
			}
		}
	}

	// Can't resolve, return expression value as-is.
	return expr
}

// resolveReference resolves expressions within "references" blocks of a Terraform plan to their specific values.
// At the moment, it only handles "var.XXX", for references to variables.
// The docs say not to parse the strings, but I don't see anywhere in the plan where these are directly used as keys, so I'm parsing them as strings.
// https://www.terraform.io/docs/configuration/expressions.html#references-to-named-values
// TODO: Find out how to canonically resolve all types of references and expressions.
func resolveReference(expr string, plan *plan) interface{} {
	exprParts := strings.Split(expr, ".")
	if exprParts[0] == "var" {
		// Variable.
		varName := exprParts[1]
		if v, ok := plan.Variables[varName]; ok {
			return v.Value
		}
	}

	return nil
}
