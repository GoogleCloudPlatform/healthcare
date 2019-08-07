// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
