package rulegen

import (
	"fmt"

	"github.com/GoogleCloudPlatform/healthcare/deploy/cft"
)

var resourceTypes = []string{
	"project",
	"bucket",
	"dataset",
	"instance",
}

// ResourceRule represents a forseti resource scanner rule.
type ResourceRule struct {
	Name          string         `yaml:"name"`
	Mode          string         `yaml:"mode"`
	ResourceTypes []string       `yaml:"resource_types"`
	ResourceTrees []resourceTree `yaml:"resource_trees"`
}

type resourceTree struct {
	Type       string         `yaml:"type"`
	ResourceID string         `yaml:"resource_id"`
	Children   []resourceTree `yaml:"children"`
}

// ResourceRules builds resource scanner rules for the given config.
func ResourceRules(config *cft.Config) ([]ResourceRule, error) {
	trees := []resourceTree{
		{Type: "project", ResourceID: "*"}, // ignore unmonitored projects
	}

	for _, project := range config.Projects {
		pt := resourceTree{
			Type:       "project",
			ResourceID: project.ID,
			Children: []resourceTree{
				{
					Type:       "bucket",
					ResourceID: project.AuditLogs.LogsGCSBucket.Name,
				},
				{
					Type:       "dataset",
					ResourceID: fmt.Sprintf("%s:%s", config.AuditLogsProjectID(project), project.AuditLogs.LogsBigqueryDataset.Name),
				},
			},
		}

		rs := project.DataResources()

		for _, b := range rs.GCSBuckets {
			pt.Children = append(pt.Children, resourceTree{
				Type:       "bucket",
				ResourceID: b.Name(),
			})
		}

		for _, d := range rs.BigqueryDatasets {
			pt.Children = append(pt.Children, resourceTree{
				Type:       "dataset",
				ResourceID: fmt.Sprintf("%s:%s", project.ID, d.Name()),
			})
		}

		for _, i := range rs.GCEInstances {
			id, err := project.InstanceID(i.Name())
			if err != nil {
				return nil, err
			}
			pt.Children = append(pt.Children, resourceTree{
				Type:       "instance",
				ResourceID: id,
			})
		}

		trees = append(trees, pt)
	}

	return []ResourceRule{{
		Name:          "Project resource trees.",
		Mode:          "required",
		ResourceTypes: resourceTypes,
		ResourceTrees: trees,
	}}, nil
}
