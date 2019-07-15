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

	cluser := new(config.GKECluster)
	if err := yaml.Unmarshal([]byte(resYAML), cluser); err != nil {
		t.Fatalf("yaml unmarshal: %v", err)
	}
	if cluser.Name() != "foo-cluser" {
		t.Fatalf("cluster name error: %v", cluser.Name())
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
  clusterLocationType: Zonal
  zone: somewhere2-c
  cluster:
    name: foo-cluser
`

	cluser := new(config.GKECluster)
	if err := yaml.Unmarshal([]byte(resYAML), cluser); err != nil {
		t.Fatalf("yaml unmarshal: %v", err)
	}
	if cluser.Name() != "foo-cluser" {
		t.Fatalf("cluster name error: %v", cluser.Name())
	}
	if cluser.ClusterLocationType != "Zonal" {
		t.Fatalf("cluster location type error: %v", cluser.ClusterLocationType)
	}
	if cluser.Zone != "somewhere2-c" {
		t.Fatalf("cluster zone error: %v", cluser.Zone)
	}
}

func TestRegionString(t *testing.T) {
	tests := []struct {
		yaml string
		want string
	}{
		{
			yaml: `
properties:
  clusterLocationType: Zonal
  zone: some-where1-x
  cluster:
    name: foo-cluser
`,
			want: "some-where1",
		},
		{
			yaml: `
properties:
  clusterLocationType: Zonal
  region: some-where2
  cluster:
    name: foo-cluser
`,
			want: "some-where2",
		},
	}

	for _, testcase := range tests {
		cluster := new(config.GKECluster)
		if err := yaml.Unmarshal([]byte(testcase.yaml), cluster); err != nil {
			t.Fatalf("yaml unmarshal: %v", err)
		}
		gotRegion, err := cluster.RegionString()
		if err != nil {
			t.Fatalf("function RegionString unexpected error %v", err)
		}
		if gotRegion != testcase.want {
			t.Fatalf("function RegionString error: got %v; expect %v", gotRegion, testcase.want)
		}
	}
}

func TestRegionStringErr(t *testing.T) {
	tests := []struct {
		yaml string
		name string
	}{
		{
			name: "zone is not correct",
			yaml: `
properties:
  clusterLocationType: Zonal
  zone: some-where1
  cluster:
    name: foo-cluser
`,
		},
		{
			name: "Neither zone nor region exist",
			yaml: `
properties:
  clusterLocationType: Zonal
  cluster:
    name: foo-cluser
`,
		},
	}

	for _, testcase := range tests {
		cluster := new(config.GKECluster)
		if err := yaml.Unmarshal([]byte(testcase.yaml), cluster); err != nil {
			t.Fatalf("yaml unmarshal: %v", err)
		}
		_, err := cluster.RegionString()
		if err == nil {
			t.Fatalf("function RegionString on %v should have error", testcase.name)
		}
	}
}
