package rulegen

import (
	"fmt"

	"github.com/GoogleCloudPlatform/healthcare/deploy/cft"
)

// EnabledAPIsRule represents a forseti enabled APIs rule.
type EnabledAPIsRule struct {
	Name      string     `yaml:"name"`
	Mode      string     `yaml:"mode"`
	Resources []resource `yaml:"resource"`
	Services  []string   `yaml:"services"`
}

// EnabledAPIsRules builds enabled APIs scanner rules for the given config.
func EnabledAPIsRules(config *cft.Config) ([]EnabledAPIsRule, error) {
	rules := []EnabledAPIsRule{{
		Name:      "Global API whitelist.",
		Mode:      "whitelist",
		Resources: []resource{{Type: "project", IDs: []string{"*"}}},
		Services:  config.Overall.AllowedAPIs,
	}}

	for _, project := range config.Projects {
		if len(project.EnabledAPIs) == 0 {
			continue
		}
		rules = append(rules, EnabledAPIsRule{
			Name:      fmt.Sprintf("API whitelist for %s.", project.ID),
			Mode:      "whitelist",
			Resources: []resource{{Type: "project", IDs: []string{project.ID}}},
			Services:  project.EnabledAPIs,
		})
	}

	return rules, nil
}
