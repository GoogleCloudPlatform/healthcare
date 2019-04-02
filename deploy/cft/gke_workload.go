package cft

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"google3/third_party/golang/yaml/yaml"
)

// GKEWorkload represents a GKE resources, not limited to workloads.
type GKEWorkload struct {
	Properties  interface{} `json:"properties"`
	ClusterName string      `json:"cluster_name"`
}

// TODO check if go-client works in this case.

// installClusterWorkloadFromFile creates and updates (when it exists) not only workloads
// but also all resources supported by "kubectl apply -f".
func installClusterWorkloadFromFile(projectID, clusterName, region, containerYamlPath string) error {

	if err := getGCloudCredentials(clusterName, region, projectID); err != nil {
		return err
	}
	if err := applyClusterWorkload(containerYamlPath); err != nil {
		return err
	}
	return nil
}

func getGCloudCredentials(clusterName, region, projectID string) error {
	cmd := exec.Command("gcloud", "container", "clusters", "get-credentials", clusterName, "--region", region, "--project", projectID)
	cmd.Stderr = os.Stderr
	if err := cmdRun(cmd); err != nil {
		return fmt.Errorf("failed to get cluster credentials for %q: %v", clusterName, err)
	}
	return nil
}

func applyClusterWorkload(containerYamlPath string) error {
	// kubectl declarative object configuration
	// https://kubernetes.io/docs/concepts/overview/object-management-kubectl/overview/
	cmd := exec.Command("kubectl", "apply", "-f", containerYamlPath)
	cmd.Stderr = os.Stderr
	if err := cmdRun(cmd); err != nil {
		return fmt.Errorf("failed to apply workloads with kubectl: %s", err)
	}
	return nil
}

// InstallClusterWorkload creates and updates (when it exists) not only workloads
// but also all resources supported by "kubectl apply -f". Data comes from a GKEWorkload struct.
func InstallClusterWorkload(projectID, clusterName, region string, workload interface{}) error {
	b, err := yaml.Marshal(workload)
	if err != nil {
		return fmt.Errorf("failed to marshal workload : %v", err)
	}
	log.Printf("Creating workload:\n%v", string(b))

	tmp, err := ioutil.TempFile("", "")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(b); err != nil {
		return fmt.Errorf("failed to write deployment to file: %v", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %v", err)
	}
	return installClusterWorkloadFromFile(projectID, clusterName, region, tmp.Name())
}

// getClusterRegion gets the region where a given cluster locates.
func getClusterRegion(*Project, string) (string, error) {
	// TODO this function will be implemented after CL 240762841 (ruihuang)
	return "us-central1", nil
}

// DeployGKEWorkloads deploys the GKE resources (e.g., workloads, services) in the project.
func DeployGKEWorkloads(project *Project) error {
	workloads, err := getGKEWorkloads(project)
	if err != nil {
		return err
	}

	for _, workload := range workloads {
		region, err := getClusterRegion(project, workload.ClusterName)
		if err != nil {
			return err
		}
		InstallClusterWorkload(project.ID, workload.ClusterName, region, workload.Properties)
	}
	return nil
}

// getGKEWorkloads returns the list of all GKE resources.
func getGKEWorkloads(project *Project) ([]GKEWorkload, error) {
	var workloads []GKEWorkload

	for _, r := range project.Resources {
		if r.GKEWorkload != nil {
			var newWorkload GKEWorkload
			if err := unmarshal(r.GKEWorkload, &newWorkload); err != nil {
				return nil, fmt.Errorf("failed to unmarshal %q: %v", newWorkload, err)
			}
			workloads = append(workloads, newWorkload)
		}

	}
	return workloads, nil
}
