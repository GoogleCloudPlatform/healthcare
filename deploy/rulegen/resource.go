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
	Children   []resourceTree `yaml:"children,omitempty"`
}

// ResourceRules builds resource scanner rules for the given config.
func ResourceRules(config *cft.Config) ([]ResourceRule, error) {
	trees := []resourceTree{
		{Type: "project", ResourceID: "*"}, // ignore unmonitored projects
	}

	for _, project := range config.AllProjects() {
		pt := resourceTree{
			Type:       "project",
			ResourceID: project.ID,
			Children:   getAuditTrees(config, project),
		}

		rs := project.ResourcesByType()

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

func getAuditTrees(config *cft.Config, project *cft.Project) []resourceTree {
	if config.ProjectForAuditLogs(project).ID != project.ID {
		return nil
	}

	if config.AuditLogsProject == nil {
		return getAuditTreesForProjects(project.ID, project)
	}
	// audit project holds audit resources for all projects
	return getAuditTreesForProjects(project.ID, config.AllProjects()...)
}

func getAuditTreesForProjects(auditLogsProjectID string, projects ...*cft.Project) []resourceTree {
	var trees []resourceTree
	for _, project := range projects {
		trees = append(trees, resourceTree{
			Type:       "dataset",
			ResourceID: fmt.Sprintf("%s:%s", auditLogsProjectID, project.AuditLogs.LogsBigqueryDataset.Name),
		})
		if project.AuditLogs.LogsGCSBucket != nil {
			trees = append(trees, resourceTree{
				Type:       "bucket",
				ResourceID: project.AuditLogs.LogsGCSBucket.Name,
			})
		}
	}
	return trees
}
