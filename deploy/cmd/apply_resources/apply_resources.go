/*
 * Copyright 2019 Google LLC.
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

// apply_resources provides a CLI to deploy a project config to GCP.
//
// Usage:
//   $ bazel run :apply_resources -- --project_yaml_path=${PROJECT_YAML_PATH?} --project=${PROJECT_ID?}
package main

import (
	"fmt"
	"log"

	"flag"
	
	"github.com/GoogleCloudPlatform/healthcare/deploy/apply"
	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
)

var (
	projectYAMLPath     = flag.String("project_yaml_path", "", "Path to project yaml file")
	generatedFieldsPath = flag.String("generated_fields_path", "", "Path to generated fields yaml file")
	projectID           = flag.String("project", "", "Project within the project yaml file to deploy config resources for")
	enableTerraform     = flag.Bool("enable_terraform", false, "DEV ONLY. Whether terraform is preferred over deployment manager.")
)

func main() {
	flag.Parse()

	if *projectYAMLPath == "" {
		log.Fatal("--project_yaml_path must be set")
	}
	if *projectID == "" {
		log.Fatal("--project must be set")
	}

	conf, err := config.Load(*projectYAMLPath, *generatedFieldsPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	proj, err := findProject(*projectID, conf)
	if err != nil {
		log.Fatal(err)
	}

	opts := &apply.Options{EnableTerraform: *enableTerraform}
	if err := apply.DeployResources(conf, proj, opts); err != nil {
		log.Fatalf("failed to deploy %q resources: %v", *projectID, err)
	}

	log.Println("Config deployed successfully")
}

func findProject(id string, c *config.Config) (*config.Project, error) {
	for _, p := range c.AllProjects() {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, fmt.Errorf("failed to find project %q", id)
}
