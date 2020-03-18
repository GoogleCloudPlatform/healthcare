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
// TODO: move to a common location that can be shared with Terraform Importer.
package terraform

import (
	"encoding/json"
	"log"
)

const maxChildModuleLevel = 100

// https://www.terraform.io/docs/internals/json-format.html#plan-representation
type plan struct {
	PlannedValues values `json:"planned_values"`
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
