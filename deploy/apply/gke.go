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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

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

// interfaceToJSONFile writes data to a json file f.
func interfaceToJSONFile(f *os.File, data interface{}) error {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data : %v", err)
	}
	log.Printf("Creating data:\n%v", string(b))

	if _, err := f.Write(b); err != nil {
		return fmt.Errorf("failed to write deployment to file: %v", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %v", err)
	}
	return nil
}

// installClusterWorkload creates and updates (when it exists) not only workloads
// but also all resources supported by "kubectl apply -f". Data comes from a GKEWorkload struct.
func installClusterWorkload(clusterName string, project *config.Project, workload interface{}) error {
	tmp, err := ioutil.TempFile("", "")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())

	if err := interfaceToJSONFile(tmp, workload); err != nil {
		return fmt.Errorf("failed to create write workload file: %v", err)
	}

	return installClusterWorkloadFromFile(clusterName, tmp.Name(), project)
}

func importBinauthz(projectID string, binauthz *config.BinAuthz) error {
	if binauthz == nil {
		return nil
	}
	tmp, err := ioutil.TempFile("", "*.json")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())

	err = interfaceToJSONFile(tmp, binauthz.Properties)
	if err != nil {
		return fmt.Errorf("failed to create write policy file: %v", err)
	}

	return configBinauthzPolicy(tmp.Name(), projectID)
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
		if c.Name() == clusterName {
			return c
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

func configBinauthzPolicy(policyYamlPath, projectID string) error {
	// binaryauthorization.googleapis.com must be enabled
	// https://cloud.google.com/binary-authorization/docs/quickstart
	cmd := exec.Command("gcloud", "beta", "container", "binauthz", "policy", "import", policyYamlPath, "--project", projectID)
	if err := cmdRun(cmd); err != nil {
		return fmt.Errorf("failed to import policy for %q: %v", projectID, err)
	}
	return nil
}
