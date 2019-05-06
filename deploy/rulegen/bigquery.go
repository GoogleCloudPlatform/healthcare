package rulegen

import (
	"fmt"
	"log"
	"strings"

	"github.com/GoogleCloudPlatform/healthcare/deploy/cft"
	"github.com/mitchellh/hashstructure"
)

// BigqueryRule represents a forseti bigquery rule.
type BigqueryRule struct {
	Name       string            `yaml:"name"`
	Mode       string            `yaml:"mode"`
	Resources  []resource        `yaml:"resource"`
	DatasetIDs []string          `yaml:"dataset_ids"`
	Bindings   []bigqueryBinding `yaml:"bindings"`
}

type bigqueryBinding struct {
	Role    string           `yaml:"role"`
	Members []bigqueryMember `yaml:"members"`
}

type bigqueryMember struct {
	Domain       string `yaml:"domain,omitempty" `
	UserEmail    string `yaml:"user_email,omitempty"`
	GroupEmail   string `yaml:"group_email,omitempty"`
	SpecialGroup string `yaml:"special_group,omitempty"`
}

// BigqueryRules builds bigquery scanner rules for the given config.
func BigqueryRules(config *cft.Config) ([]BigqueryRule, error) {
	global := BigqueryRule{
		Name:       "No public, domain or special group dataset access.",
		Mode:       "blacklist",
		Resources:  []resource{globalResource(config)},
		DatasetIDs: []string{"*"},
		Bindings: []bigqueryBinding{{
			Role: "*",
			Members: []bigqueryMember{
				{Domain: "*"},
				{SpecialGroup: "*"},
			},
		}},
	}

	rules := []BigqueryRule{global}

	for _, project := range config.AllProjects() {
		prules, err := getProjectDatasetsRules(project)
		if err != nil {
			return nil, fmt.Errorf("failed to get dataset rules for project %q: %v", project.ID, err)
		}
		rules = append(rules, prules...)
		rules = append(rules, getAuditLogDatasetRule(config, project))
	}

	return rules, nil
}

func getProjectDatasetsRules(project *cft.Project) ([]BigqueryRule, error) {
	var rules []BigqueryRule

	// group rules that have the same access together to reduce duplicated rules
	accessHashToDatasets := make(map[uint64][]*cft.BigqueryDataset)
	var hashes []uint64 // for stable ordering

	for _, dataset := range project.DataResources().BigqueryDatasets {
		h, err := hashstructure.Hash(dataset.Accesses, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to hash access %v: %v", dataset.Accesses, err)
		}
		if _, ok := accessHashToDatasets[h]; !ok {
			hashes = append(hashes, h)
		}
		accessHashToDatasets[h] = append(accessHashToDatasets[h], dataset)
	}

	for _, h := range hashes {
		ds := accessHashToDatasets[h]
		var ids []string
		for _, d := range ds {
			// forseti bigquery dataset id is in the form projectID:datasetName
			ids = append(ids, fmt.Sprintf("%s:%s", project.ID, d.Name()))
		}

		joined := strings.Join(ids, ", ")
		if len(joined) < 127 {
			joined += "."
		} else {
			joined = joined[:122] + ", ..."
		}

		rules = append(rules, BigqueryRule{
			Name:       "Whitelist for dataset(s): " + joined,
			Mode:       "whitelist",
			DatasetIDs: ids,
			Resources: []resource{{
				Type: "project",
				IDs:  []string{project.ID},
			}},
			Bindings: accessesToBindings(ds[0].Accesses),
		})
	}

	return rules, nil
}

func accessesToBindings(accesses []cft.Access) []bigqueryBinding {
	roleToMembers := make(map[string][]bigqueryMember)
	var roles []string // for stable ordering
	for _, access := range accesses {
		// only one should be non-empty (checked by bigquery CFT schema)
		member := bigqueryMember{
			UserEmail:    access.UserByEmail,
			GroupEmail:   access.GroupByEmail,
			SpecialGroup: access.SpecialGroup,
		}
		if member == (bigqueryMember{}) {
			log.Printf("unmonitored access: %v", access)
			continue
		}
		if _, ok := roleToMembers[access.Role]; !ok {
			roles = append(roles, access.Role)
		}
		roleToMembers[access.Role] = append(roleToMembers[access.Role], member)
	}
	var bs []bigqueryBinding
	for _, role := range roles {
		bs = append(bs, bigqueryBinding{role, roleToMembers[role]})
	}
	return bs
}

func getAuditLogDatasetRule(config *cft.Config, project *cft.Project) BigqueryRule {
	bindings := []bigqueryBinding{
		{Role: "OWNER", Members: []bigqueryMember{{GroupEmail: project.OwnersGroup}}},
		{Role: "WRITER", Members: []bigqueryMember{{UserEmail: project.GeneratedFields.LogSinkServiceAccount}}},
		{Role: "READER", Members: []bigqueryMember{{GroupEmail: project.AuditorsGroup}}},
	}

	auditProjectID := config.AuditLogsProjectID(project)
	return BigqueryRule{
		Name:       fmt.Sprintf("Whitelist for project %s audit logs.", project.ID),
		Mode:       "whitelist",
		DatasetIDs: []string{fmt.Sprintf("%s:%s", auditProjectID, project.AuditLogs.LogsBigqueryDataset.Name)},
		Resources: []resource{{
			Type: "project",
			IDs:  []string{auditProjectID},
		}},
		Bindings: bindings,
	}
}
