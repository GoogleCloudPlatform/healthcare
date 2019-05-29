package config

import (
	"encoding/json"
	"errors"
	"fmt"
)

// GCSBucket wraps a CFT Cloud Storage Bucket.
type GCSBucket struct {
	GCSBucketProperties `json:"properties"`
	TTLDays             int      `json:"ttl_days,omitempty"`
	ExpectedUsers       []string `json:"expected_users,omitempty"`
}

// GCSBucketProperties  represents a partial CFT bucket implementation.
type GCSBucketProperties struct {
	GCSBucketName              string     `json:"name"`
	Location                   string     `json:"location"`
	Bindings                   []Binding  `json:"bindings"`
	StorageClass               string     `json:"storageClass,omitempty"`
	Versioning                 versioning `json:"versioning"`
	Lifecycle                  *lifecycle `json:"lifecycle,omitempty"`
	PredefinedACL              string     `json:"predefinedAcl,omitempty"`
	PredefinedDefaultObjectACL string     `json:"predefinedDefaultObjectAcl,omitempty"`
	Logging                    struct {
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
	if b.PredefinedACL != "" || b.PredefinedDefaultObjectACL != "" {
		return errors.New("predefined ACLs must not be set")
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
				Action:    &action{Type: "Delete"},
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
	return "deploy/config/templates/gcs_bucket/gcs_bucket.py"
}
