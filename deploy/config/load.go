/*
 * Copyright 2019 Google LLC.
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

package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/ghodss/yaml"
	"github.com/xeipuuv/gojsonschema"
	"github.com/imdario/mergo"
	"github.com/mitchellh/go-homedir"
)

// projectConfigSchema is the path of the project config schema template relative to the repo root.
const projectConfigSchema = "project_config.yaml.schema"

// generatedFieldsSchema is the path of the generated fields schema template relative to the repo root.
const generatedFieldsSchema = "generated_fields.yaml.schema"

// NormalizePath normalizes paths specified through a local run or Bazel invocation.
func NormalizePath(path string) (string, error) {
	path, err := homedir.Expand(path)
	if err != nil {
		return "", err
	}
	path = os.ExpandEnv(path)
	if strings.HasPrefix(path, "gs://") || filepath.IsAbs(path) {
		return path, nil
	}
	// Path is relative from where the script was launched from.
	// When using `bazel run`, the environment variable BUILD_WORKING_DIRECTORY
	// will be set to the path where the command was run from.
	cwd := os.Getenv("BUILD_WORKING_DIRECTORY")
	if cwd == "" {
		if cwd, err = os.Getwd(); err != nil {
			return "", err
		}
	}
	return filepath.Abs(filepath.Join(cwd, path))
}

// Load loads a config from the given path.
func Load(confPath, genFieldsPath string) (*Config, error) {
	confPath, err := NormalizePath(confPath)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize path %q: %v", confPath, err)
	}

	m, err := loadMap(confPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load config to map: %v", err)
	}

	b, err := yaml.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config map: %v", err)
	}

	if err := ValidateConf(b); err != nil {
		return nil, err
	}

	conf := new(Config)
	if err := yaml.Unmarshal(b, conf); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v\nmerged config: %v", err, string(b))
	}

	// TODO: remove this check once conf.GeneratedFieldsPath is mandatory.
	if conf.GeneratedFieldsPath != "" {
		genFieldsPath = conf.GeneratedFieldsPath
	}

	if genFieldsPath == "" {
		return nil, errors.New("generated fields path is neither specified in config nor command line")
	}

	if genFieldsPath, err = NormalizePath(genFieldsPath); err != nil {
		return nil, fmt.Errorf("failed to normalize path %q: %v", genFieldsPath, err)
	}

	if genFieldsPath == confPath {
		return nil, errors.New("generated fields path cannot be set to the same as config path")
	}
	conf.GeneratedFieldsPath = genFieldsPath

	genFields, err := loadGeneratedFields(genFieldsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load generated fields: %v", err)
	}
	if err := conf.Init(genFields); err != nil {
		return nil, fmt.Errorf("failed to initialize config: %v", err)
	}
	return conf, nil
}

// ValidateConf validates the input project config against the default schema template.
func ValidateConf(confYAML []byte) error {
	return validate(confYAML, projectConfigSchema)
}

// validateGenFields validates the generated fields config against the default schema template.
func validateGenFields(genFieldsYAML []byte) error {
	return validate(genFieldsYAML, generatedFieldsSchema)
}

// validate validates the input yaml against the given schema template.
func validate(inputYAML []byte, schemaPath string) error {
	schemaYAML, err := ioutil.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file at path %q: %v", schemaPath, err)
	}
	schemaJSON, err := yaml.YAMLToJSON(schemaYAML)
	if err != nil {
		return fmt.Errorf("failed to convert schema file at path %q from yaml to json: %v", schemaPath, err)
	}
	confJSON, err := yaml.YAMLToJSON(inputYAML)
	if err != nil {
		return fmt.Errorf("failed to convert config file bytes from yaml to json: %v", err)
	}

	result, err := gojsonschema.Validate(gojsonschema.NewBytesLoader(schemaJSON), gojsonschema.NewBytesLoader(confJSON))
	if err != nil {
		return fmt.Errorf("failed to validate config: %v", err)
	}

	if len(result.Errors()) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("config has validation errors:")
	for _, err := range result.Errors() {
		sb.WriteString(fmt.Sprintf("\n- %v", err))
	}
	return errors.New(sb.String())
}

type importsItem struct {
	Path string                 `json:"path"`
	Data map[string]interface{} `json:"data"`

	Pattern string `json:"pattern"`
}

// loadMap loads the config at path into a map. It will also merge all imported configs.
// The given path should be absolute.
// TODO: add check for certain attributes, e.g. generated_fields_path, cannot be specified more than once.
func loadMap(path string, data map[string]interface{}) (map[string]interface{}, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file at path %q: %v", path, err)
	}

	if len(data) > 0 {
		tmpl, err := template.New(path).Option("missingkey=error").Parse(string(b))
		if err != nil {
			return nil, fmt.Errorf("failed to parse %q into template: %v", path, err)
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return nil, fmt.Errorf("failed to execute template for %q: %v", path, err)
		}
		b = buf.Bytes()
	}

	root := make(map[string]interface{})
	if err := json.Unmarshal(b, &root); err != nil {
		return nil, fmt.Errorf("failed to unmarshal raw config to map at path %q: %v", path, err)
	}

	type config struct {
		Imports             []*importsItem `json:"imports"`
		GeneratedFieldsPath string         `json:"generated_fields_path"`
	}
	conf := new(config)
	if err := json.Unmarshal(b, conf); err != nil {
		return nil, fmt.Errorf("failed to unmarshal raw config to struct with imports at path %q: %v", path, err)
	}

	dir := filepath.Dir(path)
	if conf.GeneratedFieldsPath != "" {
		if filepath.IsAbs(conf.GeneratedFieldsPath) {
			return nil, errors.New("generated fields path from config cannot be absolute")
		}
		root["generated_fields_path"] = filepath.Join(dir, conf.GeneratedFieldsPath)
	}

	pathMap := map[string]bool{
		path: true,
	}
	for _, imp := range conf.Imports {
		impPath := imp.Path
		if impPath == "" {
			continue
		}
		if !filepath.IsAbs(impPath) {
			impPath = filepath.Join(dir, impPath)
		}
		pathMap[impPath] = true

		impMap, err := loadMap(impPath, imp.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to load %q to map: %v", impPath, err)
		}
		if err := mergo.Merge(&root, impMap, mergo.WithAppendSlice); err != nil {
			return nil, fmt.Errorf("failed to merge imported file %q: %v", impPath, err)
		}
	}

	paths, err := patternPaths(path, conf.Imports)
	if err != nil {
		return nil, err
	}

	for _, p := range paths {
		if pathMap[p] {
			continue
		}
		impMap, err := loadMap(p, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to load %q to map: %v", p, err)
		}
		if err := mergo.Merge(&root, impMap, mergo.WithAppendSlice); err != nil {
			return nil, fmt.Errorf("failed to merge imported file %q: %v", p, err)
		}
	}
	return root, nil
}

// patternPaths returns all files matching the patterns defined
// in importsList.
// If projectYAMLPath match patterns, the result always ignore it.
// projectYAMLPath should be an absolute path.
// Patterns in importsList could be relative path to the projectYAMLPath
// or absolute paths.
// For example, if "./*.yaml" is an entry of "imports", the project YAML itself
// would match the pattern. We should exclude that path because we do not want to
// include the content of that YAML twice.
func patternPaths(projectYAMLPath string, importsList []*importsItem) ([]string, error) {
	allMatches := make(map[string]bool)
	projectYamlFolder := filepath.Dir(projectYAMLPath)
	for _, importItem := range importsList {
		// joinedPath would be always an absolute path (pattern).
		joinedPath := importItem.Pattern
		if joinedPath == "" {
			continue
		}
		if len(importItem.Data) > 0 {
			return nil, fmt.Errorf("import cannot have both pattern and data set together")
		}
		if !filepath.IsAbs(joinedPath) {
			joinedPath = filepath.Join(projectYamlFolder, importItem.Pattern)
		}
		matches, err := filepath.Glob(joinedPath)
		if err != nil {
			return nil, fmt.Errorf("pattern %q is malformed", importItem.Pattern)
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("pattern %q matched no files", importItem.Pattern)
		}
		for _, match := range matches {
			if match == projectYAMLPath {
				continue
			}
			allMatches[match] = true
		}
	}
	var filePathList []string
	for path := range allMatches {
		filePathList = append(filePathList, path)
	}
	return filePathList, nil
}

// loadGeneratedFields loads and validates generated fields from yaml file at path.
func loadGeneratedFields(path string) (*AllGeneratedFields, error) {
	// Create an empty file (and parent directories, if any) if not exist.
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create directory: %v", err)
	}
	if _, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0666); err != nil {
		return nil, fmt.Errorf("failed to create an empty generated fields file: %v\nnote: if you hit this error in a test, please create an empty generated fields file manually and add it as a test dependency", err)
	}
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file at path %q: %v", path, err)
	}
	if len(b) == 0 {
		return nil, nil
	}
	if err := validateGenFields(b); err != nil {
		return nil, fmt.Errorf("failed to validate generated fields at path %q: %v", path, err)
	}
	genFields := new(AllGeneratedFields)
	if err := yaml.UnmarshalStrict(b, genFields, yaml.DisallowUnknownFields); err != nil {
		return nil, fmt.Errorf("failed to unmarshal generated fields at path %q: %v", path, err)
	}
	return genFields, nil
}

// DumpGeneratedFields dumps generated fields to file at path.
func DumpGeneratedFields(generatedFields *AllGeneratedFields, path string) error {
	path, err := NormalizePath(path)
	if err != nil {
		return fmt.Errorf("failed to normalize path %q: %v", path, err)
	}
	b, err := yaml.Marshal(generatedFields)
	if err != nil {
		return fmt.Errorf("failed to marshal generated fields: %v", err)
	}
	content := []byte("# This is an auto-generated file and should not be modified manually.\n")
	content = append(content, b...)
	if err := ioutil.WriteFile(path, content, 0666); err != nil {
		return fmt.Errorf("failed to write file at path %q: %v", path, err)
	}
	return nil
}
