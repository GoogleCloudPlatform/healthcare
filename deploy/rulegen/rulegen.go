// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package rulegen provides Forseti rule generation utilities.
// Note: rules in this package implement Forseti scanner rules (https://forsetisecurity.org/docs/latest/configure/scanner/descriptions.html).
// Examples rules can be found at https://github.com/forseti-security/forseti-security/tree/master/rules.
package rulegen

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/GoogleCloudPlatform/healthcare/deploy/runner"
	"gopkg.in/yaml.v2" // don't use ghodss/yaml as it does not preserve key ordering
)

// Run runs the rule generator to generate forseti rules.
// outputPath should be empty or a path to either a local directory or a GCS bucket (starting with gs://).
// If the outputPath is empty, then the rules will be written to the forseti server bucket.
func Run(conf *config.Config, outputPath string, rn runner.Runner) (err error) {
	if conf.Forseti == nil {
		return errors.New("forseti conf must be set when using the rule generator")
	}
	if outputPath == "" {
		outputPath = conf.AllGeneratedFields.Forseti.ServiceBucket
	}

	local := outputPath

	// if output path is a bucket, write rules to temp dir first and then copy it over to the bucket
	if strings.HasPrefix(outputPath, "gs://") {
		local, err = ioutil.TempDir("", "")
		if err != nil {
			return fmt.Errorf("failed to create temp dir: %v", err)
		}
		defer func() {
			if err := os.RemoveAll(local); err != nil {
				log.Printf("failed to delete temp dir %q: %v", local, err)
			}
		}()

		defer func() {
			// only copy rules if everything prior was successful
			if err != nil {
				return
			}
			err = copyRulesToBucket(local, outputPath, rn)
		}()
	} else {
		local, err = config.NormalizePath(local)
		if err != nil {
			return fmt.Errorf("failed to normalize path %q: %v", local, err)
		}
	}
	if err := writeRules(conf, local); err != nil {
		return fmt.Errorf("failed to write rules: %v")
	}
	return writeAuditConfig(conf, local)
}

func writeAuditConfig(conf *config.Config, outputPath string) error {
	b, err := yaml.Marshal(conf)
	if err != nil {
		return fmt.Errorf("failed to marshal config file: %v", err)
	}
	p := filepath.Join(outputPath, "audit_config.yaml")
	log.Println("Writing audit config", p)
	if err := ioutil.WriteFile(p, b, 0644); err != nil {
		return fmt.Errorf("failed to write audit config to %q: %v", p, err)
	}
	return nil
}

func writeRules(conf *config.Config, outputPath string) error {
	filenameToRules := make(map[string]interface{})

	var errs []string
	add := func(fn string, rules interface{}, err error) {
		if err != nil {
			errs = append(errs, fmt.Sprintf("- %q: %v", fn, err))
		}
		filenameToRules[fn] = rules
	}

	al, err := AuditLoggingRules(conf)
	add("audit_logging", al, err)

	bq, err := BigqueryRules(conf)
	add("bigquery", bq, err)

	bkt, err := BucketRules(conf)
	add("bucket", bkt, err)

	cs, err := CloudSQLRules(conf)
	add("cloudsql", cs, err)

	api, err := EnabledAPIsRules(conf)
	add("enabled_apis", api, err)

	iam, err := IAMRules(conf)
	add("iam", iam, err)

	lien, err := LienRules(conf)
	add("lien", lien, err)

	loc, err := LocationRules(conf)
	add("location", loc, err)

	sink, err := LogSinkRules(conf)
	add("log_sink", sink, err)

	res, err := ResourceRules(conf)
	add("resource", res, err)

	if len(errs) > 0 {
		return fmt.Errorf("failed to generate rules for %d scanners:\n%v", len(errs), strings.Join(errs, "\n"))
	}

	for fn, rules := range filenameToRules {
		b, err := yaml.Marshal(map[string]interface{}{"rules": rules})
		if err != nil {
			return fmt.Errorf("failed to marshal rules for %q: %v", fn, err)
		}
		p := filepath.Join(outputPath, fn+"_rules.yaml")
		log.Println("Writing", p)
		if err := ioutil.WriteFile(p, b, 0644); err != nil {
			return fmt.Errorf("failed to write rules to %q: %v", p, err)
		}
	}
	return nil
}

// copyRulesToBucket copies the rules from the local path to the remote GCS bucket path.
func copyRulesToBucket(local, remote string, rn runner.Runner) error {
	u, err := url.Parse(remote)
	if err != nil {
		return fmt.Errorf("failed to parse %q: %v", remote, err)
	}
	u.Path = path.Join(u.Path, "rules")
	log.Printf("Uploading rules to %q", u.String())
	cmd := exec.Command("gsutil", "cp", filepath.Join(local, "*.yaml"), u.String())
	if out, err := rn.CmdCombinedOutput(cmd); err != nil {
		return fmt.Errorf("failed to copy yaml files to forseti server bucket: %v, %v", err, string(out))
	}
	return nil
}
