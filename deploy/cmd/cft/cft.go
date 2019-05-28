// CFT provides a CLI to deploy CFT definitions for a project in a projects yaml file.
//
// Usage:
//   $ bazel run :cft -- --project_yaml_path=${PROJECT_YAML_PATH?} --project=${PROJECT_ID?}
package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"flag"
	
	"github.com/GoogleCloudPlatform/healthcare/deploy/apply"
	"github.com/GoogleCloudPlatform/healthcare/deploy/cft"
	"github.com/ghodss/yaml"
)

var (
	projectYAMLPath = flag.String("project_yaml_path", "", "Path to project yaml file")
	projectID       = flag.String("project", "", "Project within the project yaml file to deploy CFT resources for")
)

func main() {
	flag.Parse()

	if *projectYAMLPath == "" {
		log.Fatal("--project_yaml_path must be set")
	}
	if *projectID == "" {
		log.Fatal("--project must be set")
	}

	// TODO: handle split yaml configs
	b, err := ioutil.ReadFile(*projectYAMLPath)
	if err != nil {
		log.Fatalf("failed to read input projects yaml file at path %q: %v", *projectYAMLPath, err)
	}

	conf := new(cft.Config)
	if err := yaml.Unmarshal(b, conf); err != nil {
		log.Fatalf("failed to unmarshal config: %v", err)
	}
	if err := conf.Init(); err != nil {
		log.Fatalf("failed to initialize config: %v", err)
	}

	proj, err := findProject(*projectID, conf)
	if err != nil {
		log.Fatal(err)
	}

	if err := apply.Apply(conf, proj); err != nil {
		log.Fatalf("failed to deploy %q resources: %v", *projectID, err)
	}

	log.Println("CFT deployment successful")
}

func findProject(id string, c *cft.Config) (*cft.Project, error) {
	for _, p := range c.AllProjects() {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, fmt.Errorf("failed to find project %q", id)
}
