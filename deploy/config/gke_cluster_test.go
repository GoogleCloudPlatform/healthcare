package config_test

import (
	"testing"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/ghodss/yaml"
)

func TestGKEClusterRegion(t *testing.T) {
	resYAML := `
properties:
  name: foo-cluser
  clusterLocationType: Regional
  region: somewhere1
`

	cluser := new(config.GKECluster)
	if err := yaml.Unmarshal([]byte(resYAML), cluser); err != nil {
		t.Fatalf("yaml unmarshal: %v", err)
	}
	if cluser.ResourceName != "foo-cluser" {
		t.Fatalf("cluster name error: %v", cluser.ResourceName)
	}
	if cluser.ClusterLocationType != "Regional" {
		t.Fatalf("cluster location type error: %v", cluser.ClusterLocationType)
	}
	if cluser.Region != "somewhere1" {
		t.Fatalf("cluster region error: %v", cluser.ClusterLocationType)
	}
}

func TestGKEClusterZone(t *testing.T) {
	resYAML := `
properties:
  name: foo-cluser
  clusterLocationType: Zonal
  zone: somewhere2-c
`

	cluser := new(config.GKECluster)
	if err := yaml.Unmarshal([]byte(resYAML), cluser); err != nil {
		t.Fatalf("yaml unmarshal: %v", err)
	}
	if cluser.ResourceName != "foo-cluser" {
		t.Fatalf("cluster name error: %v", cluser.ResourceName)
	}
	if cluser.ClusterLocationType != "Zonal" {
		t.Fatalf("cluster location type error: %v", cluser.ClusterLocationType)
	}
	if cluser.Zone != "somewhere2-c" {
		t.Fatalf("cluster zone error: %v", cluser.Zone)
	}
}
