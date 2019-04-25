package rulegen

import (
	"github.com/GoogleCloudPlatform/healthcare/deploy/cft"
)

type resource struct {
	Type string   `yaml:"type"`
	IDs  []string `yaml:"resource_ids"`
}

// globalResource tries to find the broadest scope of resources defined in the config.
// This is required due to organization and folder IDs being optional.
// The order of preference is organization, folder or all projects defined in the config.
func globalResource(config *cft.Config) resource {
	switch {
	case config.Overall.OrganizationID != "":
		return resource{Type: "organization", IDs: []string{config.Overall.OrganizationID}}
	case config.Overall.FolderID != "":
		return resource{Type: "folder", IDs: []string{config.Overall.FolderID}}
	default:
		ids := make([]string, 0, len(config.Projects))
		for _, p := range config.Projects {
			ids = append(ids, p.ID)
		}
		return resource{Type: "project", IDs: ids}
	}
}
