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
//   $ bazel run :rule_generator -- --config_path=${CONFIG_PATH?} --output_path=${OUTPUT_PATH}
package main

import (
	"log"

	"flag"
	
	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/GoogleCloudPlatform/healthcare/deploy/rulegen"
	"github.com/GoogleCloudPlatform/healthcare/deploy/runner"
)

var (
	configPath = flag.String("config_path", "", "Path to project config file")
	outputPath = flag.String("output_path", "",
		"Path to local directory or GCS bucket to write forseti rules. "+
			"If unset, directly writes to the Forseti server bucket")
	auditConfig = flag.String("audit_config", "audit_config.yaml",
		"Audit config file name")
)

func main() {
	flag.Parse()

	if *configPath == "" {
		log.Fatal("--config_path must be set")
	}

	conf, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if err := rulegen.Run(conf, *outputPath, &runner.Default{}); err != nil {
		log.Fatal(err)
	}

	log.Println("Rule generation successful")
}
