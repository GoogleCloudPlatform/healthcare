/*
 * Copyright 2020 Google LLC.
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

// terraform_engine provides an engine to power your Terraform environment.
//
// Usage:
// $ bazel run :main -- --config_path=/absolute/path/to/config --output_dir=/tmp/engine
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"flag"
	
	"github.com/GoogleCloudPlatform/healthcare/deploy/template"
	"github.com/ghodss/yaml"
	"github.com/imdario/mergo"
)

var (
	configPath = flag.String("config_path", "", "Path to config")
	outputDir  = flag.String("output_dir", "", "Path to dump output")
)

type config struct {
	TemplatesDir string                 `json:"templates_dir"`
	Data         map[string]interface{} `json:"data"`
	Templates    []*struct {
		InputDir  string                 `json:"input_dir"`
		OutputDir string                 `json:"output_dir"`
		Data      map[string]interface{} `json:"data"`
	} `json:"templates"`
}

func main() {
	flag.Parse()

	if *configPath == "" {
		log.Fatal("--config_path must be set")
	}
	if *outputDir == "" {
		log.Fatal("--outputdir must be set")
	}

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	b, err := ioutil.ReadFile(*configPath)
	if err != nil {
		return err
	}
	c := new(config)
	if err := yaml.Unmarshal(b, c); err != nil {
		return err
	}

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	if err := dump(c, filepath.Dir(*configPath), tmpDir); err != nil {
		return err
	}

	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		return fmt.Errorf("failed to mkdir %q: %v", *outputDir, err)
	}

	fs, err := filepath.Glob(filepath.Join(tmpDir, "/*"))
	if err != nil {
		return err
	}
	cp := exec.Command("cp", append([]string{"-a", "-t", *outputDir}, fs...)...)
	cp.Stderr = os.Stderr
	return cp.Run()
}

func dump(conf *config, root string, outputDir string) error {
	for _, t := range conf.Templates {
		if err := mergo.Merge(&t.Data, conf.Data); err != nil {
			return err
		}
		out, err := template.WriteString(filepath.Join(outputDir, t.OutputDir), t.Data)
		if err != nil {
			return err
		}
		in := filepath.Join(root, conf.TemplatesDir, t.InputDir)
		if err := template.WriteDir(in, out, t.Data); err != nil {
			return err
		}
	}
	return nil
}
