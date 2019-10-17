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
	"log"

	"flag"
	
	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/ghodss/yaml"
)

var (
	configPath      = flag.String("config_path", "", "Path to project config file")
	enableTerraform = flag.Bool("enable_terraform", false, "DEV ONLY. Enable terraform.")
)

func main() {
	flag.Parse()

	if *configPath == "" {
		log.Fatal("--config_path must be set")
	}

	config.EnableTerraform = *enableTerraform
	c, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config to bytes: %v", err)
	}

	b, err := yaml.Marshal(c)
	if err != nil {
		log.Fatalf("failed to marshal config: %v", err)
	}
	log.Printf("Successfully loaded config:\n%s", string(b))
	b, err = yaml.Marshal(c.AllGeneratedFields)
	if err != nil {
		log.Fatalf("failed to marshal generated fields: %v", err)
	}
	log.Printf("Successfully loaded generated fields:\n%s", string(b))
}
