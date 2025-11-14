package discovery

import (
	"context"
)

// ClusterInfo represents discovered cluster information
type ClusterInfo struct {
	// Name is the cluster name from ClusterProfile
	Name string

	// DisplayName is a human-readable cluster name
	DisplayName string

	// Namespace where the ClusterProfile resource exists
	Namespace string

	// KubernetesVersion is the Kubernetes version of the cluster
	KubernetesVersion string

	// Healthy indicates if the cluster is healthy and available
	Healthy bool

	// Labels are the labels from the ClusterProfile
	Labels map[string]string
}

// Discovery is the interface for discovering clusters
type Discovery interface {
	// ListClusters discovers and returns all available clusters
	ListClusters(ctx context.Context) ([]ClusterInfo, error)

	// GetCluster returns information about a specific cluster
	GetCluster(ctx context.Context, name string) (*ClusterInfo, error)
}
