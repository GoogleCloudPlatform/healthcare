package apply

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
)

// deployGKEWorkloads deploys the GKE resources (e.g., workloads, services) in the project.
func deployGKEWorkloads(project *config.Project) error {
	for _, w := range project.Resources.GKEWorkloads {
		if err := installClusterWorkload(w.ClusterName, project, w.Properties); err != nil {
			return fmt.Errorf("failed to deploy workload: %v", err)
		}
	}
	return nil
}

// installClusterWorkload creates and updates (when it exists) not only workloads
// but also all resources supported by "kubectl apply -f". Data comes from a GKEWorkload struct.
func installClusterWorkload(clusterName string, project *config.Project, workload interface{}) error {
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
func installClusterWorkloadFromFile(clusterName, containerYamlPath string, project *config.Project) error {
	cluster := getClusterByName(project, clusterName)
	if cluster == nil {
		return fmt.Errorf("failed to find cluster: %q", clusterName)
	}
	locationType, locationValue, err := getLocationTypeAndValue(cluster)
	if err != nil {
		return err
	}
	if err := getGCloudCredentials(clusterName, locationType, locationValue, project.ID); err != nil {
		return err
	}
	if err := applyClusterWorkload(containerYamlPath); err != nil {
		return err
	}
	return nil
}

// getClusterByName get a cluster that has the given cluster name in a project.
func getClusterByName(project *config.Project, clusterName string) *config.GKECluster {
	for _, c := range project.Resources.GKEClusters {
		if c.Parsed.Name() == clusterName {
			return &c.Parsed
		}
	}
	return nil
}

func getLocationTypeAndValue(cluster *config.GKECluster) (string, string, error) {
	switch cluster.ClusterLocationType {
	case "Regional":
		if cluster.Region == "" {
			return "", "", fmt.Errorf("failed to get cluster's region: %v", cluster.Name())
		}
		return "--region", cluster.Region, nil
	case "Zonal":
		if cluster.Zone == "" {
			return "", "", fmt.Errorf("failed to get cluster's zone: %v", cluster.Name())
		}
		return "--zone", cluster.Zone, nil
	default:
		return "", "", fmt.Errorf("failed to get cluster's location: %v", cluster.Name())
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

// askForConfirm asks users to confirm
func askForConfirm() (bool, error) {
	fmt.Print("To continue press \"Y\", or \"n\" to exit. (Y/n)")
	var input string
	if _, err := fmt.Scan(&input); err != nil {
		return false, fmt.Errorf("failed to get user confirm: %s", err)
	}
	if input == "Y" {
		return true, nil
	}
	return false, nil
}

// CheckGKEConfigs check if the settings of a GKE cluster has errors that schema cannot tell.
func CheckGKEConfigs(project *config.Project) error {
	prompt, err := validateGKEConfigs(project)
	if err != nil {
		return fmt.Errorf("failed when checking GKE configs: %s", err)
	}
	if prompt != "" {
		log.Print("The initClusterVersion of the following cluster(s) might not be set correctly:\n" + prompt)
		confirm, err := askForConfirm()
		if err != nil {
			return fmt.Errorf("failed when checking GKE configs: %s", err)
		}
		if !confirm {
			return fmt.Errorf("user chooses to exit to correct YAML")
		}
	}
	return nil
}

// validateGKEConfigs gets a string that describe if there are clusters with wrong initClusterVersion
func validateGKEConfigs(project *config.Project) (string, error) {
	defaultVersions := make(map[string]string)
	violationClusterVersions := make(map[string]string)
	prompt := ""
	for _, cluster := range project.Resources.GKEClusters {
		locationType, locationValue, err := getLocationTypeAndValue(&cluster.Parsed)
		if err != nil {
			return "", fmt.Errorf("failed when validating GKE configs: %s", err)
		}
		var defaultVersion string
		var ok bool
		if defaultVersion, ok = defaultVersions[locationValue]; !ok {
			var err error
			defaultVersion, err = defaultGKEClusterVersion(locationType, locationValue)
			if err != nil {
				return "", fmt.Errorf("failed when validating GKE configs: %s", err)
			}
		}
		// prompt := "The initClusterVersion of the following cluster(s) might not be set correctly:\n"
		if defaultVersion != cluster.Parsed.Cluster.InitClusterVersion {
			violationClusterVersions[cluster.Parsed.Cluster.Name] = fmt.Sprintf("get: %q; expect %q", cluster.Parsed.Cluster.InitClusterVersion, defaultVersion)
			prompt = prompt + fmt.Sprintf("cluster %q; get: %q; expect %q\n", cluster.Parsed.Cluster.Name, cluster.Parsed.Cluster.InitClusterVersion, defaultVersion)
		}
	}
	return prompt, nil
}

// defaultGKEClusterVersion get the current default master version in a location.
func defaultGKEClusterVersion(locationType, locationValue string) (string, error) {
	// gcloud container get-server-config --zone us-central1-a --format="value(defaultClusterVersion)"
	cmd := exec.Command("gcloud", "container", "get-server-config", locationType, locationValue, "--format", "value(defaultClusterVersion)")

	out, err := cmdOutput(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to query default GKE cluster version from gcloud: %v", err)
	}

	return strings.TrimSuffix(string(out), "\n"), nil
}
