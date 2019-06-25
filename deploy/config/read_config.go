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
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/mitchellh/go-homedir"
)

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
	if len(cwd) == 0 {
		if cwd, err = os.Getwd(); err != nil {
			return "", err
		}
	}
	return filepath.Abs(filepath.Join(cwd, path))
}

// ReadConfig read a project YAML and imports other specified files.
// projectYAMLPath should be an absolute path.
func ReadConfig(projectYAMLPath string) (*Config, map[string]*Config) {
	importedConfigPathMap := make(map[string]*Config)
	conf := readSingleConfigFile(projectYAMLPath)
	importedYAMLPaths := allImportsPaths(projectYAMLPath, conf.Imports)
	for _, importedYAMLPath := range importedYAMLPaths {
		conf := readSingleConfigFile(importedYAMLPath)
		importedConfigPathMap[importedYAMLPath] = conf
	}
	// We support projects in the imported YAMLs only.
	mergeProjects(projectYAMLPath, conf, importedConfigPathMap)
	return conf, importedConfigPathMap
}

// allImportsPaths returns all files matching the patterns defined
// in importsList.
// If projectYAMLPath match patterns, the result always ignore it.
// projectYAMLPath should be an absolute path.
// Patterns in importsList could be relative path to the projectYAMLPath
// or absolute paths.
// For example, if "./*.yaml" is an entry of "imports", the project YAML itself
// would match the pattern. We should exclude that path because we do not want to
// include the content of that YAML twice.
func allImportsPaths(projectYAMLPath string, importsList []string) []string {
	allMatches := make(map[string]bool)
	projectYamlFolder := filepath.Dir(projectYAMLPath)
	for _, importPath := range importsList {
		// joinedPath would be always an absolute path (pattern).
		joinedPath := importPath
		if !filepath.IsAbs(joinedPath) {
			joinedPath = filepath.Join(projectYamlFolder, importPath)
		}
		matches, err := filepath.Glob(joinedPath)
		if err != nil {
			log.Fatalf("Pattern \"%v\" is malformed", importPath)
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
	sort.Strings(filePathList)
	return filePathList
}

// readSingleConfigFile read the YAML to get its Config.
func readSingleConfigFile(absYAMLPath string) *Config {
	b, err := ioutil.ReadFile(absYAMLPath)
	if err != nil {
		log.Fatalf("failed to read input projects yaml file at path %q: %v", absYAMLPath, err)
	}

	conf := new(Config)
	if err := yaml.Unmarshal(b, conf); err != nil {
		log.Fatalf("failed to unmarshal config: %v", err)
	}

	if conf.Projects == nil {
		conf.Projects = []*Project{}
	}
	return conf
}

// recordProjectsPaths build the map from the project ID to the path where the project is defined.
func recordProjectsPaths(importedProjects map[string]([]string), confPath string, Projects []*Project) {
	for _, projects := range Projects {
		if paths, ok := importedProjects[projects.ID]; ok {
			importedProjects[projects.ID] = append(paths, confPath)
		} else {
			importedProjects[projects.ID] = []string{confPath}
		}
	}
}

// mergeProjects merges all YAMLs into a single "dest" YAML.
func mergeProjects(projectYAMLPath string, dest *Config, importedConfigPathMap map[string]*Config) {
	importedProjects := make(map[string]([]string))
	recordProjectsPaths(importedProjects, projectYAMLPath, dest.Projects)
	for path, pathConfPair := range importedConfigPathMap {
		recordProjectsPaths(importedProjects, path, pathConfPair.Projects)
		dest.Projects = append(dest.Projects, pathConfPair.Projects...)
	}
	// Check duplicate projects
	duplicateProjects := false
	for projectID, pathList := range importedProjects {
		if len(pathList) > 1 {
			duplicateProjects = true
			fmt.Printf("%q appears in multiple YAMLs: %v\n", projectID, pathList)
		}
	}
	if duplicateProjects {
		log.Fatal("Projects defined in multiple YAMLs.")
	}
}
