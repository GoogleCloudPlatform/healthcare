package rulegen

import (
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/mitchellh/hashstructure"
)

// bucketLogsWriter is the access logs writer.
// https://cloud.google.com/storage/docs/access-logs#delivery.
const bucketLogsWriter = "group:cloud-storage-analytics@google.com"

// wantBucketRoles are the roles that each bucket rule must have.
// Forseti will not the role in whitelist mode if it's not found in the rule's bindings list.
// Thus, every role here must be present. If there are no members for the role, add a non-existent one (e.g. user:nobody).
var wantBucketRoles = []string{
	"roles/storage.admin",
	"roles/storage.objectAdmin",
	"roles/storage.objectCreator",
	"roles/storage.objectViewer",
}

func getTemplates(ss []string) []*template.Template {
	var ts []*template.Template
	for _, s := range ss {
		ts = append(ts, template.Must(template.New("").Parse(s)))
	}
	return ts
}

var (
	// defaultServiceAccountTemplates are the default Service Accounts with Editors role.
	defaultServiceAccountTemplates = getTemplates([]string{
		// Compute Engine default service account
		"{{.ProjectNum}}-compute@developer.gserviceaccount.com",
		// Google APIs Service Agent (e.g. Deployment manager)
		"{{.ProjectNum}}@cloudservices.gserviceaccount.com",
		// Google Container Registry Service Agent
		"service-{{.ProjectNum}}@containerregistry.iam.gserviceaccount.com",
	})

	// typeToAllowedMemberTemplates is the default global whitelists for specific resource types.
	typeToAllowedMemberTemplates = []typeAndTemplates{
		{
			"project",
			getTemplates([]string{"group:*@{{.Domain}}", "serviceAccount:*.gserviceaccount.com"}),
		},
		{
			"bucket",
			getTemplates([]string{"group:*@{{.Domain}}", "user:*@{{.Domain}}", "serviceAccount:*.gserviceaccount.com"}),
		},
	}
)

type typeAndTemplates struct {
	typ   string
	tmpls []*template.Template
}

// IAMRule represents a forseti iam rule.
type IAMRule struct {
	Name               string           `yaml:"name"`
	Mode               string           `yaml:"mode"`
	Resources          []resource       `yaml:"resource"`
	InheritFromParents bool             `yaml:"inherit_from_parents"`
	Bindings           []config.Binding `yaml:"bindings"`
}

// IAMRules builds IAM scanner rules for the given config.
func IAMRules(conf *config.Config) ([]IAMRule, error) {
	rules := []IAMRule{
		{
			Name: "All projects must have an owner group from the domain.",
			Mode: "required",
			Resources: []resource{{
				Type:      "project",
				AppliesTo: "self",
				IDs:       []string{"*"},
			}},
			InheritFromParents: true,
			Bindings: []config.Binding{{
				Role:    "roles/owner",
				Members: []string{"group:*@" + conf.Overall.Domain},
			}},
		},
		{
			Name: "All billing account roles must be groups from the domain.",
			Mode: "whitelist",
			Resources: []resource{{
				Type:      "billing_account",
				AppliesTo: "self",
				IDs:       []string{"*"},
			}},
			InheritFromParents: false,
			Bindings: []config.Binding{{
				Role:    "*",
				Members: []string{"group:*@" + conf.Overall.Domain},
			}},
		},
	}

	var projectRules []IAMRule
	for _, project := range conf.AllProjects() {
		prules, err := getProjectRules(conf, project)
		if err != nil {
			return nil, err
		}
		projectRules = append(projectRules, prules...)
	}

	for _, tt := range typeToAllowedMemberTemplates {
		grule, err := getGlobalRuleForType(conf, tt, projectRules)
		if err != nil {
			return nil, err
		}
		rules = append(rules, *grule)
	}

	rules = append(rules, projectRules...)
	return rules, nil
}

// getProjectRules gets the rules for the given project as well as any resources that set IAM policies.
// TODO: should we be checking pubsub as well?
func getProjectRules(conf *config.Config, project *config.Project) ([]IAMRule, error) {
	var rules []IAMRule
	if project.AuditLogs.LogsGCSBucket != nil {
		rules = append(rules, getLogsBucketRule(project))
	}

	projectBindings, err := getProjectBindings(conf, project)
	if err != nil {
		return nil, err
	}

	rules = append(rules, IAMRule{
		Name: fmt.Sprintf("Role whitelist for project %s.", project.ID),
		Mode: "whitelist",
		Resources: []resource{{
			Type:      "project",
			AppliesTo: "self",
			IDs:       []string{project.ID},
		}},
		InheritFromParents: true,
		Bindings:           projectBindings,
	})

	bucketRules, err := getDataBucketRules(project)
	if err != nil {
		return nil, err
	}
	rules = append(rules, bucketRules...)
	return rules, nil
}

// getProjectBindings gets the project level bindings for the given project.
func getProjectBindings(conf *config.Config, project *config.Project) ([]config.Binding, error) {
	var bs []config.Binding
	for _, policy := range project.Resources.IAMPolicies {
		bs = append(bs, policy.Parsed.Bindings...)
	}

	// We can assume Forseti config exists if the rule generator is being called
	// TODO: check for other forseti service account roles granted on the project
	bs = append(bs, config.Binding{
		Role:    "roles/iam.securityReviewer",
		Members: []string{"serviceAccount:" + conf.AllGeneratedFields.Forseti.ServiceAccount},
	})

	var ms []string
	for _, saTmpl := range defaultServiceAccountTemplates {
		var b strings.Builder
		if err := saTmpl.Execute(&b, map[string]interface{}{"ProjectNum": project.GeneratedFields.ProjectNumber}); err != nil {
			return nil, err
		}
		ms = append(ms, "serviceAccount:"+b.String())
	}
	bs = append(bs, config.Binding{Role: "roles/editor", Members: ms})

	bs = config.MergeBindings(bs...)
	return bs, nil
}

// getLogsBucketRule gets the iam rule for the logs bucket.
// For configs with an audit log project, only the audit log project will return a non-nil rule containing all other projects' logs buckets.
// For configs without an audit log project, each project will have a single rule for its own local logs bucket.
func getLogsBucketRule(project *config.Project) IAMRule {
	bucket := project.AuditLogs.LogsGCSBucket

	return IAMRule{
		Name: fmt.Sprintf("Role whitelist for project %s log bucket(s).", project.ID),
		Mode: "whitelist",
		Resources: []resource{{
			Type:      "bucket",
			AppliesTo: "self",
			IDs:       []string{bucket.Name()},
		}},
		InheritFromParents: true,
		Bindings:           fillMissingBucketBindings(bucket.Bindings),
	}
}

// getDataBucketRules gets the IAM rules for data holding buckets.
func getDataBucketRules(project *config.Project) ([]IAMRule, error) {
	var rules []IAMRule

	// group rules that have the same bindings together to reduce duplicated rules
	bindingsHashToBuckets := make(map[uint64][]config.GCSBucket)
	// TODO: this pattern is repeated several times and could benefit from a helper struct once generics are supported.
	var hashes []uint64 // for stable ordering
	for _, pair := range project.Resources.GCSBuckets {
		bucket := pair.Parsed
		h, err := hashstructure.Hash(bucket.Bindings, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to hash access %v: %v", bucket.Bindings, err)
		}
		if _, ok := bindingsHashToBuckets[h]; !ok {
			hashes = append(hashes, h)
		}
		bindingsHashToBuckets[h] = append(bindingsHashToBuckets[h], bucket)
	}

	for _, h := range hashes {
		buckets := bindingsHashToBuckets[h]
		var ids []string
		for _, b := range buckets {
			ids = append(ids, b.Name())
		}

		joined := strings.Join(ids, ", ")
		if len(joined) < 127 {
			joined += "."
		} else {
			joined = joined[:127] + "..."
		}

		bindings := fillMissingBucketBindings(buckets[0].Bindings)

		rules = append(rules, IAMRule{
			Name: fmt.Sprintf("Role whitelist for project %s bucket(s): %s", project.ID, joined),
			Mode: "whitelist",
			Resources: []resource{{
				Type:      "bucket",
				AppliesTo: "self",
				IDs:       ids,
			}},
			InheritFromParents: true,
			Bindings:           bindings,
		})
	}
	return rules, nil
}

func fillMissingBucketBindings(bindings []config.Binding) []config.Binding {
	gotRoles := make(map[string]bool)
	for _, b := range bindings {
		gotRoles[b.Role] = true
	}
	for _, r := range wantBucketRoles {
		if !gotRoles[r] {
			bindings = append(bindings, config.Binding{Role: r, Members: []string{"user:nobody"}})
		}
	}
	return bindings
}

// getGlobalRuleForType returns the global rule for the given type. It should be called with the project level rules of all projects.
func getGlobalRuleForType(conf *config.Config, tt typeAndTemplates, projectRules []IAMRule) (*IAMRule, error) {
	membersSet := make(map[string]bool)
	var members []string
	for _, memberTmpl := range tt.tmpls {
		var b strings.Builder
		if err := memberTmpl.Execute(&b, map[string]interface{}{"Domain": conf.Overall.Domain}); err != nil {
			return nil, err
		}

		formatted := b.String()
		if !membersSet[formatted] {
			membersSet[formatted] = true
			members = append(members, formatted)
		}
	}

	var exprs []string
	for member := range membersSet {
		exprs = append(exprs, strings.ReplaceAll(regexp.QuoteMeta(member), `\*`, `.*`))
	}
	re, err := regexp.Compile(strings.Join(exprs, "|"))
	if err != nil {
		return nil, err
	}

	// add all members that don't conform to the wildcard whitelists
	for _, rule := range projectRules {
		for _, res := range rule.Resources {
			if res.Type != tt.typ {
				continue
			}
			for _, b := range rule.Bindings {
				for _, m := range b.Members {
					if !membersSet[m] && !re.MatchString(m) {
						membersSet[m] = true
						members = append(members, m)
					}
				}
			}
		}
	}

	return &IAMRule{
		Name:               fmt.Sprintf("Global whitelist of allowed members for %s roles.", tt.typ),
		Mode:               "whitelist",
		Resources:          []resource{{Type: tt.typ, AppliesTo: "self", IDs: []string{"*"}}},
		InheritFromParents: true,
		Bindings:           []config.Binding{{Role: "*", Members: members}},
	}, nil
}
