package cft

import (
	"fmt"
	"os"
	"os/exec"
)

// TODO check if go-client works in this case.

// InstallClusterWorkloadFromFile creates and updates (when it exists) not only workloads
// but also all resources supported by "kubectl apply -f".
func InstallClusterWorkloadFromFile(projectID, clusterName, region, containerYamlPath string) error {

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
