package rulegen

import (
	"fmt"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
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
func ResourceRules(conf *config.Config) ([]ResourceRule, error) {
	trees := []resourceTree{
		{Type: "project", ResourceID: "*"}, // ignore unmonitored projects
	}

	for _, project := range conf.AllProjects() {
		pt := resourceTree{
			Type:       "project",
			ResourceID: project.ID,
			Children:   getAuditTrees(conf, project),
		}

		for _, b := range project.Resources.GCSBuckets {
			pt.Children = append(pt.Children, resourceTree{
				Type:       "bucket",
				ResourceID: b.Name(),
			})
		}

		for _, d := range project.Resources.BQDatasets {
			pt.Children = append(pt.Children, resourceTree{
				Type:       "dataset",
				ResourceID: fmt.Sprintf("%s:%s", project.ID, d.Name()),
			})
		}

		for _, i := range project.Resources.GCEInstances {
			id, err := project.GeneratedFields.InstanceID(i.Name())
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

func getAuditTrees(conf *config.Config, project *config.Project) []resourceTree {
	if conf.ProjectForAuditLogs(project).ID != project.ID {
		return nil
	}

	if conf.AuditLogsProject == nil {
		return getAuditTreesForProjects(project.ID, project)
	}
	// audit project holds audit resources for all projects
	return getAuditTreesForProjects(project.ID, conf.AllProjects()...)
}

func getAuditTreesForProjects(auditLogsProjectID string, projects ...*config.Project) []resourceTree {
	var trees []resourceTree
	for _, project := range projects {
		trees = append(trees, resourceTree{
			Type:       "dataset",
			ResourceID: fmt.Sprintf("%s:%s", auditLogsProjectID, project.AuditLogs.LogsBQDataset.Name()),
		})
		if project.AuditLogs.LogsGCSBucket != nil {
			trees = append(trees, resourceTree{
				Type:       "bucket",
				ResourceID: project.AuditLogs.LogsGCSBucket.Name(),
			})
		}
	}
	return trees
}
