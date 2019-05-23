// Rule_generator provides a CLI to generate Forseti rules for the projects in the projects yaml file.
//
// Usage:
//   $ bazel run :rule_generator -- --project_yaml_path=${PROJECTS_YAML_PATH?} --output_path=${OUTPUT_PATH}
package main

import (
	"io/ioutil"
	"log"

	"flag"
	
	"github.com/GoogleCloudPlatform/healthcare/deploy/cft"
	"github.com/GoogleCloudPlatform/healthcare/deploy/rulegen"
	"github.com/ghodss/yaml"
)

var (
	projectYAMLPath = flag.String("project_yaml_path", "", "Path to projects yaml file")
	outputPath      = flag.String("output_path", "",
		"Path to local directory or GCS bucket to write forseti rules. "+
			"If unset, directly writes to the Forseti server bucket")
)

func main() {
	flag.Parse()

	if *projectYAMLPath == "" {
		log.Fatal("--project_yaml_path must be set")
	}

	b, err := ioutil.ReadFile(*projectYAMLPath)
	if err != nil {
		log.Fatalf("failed to read input projects yaml file at path %q: %v", *projectYAMLPath, err)
	}

	conf := new(cft.Config)
	if err := yaml.Unmarshal(b, conf); err != nil {
		log.Fatalf("failed to unmarshal config: %v", err)
	}

	if err := conf.Init(); err != nil {
		log.Fatalf("failed to init config: %v", err)
	}

	if err := rulegen.Run(conf, *outputPath); err != nil {
		log.Fatal(err)
	}

	log.Println("Rule generation successful")
}
