package config_test

import (
	"testing"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/ghodss/yaml"
)

func TestGKEClusterRegion(t *testing.T) {
	resYAML := `
properties:
  clusterLocationType: Regional
  region: somewhere1
  cluster:
    name: foo-cluser
`

	cluster := new(config.GKECluster)
	if err := yaml.Unmarshal([]byte(resYAML), cluster); err != nil {
		t.Fatalf("yaml unmarshal: %v", err)
	}
	if err := cluster.Init(); err != nil {
		t.Fatalf("cluster.Init = %v", err)
	}
	if cluster.Name() != "foo-cluser" {
		t.Fatalf("cluster name error: %v", cluster.Name())
	}
	if cluster.ClusterLocationType != "Regional" {
		t.Fatalf("cluster location type error: %v", cluster.ClusterLocationType)
	}
	if cluster.Region != "somewhere1" {
		t.Fatalf("cluster region error: %v", cluster.ClusterLocationType)
	}
}

func TestGKEClusterZone(t *testing.T) {
	resYAML := `
properties:
  clusterLocationType: Zonal
  zone: somewhere2-c
  cluster:
    name: foo-cluser
`

	cluster := new(config.GKECluster)
	if err := yaml.Unmarshal([]byte(resYAML), cluster); err != nil {
		t.Fatalf("yaml unmarshal: %v", err)
	}
	if err := cluster.Init(); err != nil {
		t.Fatalf("cluster.Init = %v", err)
	}
	if cluster.Name() != "foo-cluser" {
		t.Fatalf("cluster name error: %v", cluster.Name())
	}
	if cluster.ClusterLocationType != "Zonal" {
		t.Fatalf("cluster location type error: %v", cluster.ClusterLocationType)
	}
	if cluster.Zone != "somewhere2-c" {
		t.Fatalf("cluster zone error: %v", cluster.Zone)
	}
}
