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
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"flag"
	
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
		in := filepath.Join(root, conf.TemplatesDir, t.InputDir)
		out := filepath.Join(outputDir, t.OutputDir)
		if err := mergo.Merge(&t.Data, conf.Data); err != nil {
			return err
		}
		if err := writeTemplate(in, out, t.Data); err != nil {
			return err
		}
	}
	return nil
}

func writeTemplate(in, out string, data map[string]interface{}) error {
	fs, err := ioutil.ReadDir(in)
	if err != nil {
		return fmt.Errorf("failed to read dir %q: %v", in, err)
	}

	for _, f := range fs {
		in := filepath.Join(in, f.Name())

		var buf bytes.Buffer
		nameTmpl, err := template.New(in).Option("missingkey=error").Parse(filepath.Join(out, f.Name()))
		if err != nil {
			return err
		}
		if err := nameTmpl.Execute(&buf, data); err != nil {
			return err
		}
		out := buf.String()

		if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
			return fmt.Errorf("failed to mkdir %q: %v", out, err)
		}

		if f.IsDir() {
			if err := writeTemplate(in, out, data); err != nil {
				return err
			}
			continue
		}

		b, err := ioutil.ReadFile(in)
		if err != nil {
			return fmt.Errorf("failed to read %q: %v", in, err)
		}

		tmpl, err := template.New(in).Option("missingkey=error").Parse(string(b))
		if err != nil {
			return fmt.Errorf("failed to parse template %q: %v", in, err)
		}

		outFile, err := os.OpenFile(out, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer outFile.Close()

		if err := tmpl.Execute(outFile, data); err != nil {
			return fmt.Errorf("failed to execute template %q: %v", in, err)
		}
	}
	return nil
}
