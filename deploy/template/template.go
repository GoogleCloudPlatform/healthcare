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

// Package template provides utility functions around reading and writing templates.
package template

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"
)

// WriteDir generates files to directory `outputDir` based on templates from directory `inputDir` and `data`.
func WriteDir(inputDir, outputDir string, data map[string]interface{}) error {
	fs, err := ioutil.ReadDir(inputDir)
	if err != nil {
		return fmt.Errorf("read dir %q: %v", inputDir, err)
	}

	for _, f := range fs {
		in := filepath.Join(inputDir, f.Name())
		out := filepath.Join(outputDir, f.Name())

		if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
			return fmt.Errorf("mkdir %q: %v", filepath.Dir(out), err)
		}

		if f.IsDir() {
			if err := WriteDir(in, out, data); err != nil {
				return err
			}
			continue
		}

		b, err := ioutil.ReadFile(in)
		if err != nil {
			return fmt.Errorf("read %q: %v", in, err)
		}

		tmpl, err := template.New(in).Option("missingkey=error").Parse(string(b))
		if err != nil {
			return fmt.Errorf("parse template %q: %v", in, err)
		}

		outFile, err := os.OpenFile(out, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer outFile.Close()

		if err := tmpl.Execute(outFile, data); err != nil {
			return fmt.Errorf("execute template %q: %v", in, err)
		}
	}
	return nil
}

// WriteString generates a string with template `text` filled with values from `data`.
func WriteString(text string, data map[string]interface{}) (string, error) {
	var buf bytes.Buffer
	tmpl, err := template.New("").Option("missingkey=error").Parse(text)
	if err != nil {
		return "", err
	}
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
