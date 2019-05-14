package cft

import (
	"fmt"
)

// AllGeneratedFields defines the generated_fields block.
// AllGeneratedFields contains resource information when the resources are deployed.
// See field_generation_test for examples.
type AllGeneratedFields struct {
	Projects map[string]*GeneratedFields `json:"projects"`
	Forseti  ForsetiServiceInfo          `json:"forseti"`
}

// GeneratedFields defines the generated_fields of a single project.
type GeneratedFields struct {
	ProjectNumber         string            `json:"project_number"`
	LogSinkServiceAccount string            `json:"log_sink_service_account"`
	GCEInstanceInfoList   []GCEInstanceInfo `json:"gce_instance_info"`
}

// GCEInstanceInfo defines the generated fields for instances in a project.
type GCEInstanceInfo struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// ForsetiServiceInfo defines the generated_fields of the forseti service.
type ForsetiServiceInfo struct {
	ServiceAccount string `json:"service_account"`
	ServiceBucket  string `json:"server_bucket"`
}

// Project returns the GeneratedFields pointer with the given project ID.
func (a *AllGeneratedFields) Project(id string) (*GeneratedFields, error) {
	projGeneratedFields, ok := a.Projects[id]
	if !ok {
		return nil, fmt.Errorf("project %q does not exist in generated_fields", id)
	}
	return projGeneratedFields, nil
}

// InstanceID returns the ID of the instance with the given name.
func (g *GeneratedFields) InstanceID(name string) (string, error) {
	for _, info := range g.GCEInstanceInfoList {
		if info.Name == name {
			return info.ID, nil
		}
	}
	return "", fmt.Errorf("info for instance %q not found in generated_fields", name)
}
