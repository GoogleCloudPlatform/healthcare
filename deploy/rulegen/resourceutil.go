package rulegen

import (
	"github.com/GoogleCloudPlatform/healthcare/deploy/cft"
)

type resource struct {
	Type      string   `yaml:"type,omitempty"`
	AppliesTo string   `yaml:"applies_to,omitempty"`
	IDs       []string `yaml:"resource_ids"`
}

// globalResource tries to find the broadest scope of resources defined in the config.
// This is required due to organization and folder IDs being optional.
// The order of preference is organization, folders or all projects defined in the config.
func globalResource(config *cft.Config) resource {
	switch {
	case config.Overall.OrganizationID != "":
		return resource{Type: "organization", IDs: []string{config.Overall.OrganizationID}}
	case config.Overall.FolderID != "":
		ids := []string{config.Overall.FolderID}
		for _, p := range config.AllProjects() {
			if p.FolderID != "" {
				ids = append(ids, p.FolderID)
			}
		}
		return resource{Type: "folder", IDs: ids}
	default:
		var ids []string
		for _, p := range config.AllProjects() {
			ids = append(ids, p.ID)
		}
		return resource{Type: "project", IDs: ids}
	}
}
