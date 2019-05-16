package cft

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"text/template"
)

var metricFilterTemplate = template.Must(template.New("metricFilter").Parse(`resource.type=gcs_bucket AND
logName=projects/{{.Project.ID}}/logs/cloudaudit.googleapis.com%2Fdata_access AND
protoPayload.resourceName=projects/_/buckets/{{.Bucket.Name}} AND
protoPayload.status.code!=7 AND
protoPayload.authenticationInfo.principalEmail!=({{.ExpectedUsers}})
`))

// GCSBucket wraps a CFT Cloud Storage Bucket.
type GCSBucket struct {
	GCSBucketProperties `json:"properties"`
	TTLDays             int      `json:"ttl_days,omitempty"`
	ExpectedUsers       []string `json:"expected_users,omitempty"`
}

// GCSBucketProperties  represents a partial CFT bucket implementation.
type GCSBucketProperties struct {
	GCSBucketName string     `json:"name"`
	Location      string     `json:"location"`
	Bindings      []Binding  `json:"bindings"`
	Versioning    versioning `json:"versioning"`
	Lifecycle     *lifecycle `json:"lifecycle,omitempty"`
	Logging       struct {
		LogBucket string `json:"logBucket"`
	} `json:"logging"`
}

type versioning struct {
	// Use pointer to differentiate between zero value and intentionally being set to false.
	Enabled *bool `json:"enabled"`
}

type lifecycle struct {
	// implement as pair so user-defined rules are not lost
	Rules []lifecycleRulePair `json:"rule,omitempty"`
}

type lifecycleRulePair struct {
	raw    json.RawMessage
	parsed lifecycleRule
}

type lifecycleRule struct {
	Action    *action    `json:"action,omitempty"`
	Condition *condition `json:"condition,omitempty"`
}

type action struct {
	Type string `json:"type,omitempty"`
}

type condition struct {
	Age    int  `json:"age,omitempty"`
	IsLive bool `json:"isLive,omitempty"`
}

func (p *lifecycleRulePair) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &p.raw); err != nil {
		return fmt.Errorf("failed to unmarshal lifecycle rule into raw form: %v", err)
	}
	if err := json.Unmarshal(data, &p.parsed); err != nil {
		return fmt.Errorf("failed to unmarshal lifecycle rule into parsed form: %v", err)
	}
	return nil
}

func (p *lifecycleRulePair) MarshalJSON() ([]byte, error) {
	return interfacePair{p.raw, p.parsed}.MarshalJSON()
}

// Init initializes the bucket with the given project.
func (b *GCSBucket) Init(project *Project) error {
	if b.GCSBucketName == "" {
		return errors.New("name must be set")
	}
	if b.Location == "" {
		return errors.New("location must be set")
	}
	if b.Versioning.Enabled != nil && !*b.Versioning.Enabled {
		return errors.New("versioning must not be disabled")
	}

	t := true
	b.Versioning.Enabled = &t

	appendGroupPrefix := func(ss ...string) []string {
		res := make([]string, 0, len(ss))
		for _, s := range ss {
			res = append(res, "group:"+s)
		}
		return res
	}

	// Note: duplicate bindings are de-duplicated by deployment manager.
	bindings := []Binding{
		{Role: "roles/storage.admin", Members: appendGroupPrefix(project.OwnersGroup)},
	}
	if len(project.DataReadWriteGroups) > 0 {
		bindings = append(bindings, Binding{
			Role: "roles/storage.objectAdmin", Members: appendGroupPrefix(project.DataReadWriteGroups...),
		})
	}
	if len(project.DataReadOnlyGroups) > 0 {
		bindings = append(bindings, Binding{
			Role: "roles/storage.objectViewer", Members: appendGroupPrefix(project.DataReadOnlyGroups...),
		})
	}

	b.Bindings = MergeBindings(append(bindings, b.Bindings...)...)

	// TODO: this shouldn't be possible (data buckets should imply log bucket exists).
	if project.AuditLogs.LogsGCSBucket == nil {
		return nil
	}
	if project.AuditLogs.LogsGCSBucket != nil {
		b.Logging.LogBucket = project.AuditLogs.LogsGCSBucket.Name()
	}

	if b.TTLDays > 0 {
		if b.Lifecycle == nil {
			b.Lifecycle = &lifecycle{}
		}
		b.Lifecycle.Rules = append(b.Lifecycle.Rules, lifecycleRulePair{
			parsed: lifecycleRule{
				Action:    &action{Type: "DELETE"},
				Condition: &condition{Age: b.TTLDays, IsLive: true},
			},
		})
	}
	return nil
}

// Name returns the name of the bucket.
func (b *GCSBucket) Name() string {
	return b.GCSBucketName
}

// TemplatePath returns the name of the template to use for the bucket.
func (b *GCSBucket) TemplatePath() string {
	return "deploy/cft/templates/gcs_bucket/gcs_bucket.py"
}

// DependentResources gets the dependent resources of this bucket.
// If the bucket has expected users, this list will contain a metric that will detect unexpected
// access to the bucket from users not in the expected users list.
func (b *GCSBucket) DependentResources(project *Project) ([]parsedResource, error) {
	if len(b.ExpectedUsers) == 0 {
		return nil, nil
	}

	var buf bytes.Buffer
	data := struct {
		Project       *Project
		Bucket        *GCSBucket
		ExpectedUsers string
	}{
		project,
		b,
		strings.Join(b.ExpectedUsers, " AND "),
	}
	if err := metricFilterTemplate.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute filter template: %v", err)
	}

	m := &Metric{
		MetricProperties: MetricProperties{
			MetricName:  "unexpected-access-" + b.Name(),
			Description: "Count of unexpected data access to " + b.Name(),
			Filter:      buf.String(),
			Descriptor:  unexpectedUserDescriptor,
			LabelExtractors: principalEmailLabelExtractor,
		},
	}
	return []parsedResource{m}, nil
}
