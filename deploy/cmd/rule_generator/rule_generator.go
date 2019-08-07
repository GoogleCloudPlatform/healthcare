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

// Rule_generator provides a CLI to generate Forseti rules for the projects in the projects yaml file.
//
// Usage:
//   $ bazel run :rule_generator -- --project_yaml_path=${PROJECTS_YAML_PATH?} --output_path=${OUTPUT_PATH}
package main

import (
	"log"

	"flag"
	
	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/GoogleCloudPlatform/healthcare/deploy/rulegen"
)

var (
	projectYAMLPath     = flag.String("project_yaml_path", "", "Path to projects yaml file")
	generatedFieldsPath = flag.String("generated_fields_path", "", "Path to generated fields yaml file")
	outputPath          = flag.String("output_path", "",
		"Path to local directory or GCS bucket to write forseti rules. "+
			"If unset, directly writes to the Forseti server bucket")
)

func main() {
	flag.Parse()

	if *projectYAMLPath == "" {
		log.Fatal("--project_yaml_path must be set")
	}
	if *generatedFieldsPath == "" {
		log.Fatal("--generated_fields_path must be set")
	}

	conf, err := config.Load(*projectYAMLPath, *generatedFieldsPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if err := rulegen.Run(conf, *outputPath); err != nil {
		log.Fatal(err)
	}

	log.Println("Rule generation successful")
}
