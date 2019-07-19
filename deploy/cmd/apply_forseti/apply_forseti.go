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
)

func main() {
	flag.Parse()

	if *projectYAMLPath == "" {
		log.Fatal("--project_yaml_path must be set")
	}

	conf, err := config.Load(*projectYAMLPath, *generatedFieldsPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if err := apply.Forseti(conf); err != nil {
		log.Fatalf("failed to apply forseti: %v", err)
	}
}
