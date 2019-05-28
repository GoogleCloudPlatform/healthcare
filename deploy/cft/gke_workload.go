package cft

// GKEWorkload represents a GKE resources, not limited to workloads.
type GKEWorkload struct {
	Properties  interface{} `json:"properties"`
	ClusterName string      `json:"cluster_name"`
}
