package rulegen

import (
	"fmt"
	"log"
	"strings"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/mitchellh/hashstructure"
)

// BigqueryRule represents a forseti bigquery rule.
type BigqueryRule struct {
	Name       string            `yaml:"name"`
	Mode       string            `yaml:"mode"`
	DatasetIDs []string          `yaml:"dataset_ids"`
	Resources  []resource        `yaml:"resource"`
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
func BigqueryRules(conf *config.Config) ([]BigqueryRule, error) {
	// TODO: check for public access in config and allow it if it is intentionally given
	global := BigqueryRule{
		Name:       "No public, domain or special group dataset access.",
		Mode:       "blacklist",
		Resources:  []resource{globalResource(conf)},
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

	for _, project := range conf.AllProjects() {
		prules, err := getProjectDatasetsRules(project)
		if err != nil {
			return nil, fmt.Errorf("failed to get dataset rules for project %q: %v", project.ID, err)
		}
		rules = append(rules, prules...)
		rules = append(rules, getAuditLogDatasetRule(conf, project))
	}

	return rules, nil
}

func getProjectDatasetsRules(project *config.Project) ([]BigqueryRule, error) {
	var rules []BigqueryRule

	// group rules that have the same access together to reduce duplicated rules
	accessHashToDatasets := make(map[uint64][]config.BigqueryDataset)
	var hashes []uint64 // for stable ordering

	for _, pair := range project.Resources.BQDatasets {
		dataset := pair.Parsed
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
			joined += "..."
		}

		bindings, err := accessesToBindings(ds[0].Accesses)
		if err != nil {
			return nil, err
		}
		rules = append(rules, BigqueryRule{
			Name:       "Whitelist for dataset(s): " + joined,
			Mode:       "whitelist",
			DatasetIDs: ids,
			Resources: []resource{{
				Type: "project",
				IDs:  []string{project.ID},
			}},
			Bindings: bindings,
		})
	}

	return rules, nil
}

// accessToBindings converts a list of access to bindings.
// Note that due to the way forseti scanner works, all valid roles must be present.
// Missing roles will allow any member for the role.
func accessesToBindings(accesses []config.Access) ([]bigqueryBinding, error) {
	roles := []string{"OWNER", "WRITER", "READER"}
	roleToMembers := map[string][]bigqueryMember{
		"OWNER":  nil,
		"WRITER": nil,
		"READER": nil,
	}
	for _, access := range accesses {
		// only one should be non-empty (checked by bigquery config schema)
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
			return nil, fmt.Errorf("unexpected role %q, want one of %v", access.Role, roles)
		}
		roleToMembers[access.Role] = append(roleToMembers[access.Role], member)
	}
	var bs []bigqueryBinding
	for _, role := range roles {
		bs = append(bs, bigqueryBinding{role, roleToMembers[role]})
	}
	return bs, nil
}

func getAuditLogDatasetRule(conf *config.Config, project *config.Project) BigqueryRule {
	auditProject := conf.ProjectForAuditLogs(project)

	bindings := []bigqueryBinding{
		{Role: "OWNER", Members: []bigqueryMember{{GroupEmail: auditProject.OwnersGroup}}},
		{Role: "WRITER", Members: []bigqueryMember{{UserEmail: project.GeneratedFields.LogSinkServiceAccount}}},
		{Role: "READER", Members: []bigqueryMember{{GroupEmail: project.AuditorsGroup}}},
	}

	return BigqueryRule{
		Name:       fmt.Sprintf("Whitelist for project %s audit logs.", project.ID),
		Mode:       "whitelist",
		DatasetIDs: []string{fmt.Sprintf("%s:%s", auditProject.ID, project.AuditLogs.LogsBQDataset.Name())},
		Resources: []resource{{
			Type: "project",
			IDs:  []string{auditProject.ID},
		}},
		Bindings: bindings,
	}
}
