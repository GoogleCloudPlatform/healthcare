package rulegen

import (
	"fmt"

	"github.com/GoogleCloudPlatform/healthcare/deploy/cft"
)

const allBigquerySinksDestination = "bigquery.googleapis.com/*"

// LogSinkRule represents a forseti log sink rule.
type LogSinkRule struct {
	Name      string     `yaml:"name"`
	Mode      string     `yaml:"mode"`
	Resources []resource `yaml:"resource"`
	Sink      sink       `yaml:"sink"`
}

type sink struct {
	Destination     string `yaml:"destination"`
	Filter          string `yaml:"filter"`
	IncludeChildren string `yaml:"include_children"`
}

// LogSinkRules builds log sink scanner rules for the given config.
func LogSinkRules(config *cft.Config) ([]LogSinkRule, error) {
	gr := globalResource(config)
	if gr.Type == "project" {
		gr.AppliesTo = "self"
	} else {
		gr.AppliesTo = "children"
	}
	rules := []LogSinkRule{
		{
			Name:      "Require a BigQuery Log sink in all projects.",
			Mode:      "required",
			Resources: []resource{gr},
			Sink:      getSink(allBigquerySinksDestination),
		},
		{
			Name:      "Only allow BigQuery Log sinks in all projects.",
			Mode:      "whitelist",
			Resources: []resource{gr},
			Sink:      getSink(allBigquerySinksDestination),
		},
	}

	for _, project := range config.AllProjects() {
		res := []resource{{Type: "project", AppliesTo: "self", IDs: []string{project.ID}}}
		s := getSink(auditLogSinkDestination(config, project))
		rules = append(rules,
			LogSinkRule{
				Name:      fmt.Sprintf("Require Log sink for project %s.", project.ID),
				Mode:      "required",
				Resources: res,
				Sink:      s,
			},
			LogSinkRule{
				Name:      fmt.Sprintf("Whitelist Log sink for project %s.", project.ID),
				Mode:      "whitelist",
				Resources: res,
				Sink:      s,
			},
		)
	}

	return rules, nil
}

func getSink(destination string) sink {
	return sink{
		Destination: destination,
		// TODO: Change the filter to be specifically an audit logs sink once deployed log sinks use that filter.
		Filter:          "*",
		IncludeChildren: "*",
	}
}

func auditLogSinkDestination(config *cft.Config, project *cft.Project) string {
	return fmt.Sprintf("bigquery.googleapis.com/projects/%s/datasets/%s", config.ProjectForAuditLogs(project).ID, project.AuditLogs.LogsBigqueryDataset.Name)
}
