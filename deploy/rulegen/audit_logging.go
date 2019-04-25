package rulegen

import "github.com/GoogleCloudPlatform/healthcare/deploy/cft"

// AuditLoggingRule represents a forseti audit logging rule.
type AuditLoggingRule struct {
	Name      string     `yaml:"name"`
	Resources []resource `yaml:"resource"`
	Service   string     `yaml:"service"`
	LogTypes  []string   `yaml:"log_types"`
}

// AuditLoggingRules builds audit logging scanner rules for the given config.
func AuditLoggingRules(config *cft.Config) ([]AuditLoggingRule, error) {
	return []AuditLoggingRule{{
		Name: "Require all Cloud Audit logs.",
		Resources: []resource{{
			Type: "project",
			IDs:  []string{"*"},
		}},
		Service: "allServices",
		LogTypes: []string{
			"ADMIN_READ",
			"DATA_READ",
			"DATA_WRITE",
		},
	}}, nil
}
