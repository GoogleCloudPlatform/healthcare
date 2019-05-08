package cft

// GKECluster wraps a CFT GKE cluster.
type GKECluster struct {
	GKEClusterProperties `json:"properties"`
}

// GKEClusterProperties represents a partial GKE cluster implementation.
type GKEClusterProperties struct {
	ResourceName        string `json:"name"`
	ClusterLocationType string `json:"clusterLocationType"`
	Region              string `json:"region"`
	Zone                string `json:"zone"`
}

// Init initializes a new GKE cluster with the given project.
func (cluster *GKECluster) Init(proj *Project) error {
	return nil
}

// Name returns the name of this cluster.
func (cluster *GKECluster) Name() string {
	return cluster.ResourceName
}

// TemplatePath returns the name of the template to use for this cluster.
func (cluster *GKECluster) TemplatePath() string {
	return "deploy/cft/templates/gke/gke.py"
}

// getClusterByName get a cluster that has the given cluster name in a project.
// TODO Replace this function with a general implementation.
func getClusterByName(project *Project, clusterName string) *GKECluster {
	for _, res := range project.Resources {
		if len(res.GKEClusterPair.Raw) > 0 {
			if res.GKEClusterPair.Parsed.Name() == clusterName {
				return &res.GKEClusterPair.Parsed
			}
		}
	}
	return nil
}
