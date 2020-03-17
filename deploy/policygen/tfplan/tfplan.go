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

// Package tfplan provides definitions of Terraform plan in JSON format.
package tfplan

import (
	"encoding/json"
	"log"
)

const maxChildModuleLevel = 100

// jsonPlan represents Terraform 0.12 plan exported in json format by 'terraform show -json plan.tfplan' command.
// https://www.terraform.io/docs/internals/json-format.html#values-representation
// TODO: move to a common location that can be shared with Terraform Importer.
type jsonPlan struct {
	PlannedValues struct {
		RootModules struct {
			Resources    []JSONResource `json:"resources"`
			ChildModules []childModule  `json:"child_modules"`
		} `json:"root_module"`
	} `json:"planned_values"`
}

type childModule struct {
	Address      string         `json:"address"`
	Resources    []JSONResource `json:"resources"`
	ChildModules []childModule  `json:"child_modules"`
}

// JSONResource represent single Terraform resource definition.
type JSONResource struct {
	Name    string                 `json:"name"`
	Address string                 `json:"address"`
	Kind    string                 `json:"type"`
	Mode    string                 `json:"mode"` // "managed" for resources, or "data" for data resources
	Values  map[string]interface{} `json:"values"`
}

// ReadJSONResources unmarshal json data to go struct and returns array of all JSONResource from root and child modules.
func ReadJSONResources(data []byte) ([]JSONResource, error) {
	plan := jsonPlan{}
	err := json.Unmarshal(data, &plan)
	if err != nil {
		return nil, err
	}
	var result []JSONResource
	for _, resource := range plan.PlannedValues.RootModules.Resources {
		result = append(result, resource)
	}
	for _, module := range plan.PlannedValues.RootModules.ChildModules {
		result = append(result, resourcesFromModule(&module, 0)...)
	}
	return result, nil
}

// resourcesFromModule returns array of all JSONResource from the module recursively.
func resourcesFromModule(module *childModule, level int) []JSONResource {
	if level > maxChildModuleLevel {
		log.Printf("The configuration has more than %d level of modules. Modules with a depth more than %d will be ignored.", maxChildModuleLevel, maxChildModuleLevel)
		return nil
	}
	var result []JSONResource
	for _, resource := range module.Resources {
		result = append(result, resource)
	}
	for _, c := range module.ChildModules {
		result = append(result, resourcesFromModule(&c, level+1)...)
	}
	return result
}
