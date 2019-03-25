package cft

import (
	"errors"
	"fmt"
)

// GCSBucket wraps a CFT Cloud Storage Bucket.
// TODO: set logging bucket ID
// TODO: add support for expected users + creating unexpected users metrics.
type GCSBucket struct {
	GCSBucketProperties `yaml:"properties"`
	NoPrefix            bool `yaml:"no_prefix,omitempty"`
}

// GCSBucketProperties  represents a partial CFT bucket implementation.
type GCSBucketProperties struct {
	GCSBucketName string     `yaml:"name"`
	Bindings      []binding  `yaml:"bindings"`
	Versioning    versioning `yaml:"versioning"`
}

type binding struct {
	Role    string   `yaml:"role"`
	Members []string `yaml:"members"`
}

type versioning struct {
	Enabled bool `yaml:"enabled"`
}

// NewGCSBucket creates a new default bucket.
func NewGCSBucket() *GCSBucket {
	return &GCSBucket{
		GCSBucketProperties: GCSBucketProperties{
			Versioning: versioning{
				Enabled: true,
			},
		},
	}
}

// Init initializes the bucket with the given project.
func (b *GCSBucket) Init(project *Project) error {
	if b.GCSBucketName == "" {
		return errors.New("name must be set")
	}
	if !b.Versioning.Enabled {
		return errors.New("versioning must not be disabled")
	}

	if !b.NoPrefix {
		b.GCSBucketName = fmt.Sprintf("%s-%s", project.ID, b.GCSBucketName)
	}

	b.Bindings = b.getBindings(project)
	return nil
}

func (b *GCSBucket) getBindings(project *Project) []binding {
	appendGroupPrefix := func(ss ...string) []string {
		res := make([]string, 0, len(ss))
		for _, s := range ss {
			res = append(res, "group:"+s)
		}
		return res
	}

	// Note: duplicate bindings are de-duplicated by deployment manager.
	defaultBindings := []binding{
		{"roles/storage.admin", appendGroupPrefix(project.OwnersGroup)},
		{"roles/storage.objectAdmin", appendGroupPrefix(project.DataReadWriteGroups...)},
		{"roles/storage.objectViewer", appendGroupPrefix(project.DataReadOnlyGroups...)},
	}

	roleToMembers := make(map[string][]string)

	allBindings := append(defaultBindings, b.Bindings...)
	var roles []string // preserve ordering

	for _, b := range allBindings {
		if _, ok := roleToMembers[b.Role]; !ok {
			roles = append(roles, b.Role)
		}
		roleToMembers[b.Role] = append(roleToMembers[b.Role], b.Members...)
	}

	var merged []binding
	for _, role := range roles {
		merged = append(merged, binding{Role: role, Members: roleToMembers[role]})
	}
	return merged
}

// Name returns the name of the bucket.
func (b *GCSBucket) Name() string {
	return b.GCSBucketName
}

// TemplatePath returns the name of the template to use for the bucket.
func (b *GCSBucket) TemplatePath() string {
	return "gcs_bucket.py"
}
