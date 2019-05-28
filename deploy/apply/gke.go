package apply

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"github.com/GoogleCloudPlatform/healthcare/deploy/cft"
)

// deployGKEWorkloads deploys the GKE resources (e.g., workloads, services) in the project.
func deployGKEWorkloads(project *cft.Project) error {
	for _, w := range project.Resources.GKEWorkloads {
		if err := installClusterWorkload(w.ClusterName, project, w.Properties); err != nil {
			return fmt.Errorf("failed to deploy workload: %v", err)
		}
	}
	return nil
}

// installClusterWorkload creates and updates (when it exists) not only workloads
// but also all resources supported by "kubectl apply -f". Data comes from a GKEWorkload struct.
func installClusterWorkload(clusterName string, project *cft.Project, workload interface{}) error {
	b, err := json.Marshal(workload)
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
	return installClusterWorkloadFromFile(clusterName, tmp.Name(), project)
}

// installClusterWorkloadFromFile creates and updates (when it exists) not only workloads
// but also all resources supported by "kubectl apply -f".
func installClusterWorkloadFromFile(clusterName, containerYamlPath string, project *cft.Project) error {
	cluster := getClusterByName(project, clusterName)
	if cluster == nil {
		return fmt.Errorf("failed to find cluster: %q", clusterName)
	}
	locationType, locationValue, err := getLocationTypeAndValue(cluster)
	if err != nil {
		return err
	}
	if err := getGCloudCredentials(clusterName+"-cluster", locationType, locationValue, project.ID); err != nil {
		return err
	}
	if err := applyClusterWorkload(containerYamlPath); err != nil {
		return err
	}
	return nil
}

// getClusterByName get a cluster that has the given cluster name in a project.
func getClusterByName(project *cft.Project, clusterName string) *cft.GKECluster {
	for _, c := range project.Resources.GKEClusters {
		if c.Parsed.Name() == clusterName {
			return &c.Parsed
		}
	}
	return nil
}

func getLocationTypeAndValue(cluster *cft.GKECluster) (string, string, error) {
	switch cluster.ClusterLocationType {
	case "Regional":
		if cluster.Region == "" {
			return "", "", fmt.Errorf("failed to get cluster's region: %v", cluster.ResourceName)
		}
		return "--region", cluster.Region, nil
	case "Zonal":
		if cluster.Zone == "" {
			return "", "", fmt.Errorf("failed to get cluster's zone: %v", cluster.ResourceName)
		}
		return "--zone", cluster.Zone, nil
	default:
		return "", "", fmt.Errorf("failed to get cluster's location: %v", cluster.ResourceName)
	}
}

func getGCloudCredentials(clusterName, locationType, locationValue, projectID string) error {
	cmd := exec.Command("gcloud", "container", "clusters", "get-credentials", clusterName, locationType, locationValue, "--project", projectID)
	if err := cmdRun(cmd); err != nil {
		return fmt.Errorf("failed to get cluster credentials for %q: %v", clusterName, err)
	}
	return nil
}

func applyClusterWorkload(containerYamlPath string) error {
	// kubectl declarative object configuration
	// https://kubernetes.io/docs/concepts/overview/object-management-kubectl/overview/
	cmd := exec.Command("kubectl", "apply", "-f", containerYamlPath)
	if err := cmdRun(cmd); err != nil {
		return fmt.Errorf("failed to apply workloads with kubectl: %s", err)
	}
	return nil
}
