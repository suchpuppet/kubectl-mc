package discovery

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// ClusterProfileDiscovery implements Discovery using sig-multicluster ClusterProfile API
type ClusterProfileDiscovery struct {
	client    dynamic.Interface
	namespace string
}

var (
	// clusterProfileGVR is the GroupVersionResource for ClusterProfile
	clusterProfileGVR = schema.GroupVersionResource{
		Group:    "multicluster.x-k8s.io",
		Version:  "v1alpha1",
		Resource: "clusterprofiles",
	}
)

// NewClusterProfileDiscovery creates a new ClusterProfile-based discovery client
func NewClusterProfileDiscovery(client dynamic.Interface, namespace string) *ClusterProfileDiscovery {
	return &ClusterProfileDiscovery{
		client:    client,
		namespace: namespace,
	}
}

// ListClusters discovers all clusters via ClusterProfile API
func (d *ClusterProfileDiscovery) ListClusters(ctx context.Context) ([]ClusterInfo, error) {
	// List all ClusterProfile resources in the specified namespace
	list, err := d.client.Resource(clusterProfileGVR).Namespace(d.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list ClusterProfiles: %w", err)
	}

	clusters := make([]ClusterInfo, 0, len(list.Items))
	for _, item := range list.Items {
		cluster, err := d.parseClusterProfile(&item)
		if err != nil {
			// Log warning but continue with other clusters
			// TODO: Add proper logging
			continue
		}
		clusters = append(clusters, *cluster)
	}

	return clusters, nil
}

// GetCluster returns information about a specific cluster
func (d *ClusterProfileDiscovery) GetCluster(ctx context.Context, name string) (*ClusterInfo, error) {
	item, err := d.client.Resource(clusterProfileGVR).Namespace(d.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get ClusterProfile %s: %w", name, err)
	}

	return d.parseClusterProfile(item)
}

// parseClusterProfile extracts ClusterInfo from an unstructured ClusterProfile resource
func (d *ClusterProfileDiscovery) parseClusterProfile(obj *unstructured.Unstructured) (*ClusterInfo, error) {
	cluster := &ClusterInfo{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
		Labels:    obj.GetLabels(),
	}

	// Extract display name from spec
	if displayName, found, err := unstructured.NestedString(obj.Object, "spec", "displayName"); err == nil && found {
		cluster.DisplayName = displayName
	} else {
		// Fallback to name if displayName not found
		cluster.DisplayName = cluster.Name
	}

	// Extract Kubernetes version from status
	if version, found, err := unstructured.NestedString(obj.Object, "status", "version", "kubernetes"); err == nil && found {
		cluster.KubernetesVersion = version
	}

	// Determine health from conditions
	cluster.Healthy = d.isClusterHealthy(obj)

	return cluster, nil
}

// isClusterHealthy checks the ClusterProfile conditions to determine health
func (d *ClusterProfileDiscovery) isClusterHealthy(obj *unstructured.Unstructured) bool {
	conditions, found, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if err != nil || !found {
		return false
	}

	// Check for ControlPlaneHealthy condition
	for _, cond := range conditions {
		condMap, ok := cond.(map[string]interface{})
		if !ok {
			continue
		}

		condType, _, _ := unstructured.NestedString(condMap, "type")
		status, _, _ := unstructured.NestedString(condMap, "status")

		if condType == "ControlPlaneHealthy" && status == "True" {
			return true
		}
	}

	return false
}
