package rulegen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
)

// LocationRule represents a forseti location rule.
type LocationRule struct {
	Name      string      `yaml:"name"`
	Mode      string      `yaml:"mode"`
	Resources []resource  `yaml:"resource"`
	AppliesTo []appliesTo `yaml:"applies_to"`
	Locations []string    `yaml:"locations"`
}

type appliesTo struct {
	Type        string   `yaml:"type"`
	ResourceIDs []string `yaml:"resource_ids"`
}

// LocationRules builds location scanner rules for the given config.
func LocationRules(conf *config.Config) ([]LocationRule, error) {
	allLocs := make(map[string]bool)
	var projectRules []LocationRule

	for _, project := range conf.AllProjects() {
		m := make(locationToResources)
		if err := m.addResources(project); err != nil {
			return nil, err
		}

		for _, loc := range m.locations() {
			allLocs[loc] = true
			typToIDs := m[loc]
			applies := make([]appliesTo, 0, len(typToIDs))

			for _, typ := range typToIDs.types() {
				applies = append(applies, appliesTo{Type: typ, ResourceIDs: typToIDs[typ]})
			}

			projectRules = append(projectRules, LocationRule{
				Name:      fmt.Sprintf("Project %s resource whitelist for location %s.", project.ID, loc),
				Mode:      "whitelist",
				Resources: []resource{{Type: "project", IDs: []string{project.ID}}},
				AppliesTo: applies,
				Locations: []string{loc},
			})
		}

		auditProject := conf.ProjectForAuditLogs(project)
		res := []resource{{Type: "project", IDs: []string{auditProject.ID}}}

		if project.AuditLogs.LogsGCSBucket != nil {
			projectRules = append(projectRules, LocationRule{
				Name:      fmt.Sprintf("Project %s audit logs bucket location whitelist.", project.ID),
				Mode:      "whitelist",
				Resources: res,
				AppliesTo: []appliesTo{{Type: "bucket", ResourceIDs: []string{project.AuditLogs.LogsGCSBucket.Name()}}},
				Locations: []string{project.AuditLogs.LogsGCSBucket.Location},
			})
		}

		logDatasetID := fmt.Sprintf("%s:%s", auditProject.ID, project.AuditLogs.LogsBQDataset.Name())
		projectRules = append(projectRules, LocationRule{
			Name:      fmt.Sprintf("Project %s audit logs dataset location whitelist.", project.ID),
			Mode:      "whitelist",
			Resources: res,
			AppliesTo: []appliesTo{{Type: "dataset", ResourceIDs: []string{logDatasetID}}},
			Locations: []string{project.AuditLogs.LogsBQDataset.Location},
		})
	}

	locs := make([]string, 0, len(allLocs))
	for loc := range allLocs {
		locs = append(locs, loc)
	}
	sort.Strings(locs)

	globalRule := LocationRule{
		Name:      "Global location whitelist.",
		Mode:      "whitelist",
		Resources: []resource{globalResource(conf)},
		AppliesTo: []appliesTo{{Type: "*", ResourceIDs: []string{"*"}}},
		Locations: locs,
	}

	return append([]LocationRule{globalRule}, projectRules...), nil
}

// locationToResourceInfo is used to group locations of multiple resources by their location and type.
// e.g. {"US": {"dataset": ["p1:d1"]}}
type locationToResources map[string]resourceTypeToIDs

// resourceTypeToIDs maps a resource type to a list of ids.
type resourceTypeToIDs map[string][]string

// locations a sorted list of all locations.
func (m locationToResources) locations() []string {
	locs := make([]string, 0, len(m))
	for l := range m {
		locs = append(locs, l)
	}
	sort.Strings(locs)
	return locs
}

func (m locationToResources) addResources(project *config.Project) error {
	for _, bucket := range project.Resources.GCSBuckets {
		m.add(bucket.Location, "bucket", bucket.Name())
	}
	for _, dataset := range project.Resources.BQDatasets {
		id := fmt.Sprintf("%s:%s", project.ID, dataset.Name())
		m.add(dataset.Location, "dataset", id)
	}
	for _, instance := range project.Resources.GCEInstances {
		id, err := project.GeneratedFields.InstanceID(instance.Name())
		if err != nil {
			return err
		}
		m.add(instance.Zone, "instance", id)
	}
	return nil
}

func (m locationToResources) add(loc, typ string, ids ...string) {
	loc = strings.ToUpper(loc)
	if _, ok := m[loc]; !ok {
		m[loc] = make(resourceTypeToIDs)
	}
	m[loc][typ] = append(m[loc][typ], ids...)
}

// types returns a sorted list of types for the given location.
func (m resourceTypeToIDs) types() []string {
	var types []string
	for t := range m {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}
