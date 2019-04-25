// Package rulegen provides Forseti rule generation utilities.
// Note: rules in this package implement Forseti scanner rules (https://forsetisecurity.org/docs/latest/configure/scanner/descriptions.html).
// Examples rules can be found at https://github.com/forseti-security/forseti-security/tree/master/rules.
package rulegen

import (
	"fmt"

	"github.com/GoogleCloudPlatform/healthcare/deploy/cft"
	"gopkg.in/yaml.v2" // don't use ghodss/yaml as it does not preserve key ordering
)

type generator struct {
	name  string
	rules interface{}
}

// Run runs the rule generator. Currently it does not write the rules anywhere except stdout.
func Run(config *cft.Config) error {
	var gens []generator
	bqRules, err := BigqueryRules(config)
	if err != nil {
		return fmt.Errorf("failed to generate bigquery rules: %v", err)
	}
	gens = append(gens, generator{"bigquery", bqRules})

	for _, gen := range gens {
		b, err := yaml.Marshal(map[string]interface{}{"rules": gen.rules})
		if err != nil {
			return fmt.Errorf("failed to marshal rules: %v", err)
		}
		fmt.Println(string(b))
	}
	return nil
}
