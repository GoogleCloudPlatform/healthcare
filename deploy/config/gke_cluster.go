package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
)

const (
	clusterRegionFindReString = `^\w+-\w+`
	clusterZoneMatchReString  = clusterRegionFindReString + `-\w+$`
)

var reClusterRegionFind = regexp.MustCompile(clusterRegionFindReString)
var reClusterZoneMatch = regexp.MustCompile(clusterZoneMatchReString)

// GKECluster wraps a CFT GKE cluster.
type GKECluster struct {
	GKEClusterProperties `json:"properties"`
	raw                  json.RawMessage
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
	Name             string         `json:"name"`
	InitialNodeCount int            `json:"initialNodeCount,omitempty"`
	NodeConfig       NodeConfigInfo `json:"nodeConfig,omitempty"`
}

// NodeConfigInfo represents a partial node config implementation.
type NodeConfigInfo struct {
	Accelerators []*Accelerator `json:"accelerators,omitempty"`
}

// Accelerator represents a partial accelerator implementation.
type Accelerator struct {
	AcceleratorCount string `json:"acceleratorCount,omitempty"`
	AcceleratorType  string `json:"acceleratorType,omitempty"`
}

// RegionString get the region of the cluster, even if zone is specified.
func (c *GKECluster) RegionString() (string, error) {
	if c.Region != "" {
		return c.Region, nil
	}
	if c.Zone != "" {
		matched := reClusterZoneMatch.Match([]byte(c.Zone))
		if matched != true {
			return c.Zone, errors.New("cannot identify cluster zone format")
		}
		region := reClusterRegionFind.Find([]byte(c.Zone))
		return string(region), nil
	}
	return "", errors.New("cannot get region nor zone")
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

// aliasGKECluster is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
type aliasGKECluster GKECluster

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (c *GKECluster) UnmarshalJSON(data []byte) error {
	var alias aliasGKECluster
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return fmt.Errorf("failed to unmarshal to parsed alias: %v", err)
	}
	*c = GKECluster(alias)
	if c.Cluster.NodeConfig.Accelerators == nil {
		c.Cluster.NodeConfig.Accelerators = []*Accelerator{}
	}
	return nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (c *GKECluster) MarshalJSON() ([]byte, error) {
	return interfacePair{c.raw, aliasGKECluster(*c)}.MarshalJSON()
}
