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

package apply

import (
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/GoogleCloudPlatform/healthcare/deploy/testconf"
	"github.com/google/go-cmp/cmp"
)

func TestGetGCloudCredentials(t *testing.T) {
	region := "foo-center"
	clusterName := "bar-cluster"
	projectID := "foo-project"
	wantCmd := strings.Join([]string{"gcloud", "container", "clusters", "get-credentials", clusterName, "--region", region, "--project", projectID}, " ")
	r := &testRunner{}
	if err := getGCloudCredentials(clusterName, "--region", region, projectID, r); err != nil {
		t.Fatalf("getGCloudCredentials error: %v", err)
	}
	if len(r.called) != 1 {
		t.Fatalf("getGCloudCredentials did not run correct number of commands: got %d, want 1", len(r.called))
	}
	if diff := cmp.Diff(r.called[0], wantCmd); len(diff) != 0 {
		t.Fatalf("getGCloudCredentials commands differ: (-got, +want)\n:%v", diff)
	}
}

func TestApplyClusterResource(t *testing.T) {
	containerYamlPath := "foo/bar/abc.yaml"
	wantCmd := strings.Join([]string{"kubectl", "apply", "-f", containerYamlPath}, " ")
	r := &testRunner{}
	if err := applyClusterWorkload(containerYamlPath, r); err != nil {
		t.Fatalf("applyClusterWorkload error: %v", err)
	}
	if len(r.called) != 1 {
		t.Fatalf("applyClusterWorkload did not run correct number of commands: got %d, want 1", len(r.called))
	}
	if diff := cmp.Diff(r.called[0], wantCmd); len(diff) != 0 {
		t.Fatalf("applyClusterWorkload commands differ: (-got, +want)\n:%v", diff)
	}
}

func TestLocationTypeAndValue(t *testing.T) {
	testcases := []struct {
		in            config.GKECluster
		locationType  string
		locationValue string
	}{
		{
			in: config.GKECluster{GKEClusterProperties: config.GKEClusterProperties{
				ClusterLocationType: "Regional",
				Region:              "some_region",
				Cluster:             config.GKEClusterSettings{"cluster_with_region"},
			}},
			locationType:  "--region",
			locationValue: "some_region",
		},
		{
			in: config.GKECluster{GKEClusterProperties: config.GKEClusterProperties{
				ClusterLocationType: "Zonal",
				Zone:                "some_zone",
				Cluster:             config.GKEClusterSettings{"cluster_with_zone"},
			}},
			locationType:  "--zone",
			locationValue: "some_zone",
		},
	}

	for _, tc := range testcases {
		locationType, locationValue, err := getLocationTypeAndValue(&tc.in)
		if err != nil {
			t.Errorf("getLocationTypeAndValue error at cluster %q", tc.in.Name())
		}
		if locationType != tc.locationType {
			t.Errorf("getLocationTypeAndValue locationType error at cluster %q: %q", tc.in.Name(), locationType)
		}
		if locationValue != tc.locationValue {
			t.Errorf("getLocationTypeAndValue locationValue error at cluster %q: %q", tc.in.Name(), locationValue)
		}
	}
}

func TestInstallClusterWorkload(t *testing.T) {
	configExtend := &testconf.ConfigData{`
resources:
  gke_clusters:
  - properties:
      name: cluster1
      clusterLocationType: Regional
      region: somewhere1
      cluster:
        name: cluster1
  gke_workloads:
  - cluster_name: cluster1
    properties:
      apiVersion: extensions/v1beta1`,
	}

	wantCmds := []string{
		"gcloud container clusters get-credentials cluster1 --region somewhere1 --project my-project",
		"kubectl apply -f",
	}

	_, project := testconf.ConfigAndProject(t, configExtend)
	r := &testRunner{}
	if err := deployGKEWorkloads(project, r); err != nil {
		t.Fatalf("deployGKEWorkloads error: %v", err)
	}
	if len(r.called) != 2 {
		t.Fatalf("deployGKEWorkloads does not run correct number of commands: got %d, want 2", len(r.called))
	}
	if diff := cmp.Diff(r.called[0], wantCmds[0]); len(diff) != 0 {
		t.Fatalf("credentials commands differ: (-got, +want)\n:%v", diff)
	}
	if !strings.Contains(r.called[1], wantCmds[1]) {
		t.Fatalf("kubectl commands differ: got %q, want contains %q", r.called[1], wantCmds[1])
	}
}

func TestLocationTypeAndValueError(t *testing.T) {
	testcases := []struct {
		in  config.GKECluster
		err string
	}{
		{
			in: config.GKECluster{GKEClusterProperties: config.GKEClusterProperties{
				ClusterLocationType: "Zonal",
				Region:              "some_region",
				Zone:                "",
				Cluster:             config.GKEClusterSettings{"cluster_zonal_error"},
			}},
			err: "failed to get cluster's zone: cluster_zonal_error",
		},
		{
			in: config.GKECluster{GKEClusterProperties: config.GKEClusterProperties{
				ClusterLocationType: "Regional",
				Zone:                "some_zone",
				Cluster:             config.GKEClusterSettings{"cluster_regional_error"},
			}},
			err: "failed to get cluster's region: cluster_regional_error",
		},
		{
			in: config.GKECluster{GKEClusterProperties: config.GKEClusterProperties{
				ClusterLocationType: "Location",
				Region:              "some_region",
				Zone:                "some_zone",
				Cluster:             config.GKEClusterSettings{"cluster_wrong_type"},
			}},
			err: "failed to get cluster's location: cluster_wrong_type",
		},
	}

	for _, tc := range testcases {
		_, _, err := getLocationTypeAndValue(&tc.in)
		if err == nil || !strings.Contains(err.Error(), tc.err) {
			t.Errorf("getLocationTypeAndValue for cluster %q: got %q, want error with substring %q", tc.in.Name(), err, tc.err)
		}
	}
}

func TestInstallClusterWorkloadErrors(t *testing.T) {
	testcases := []struct {
		in  testconf.ConfigData
		err string
	}{
		{
			in: testconf.ConfigData{`
resources:
  gke_clusters:
  - properties:
      name: cluster1
      clusterLocationType: Regional
      region: somewhere1
      cluster:
        name: cluster1
  gke_workloads:
  - cluster_name: clusterX
    properties:
      apiVersion: extensions/v1beta1`,
			},
			err: "failed to find cluster: \"clusterX\"",
		},
		{
			in: testconf.ConfigData{`
resources:
  gke_clusters:
  - properties:
      name: cluster1
      clusterLocationType: Location
      region: somewhere1
      cluster:
        name: cluster1
  gke_workloads:
  - cluster_name: cluster1
    properties:
      apiVersion: extensions/v1beta1`,
			},
			err: "failed to get cluster's location: cluster1",
		},
	}

	for _, tc := range testcases {
		_, project := testconf.ConfigAndProject(t, &tc.in)
		err := deployGKEWorkloads(project, &testRunner{})
		if err == nil || !strings.Contains(err.Error(), tc.err) {
			t.Errorf("deployGKEWorkloads unexpected error: got %q, want error with substring %q", err, tc.err)
		}
	}
}

func TestImportBinaryAuthorizationPolicy(t *testing.T) {
	configExtend := &testconf.ConfigData{`
binauthz:
  properties: {}`}

	wantArgSegs := []string{"gcloud beta container binauthz policy import", "--project my-project"}

	_, project := testconf.ConfigAndProject(t, configExtend)
	if project.BinauthzPolicy == nil {
		return
	}
	r := &testRunner{}
	if err := importBinauthz(project.ID, project.BinauthzPolicy, r); err != nil {
		t.Fatalf("importBinauthz error: %v", err)
	}
	if len(r.called) != 1 {
		t.Fatalf("importBinauthz does not run correct number of commands: got %d, want 1", len(r.called))
	}
	for _, s := range wantArgSegs {
		if !strings.Contains(r.called[0], s) {
			t.Errorf("binauthz commands does not contain wanted args: got %q, want %q", r.called[0], s)
		}
	}
}
