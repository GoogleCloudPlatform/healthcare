package rulegen

import (
	"fmt"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
)

// EnabledAPIsRule represents a forseti enabled APIs rule.
type EnabledAPIsRule struct {
	Name      string     `yaml:"name"`
	Mode      string     `yaml:"mode"`
	Resources []resource `yaml:"resource"`
	Services  []string   `yaml:"services"`
}

// EnabledAPIsRules builds enabled APIs scanner rules for the given config.
func EnabledAPIsRules(conf *config.Config) ([]EnabledAPIsRule, error) {
	rules := []EnabledAPIsRule{{
		Name:      "Global API whitelist.",
		Mode:      "whitelist",
		Resources: []resource{{Type: "project", IDs: []string{"*"}}},
		Services:  conf.Overall.AllowedAPIs,
	}}

	for _, project := range conf.AllProjects() {
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
