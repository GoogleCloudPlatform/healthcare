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

	"github.com/GoogleCloudPlatform/healthcare/deploy/cft"
	"gopkg.in/yaml.v2" // don't use ghodss/yaml as it does not preserve key ordering
)

// The following vars are stubbed in tests.
var (
	cmdCombinedOutput = (*exec.Cmd).CombinedOutput
)

// Run runs the rule generator to generate forseti rules.
// outputPath should be empty or a path to either a local directory or a GCS bucket (starting with gs://).
// If the outputPath is empty, then the rules will be written to the forseti server bucket.
func Run(config *cft.Config, outputPath string) (err error) {
	if config.Forseti == nil {
		return errors.New("forseti config must be set when using the rule generator")
	}
	if outputPath == "" {
		outputPath = config.AllGeneratedFields.Forseti.ServiceBucket
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
			err = copyRulesToBucket(local, outputPath)
		}()
	}
	return writeRules(config, local)
}

func writeRules(config *cft.Config, outputPath string) error {
	filenameToRules := make(map[string]interface{})

	var errs []string
	add := func(fn string, rules interface{}, err error) {
		if err != nil {
			errs = append(errs, fmt.Sprintf("- %q: %v", fn, err))
		}
		filenameToRules[fn] = rules
	}

	al, err := AuditLoggingRules(config)
	add("audit_logging", al, err)

	bq, err := BigqueryRules(config)
	add("bigquery", bq, err)

	bkt, err := BucketRules(config)
	add("bucket", bkt, err)

	cs, err := CloudSQLRules(config)
	add("cloudsql", cs, err)

	api, err := EnabledAPIsRules(config)
	add("enabled_apis", api, err)

	iam, err := IAMRules(config)
	add("iam", iam, err)

	lien, err := LienRules(config)
	add("lien", lien, err)

	loc, err := LocationRules(config)
	add("location", loc, err)

	sink, err := LogSinkRules(config)
	add("log_sink", sink, err)

	res, err := ResourceRules(config)
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
func copyRulesToBucket(local, remote string) error {
	u, err := url.Parse(remote)
	if err != nil {
		return fmt.Errorf("failed to parse %q: %v", remote, err)
	}
	u.Path = path.Join(u.Path, "rules")
	cmd := exec.Command("gsutil", "cp", filepath.Join(local, "*.yaml"), u.String())
	if out, err := cmdCombinedOutput(cmd); err != nil {
		return fmt.Errorf("failed to copy yaml files to forseti server bucket: %v, %v", err, string(out))
	}
	return nil
}
