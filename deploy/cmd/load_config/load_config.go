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

// Load_config prints the merged, parsed and validated config to stdout.
package main

import (
	"fmt"
	"log"

	"flag"
	
	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
)

var (
	projectYAMLPath     = flag.String("project_yaml_path", "", "Path to project yaml file")
	generatedFieldsPath = flag.String("generated_fields_path", "", "Path to generated fields yaml file")
	enableTerraform     = flag.Bool("enable_terraform", false, "DEV ONLY. Enable terraform.")
)

func main() {
	flag.Parse()

	if *projectYAMLPath == "" {
		log.Fatal("--project_yaml_path must be set")
	}

	config.EnableTerraform = *enableTerraform
	b, err := config.LoadBytes(*projectYAMLPath)
	if err != nil {
		log.Fatalf("failed to load config to bytes: %v", err)
	}

	if *generatedFieldsPath != "" {
		if _, err := config.LoadGeneratedFields(*generatedFieldsPath); err != nil {
			log.Fatalf("failed to validate generated fields yaml: %v", err)
		}
	}

	fmt.Println(string(b))
}
