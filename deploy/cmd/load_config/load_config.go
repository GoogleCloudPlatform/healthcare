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
)

func main() {
	flag.Parse()

	if *projectYAMLPath == "" {
		log.Fatal("--project_yaml_path must be set")
	}

	b, err := config.LoadBytes(*projectYAMLPath)
	if err != nil {
		log.Fatalf("failed to load config to bytes: %v", err)
	}

	// TODO: remove this check after --generated_fields_path becomes mandatory.
	if *generatedFieldsPath != "" {
		if _, err := config.LoadGeneratedFields(*generatedFieldsPath, *generatedFieldsPath == *projectYAMLPath); err != nil {
			log.Fatalf("failed to validate generated fields yaml: %v", err)
		}
	}

	fmt.Println(string(b))
}
