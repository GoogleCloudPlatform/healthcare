package rulegen

import "github.com/GoogleCloudPlatform/healthcare/deploy/config"

// LienRule represents a forseti lien rule.
type LienRule struct {
	Name         string     `yaml:"name"`
	Mode         string     `yaml:"mode"`
	Resources    []resource `yaml:"resource"`
	Restrictions []string   `yaml:"restrictions"`
}

// LienRules builds lien scanner rules for the given config.
func LienRules(conf *config.Config) ([]LienRule, error) {
	return []LienRule{{
		Name:         "Require project deletion liens for all projects.",
		Mode:         "required",
		Resources:    []resource{globalResource(conf)},
		Restrictions: []string{"resourcemanager.projects.delete"},
	}}, nil
}
