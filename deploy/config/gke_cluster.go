package config

// GKECluster wraps a CFT GKE cluster.
type GKECluster struct {
	GKEClusterProperties `json:"properties"`
}

// GKEClusterProperties represents a partial GKE cluster implementation.
type GKEClusterProperties struct {
	ClusterLocationType string             `json:"clusterLocationType"`
	Region              string             `json:"region"`
	Zone                string             `json:"zone"`
	Cluster             GKEClusterSettings `json:"cluster"`
}

// GKEClusterSettings the cluster settings in a GKE cluster.
type GKEClusterSettings struct {
	Name string `json:"name"`
}

// Init initializes a new GKE cluster with the given project.
func (*GKECluster) Init(proj *Project) error {
	return nil
}

// Name returns the name of this cluster.
func (c *GKECluster) Name() string {
	return c.Cluster.Name
}

// TemplatePath returns the name of the template to use for this cluster.
func (*GKECluster) TemplatePath() string {
	return "deploy/config/templates/gke/gke.py"
}
