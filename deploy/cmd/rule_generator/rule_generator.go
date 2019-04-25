// Rule_generator provides a CLI to generate Forseti rules for the projects in the projects yaml file.
//
// Usage:
//   $ bazel run :rule_generator -- --projects_yaml_path=${PROJECTS_YAML_PATH?}
package main

import (
	"io/ioutil"
	"log"

	"flag"
	
	"github.com/GoogleCloudPlatform/healthcare/deploy/cft"
	"github.com/GoogleCloudPlatform/healthcare/deploy/rulegen"
	"github.com/ghodss/yaml"
)

var projectsYAMLPath = flag.String("projects_yaml_path", "", "Path to projects yaml file")

func main() {
	flag.Parse()

	if *projectsYAMLPath == "" {
		log.Fatal("--projects_yaml_path must be set")
	}

	b, err := ioutil.ReadFile(*projectsYAMLPath)
	if err != nil {
		log.Fatalf("failed to read input projects yaml file at path %q: %v", *projectsYAMLPath, err)
	}

	conf := new(cft.Config)
	if err := yaml.Unmarshal(b, conf); err != nil {
		log.Fatalf("failed to unmarshal config: %v", err)
	}

	if err := conf.Init(); err != nil {
		log.Fatal(err)
	}

	if err := rulegen.Run(conf); err != nil {
		log.Fatal(err)
	}

	log.Println("Rule generation successful")
}
