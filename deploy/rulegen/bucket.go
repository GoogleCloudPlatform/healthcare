package rulegen

import "github.com/GoogleCloudPlatform/healthcare/deploy/cft"

// BucketRule represents a forseti GCS bucket ACL rule.
type BucketRule struct {
	Name      string     `yaml:"name"`
	Bucket    string     `yaml:"bucket"`
	Entity    string     `yaml:"entity"`
	Email     string     `yaml:"email"`
	Domain    string     `yaml:"domain"`
	Role      string     `yaml:"role"`
	Resources []resource `yaml:"resource"`
}

// BucketRules builds bucket scanner rules for the given config.
func BucketRules(config *cft.Config) ([]BucketRule, error) {
	return []BucketRule{{
		Name:      "Disallow all acl rules, only allow IAM.",
		Bucket:    "*",
		Entity:    "*",
		Email:     "*",
		Domain:    "*",
		Role:      "*",
		Resources: []resource{{IDs: []string{"*"}}},
	}}, nil
}
