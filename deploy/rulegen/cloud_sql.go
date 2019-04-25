package rulegen

import (
	"fmt"
	"strconv"

	"github.com/GoogleCloudPlatform/healthcare/deploy/cft"
)

// CloudSQLRule represents a forseti cloud SQL rule.
type CloudSQLRule struct {
	Name               string     `yaml:"name"`
	Resources          []resource `yaml:"resource"`
	InstanceName       string     `yaml:"instance_name"`
	AuthorizedNetworks string     `yaml:"authorized_networks"`
	SSLEnabled         string     `yaml:"ssl_enabled"`
}

// CloudSQLRules builds cloud SQL scanner rules for the given config.
func CloudSQLRules(config *cft.Config) ([]CloudSQLRule, error) {
	var rules []CloudSQLRule
	for _, b := range []bool{false, true} {
		ssl := "enabled"
		if !b {
			ssl = "disabled"
		}

		rules = append(rules, CloudSQLRule{
			Name:               fmt.Sprintf("Disallow publicly exposed cloudsql instances (SSL %s).", ssl),
			Resources:          []resource{globalResource(config)},
			InstanceName:       "*",
			AuthorizedNetworks: "0.0.0.0/0",
			SSLEnabled:         strconv.FormatBool(b),
		})
	}
	return rules, nil
}
