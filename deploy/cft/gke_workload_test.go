package cft

import (
	"os/exec"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetGCloudCredentials(t *testing.T) {
	region := "foo-center"
	clusterName := "bar-cluster"
	projectID := "foo-project"
	var gotArgs [][]string
	cmdRun = func(cmd *exec.Cmd) error {
		gotArgs = append(gotArgs, cmd.Args)
		return nil
	}
	wantArgs := [][]string{{
		"gcloud", "container", "clusters", "get-credentials", clusterName, "--region", region, "--project", projectID}}
	if err := getGCloudCredentials(clusterName, region, projectID); err != nil {
		t.Fatalf("getGCloudCredentials error: %v", err)
	}
	if diff := cmp.Diff(gotArgs, wantArgs); len(diff) != 0 {
		t.Fatalf("getGCloudCredentials commands differ: (-got, +want)\n:%v", diff)
	}
}

func TestApplyClusterResource(t *testing.T) {
	containerYamlPath := "foo/bar/abc.yaml"
	var gotArgs [][]string
	cmdRun = func(cmd *exec.Cmd) error {
		gotArgs = append(gotArgs, cmd.Args)
		return nil
	}
	wantArgs := [][]string{{
		"kubectl", "apply", "-f", containerYamlPath}}
	if err := applyClusterWorkload(containerYamlPath); err != nil {
		t.Fatalf("applyClusterWorkload error: %v", err)
	}
	if diff := cmp.Diff(gotArgs, wantArgs); len(diff) != 0 {
		t.Fatalf("applyClusterWorkload commands differ: (-got, +want)\n:%v", diff)
	}
}

func TestGetGKEWorkload(t *testing.T) {
	configExtend := &ConfigData{`
resources:
- gke_workload:
    cluster_name: cluster1
    properties:
      apiVersion: extensions/v1beta1
      kind: Deployment
- gke_workload:
    cluster_name: cluster2
    properties:
      apiVersion: extensions/v1beta1
      kind: Service`,
	}
	_, project := getTestConfigAndProject(t, configExtend)
	workloads, err := getGKEWorkloads(project)
	if err != nil {
		t.Fatalf("getGKEWorkloads: %v", err)
	}
	if len(workloads) != 2 {
		t.Fatalf("workload len error: %v", len(workloads))
	}
	if workloads[0].ClusterName != "cluster1" || workloads[1].ClusterName != "cluster2" {
		t.Fatalf("workload context error: %v", workloads)
	}
}
