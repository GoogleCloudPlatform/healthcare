package cft

import (
	"testing"

	"github.com/ghodss/yaml"
)

func TestGKEClusterRegion(t *testing.T) {
	resYAML := `
properties:
  name: foo-cluser
  clusterLocationType: Regional
  region: somewhere1
`

	cluser := new(GKECluster)
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

	cluser := new(GKECluster)
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

func TestGetParsedClusters(t *testing.T) {
	configExtend := &ConfigData{`
resources:
- gke_cluster:
    properties:
      name: cluster1
      clusterLocationType: Regional
      region: somewhere1
- gke_cluster:
    properties:
      name: cluster2
      clusterLocationType: Zonal
      zone: somewhere2-c`,
	}
	_, project := getTestConfigAndProject(t, configExtend)
	cluster := getClusterByName(project, "cluster1")
	if cluster == nil {
		t.Fatalf("getClusterByName cannot find cluster")
	}
	if cluster.Name() != "cluster1" {
		t.Fatalf("getClusterByName find a wrong cluster: %v", cluster.Name())
	}
	cluster = getClusterByName(project, "cluster2")
	if cluster == nil {
		t.Fatalf("getClusterByName cannot find cluster")
	}
	if cluster.Name() != "cluster2" {
		t.Fatalf("getClusterByName find a wrong cluster: %v", cluster.Name())
	}
}
