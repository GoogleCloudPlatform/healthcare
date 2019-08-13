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

// apply_forseti installs forseti in the forseti project.
// TODO: once forseti can be updated apply it in apply binary and remove this.
package main

import (
	"log"

	"flag"
	
	"github.com/GoogleCloudPlatform/healthcare/deploy/apply"
	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
)

var (
	projectYAMLPath     = flag.String("project_yaml_path", "", "Path to project yaml file")
	generatedFieldsPath = flag.String("generated_fields_path", "", "Path to generated fields yaml file")
	enableRemoteState   = flag.Bool("enable_remote_state", false, "DEV ONLY. Enable remote state.")
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

	if err := apply.ForsetiConfig(conf, *enableRemoteState); err != nil {
		log.Fatalf("failed to apply forseti: %v", err)
	}
}
