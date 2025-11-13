package discovery

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
)

func TestParseClusterProfile(t *testing.T) {
	tests := []struct {
		name            string
		clusterProfile  *unstructured.Unstructured
		expectedName    string
		expectedHealthy bool
		expectError     bool
	}{
		{
			name: "healthy cluster profile",
			clusterProfile: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "multicluster.x-k8s.io/v1alpha1",
					"kind":       "ClusterProfile",
					"metadata": map[string]interface{}{
						"name":      "test-cluster",
						"namespace": "default",
					},
					"spec": map[string]interface{}{
						"displayName": "Test Cluster",
					},
					"status": map[string]interface{}{
						"version": map[string]interface{}{
							"kubernetes": "v1.30.0",
						},
						"conditions": []interface{}{
							map[string]interface{}{
								"type":   "ControlPlaneHealthy",
								"status": "True",
							},
						},
					},
				},
			},
			expectedName:    "test-cluster",
			expectedHealthy: true,
			expectError:     false,
		},
		{
			name: "unhealthy cluster profile",
			clusterProfile: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "multicluster.x-k8s.io/v1alpha1",
					"kind":       "ClusterProfile",
					"metadata": map[string]interface{}{
						"name":      "unhealthy-cluster",
						"namespace": "default",
					},
					"spec": map[string]interface{}{
						"displayName": "Unhealthy Cluster",
					},
					"status": map[string]interface{}{
						"conditions": []interface{}{
							map[string]interface{}{
								"type":   "ControlPlaneHealthy",
								"status": "False",
							},
						},
					},
				},
			},
			expectedName:    "unhealthy-cluster",
			expectedHealthy: false,
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			client := fake.NewSimpleDynamicClient(scheme)
			discovery := NewClusterProfileDiscovery(client, "default")

			cluster, err := discovery.parseClusterProfile(tt.clusterProfile)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if cluster.Name != tt.expectedName {
				t.Errorf("expected name %s, got %s", tt.expectedName, cluster.Name)
			}

			if cluster.Healthy != tt.expectedHealthy {
				t.Errorf("expected healthy %v, got %v", tt.expectedHealthy, cluster.Healthy)
			}
		})
	}
}

func TestIsClusterHealthy(t *testing.T) {
	tests := []struct {
		name       string
		conditions []interface{}
		expected   bool
	}{
		{
			name: "healthy cluster",
			conditions: []interface{}{
				map[string]interface{}{
					"type":   "ControlPlaneHealthy",
					"status": "True",
				},
			},
			expected: true,
		},
		{
			name: "unhealthy cluster",
			conditions: []interface{}{
				map[string]interface{}{
					"type":   "ControlPlaneHealthy",
					"status": "False",
				},
			},
			expected: false,
		},
		{
			name:       "no conditions",
			conditions: []interface{}{},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			client := fake.NewSimpleDynamicClient(scheme)
			discovery := NewClusterProfileDiscovery(client, "default")

			obj := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"status": map[string]interface{}{
						"conditions": tt.conditions,
					},
				},
			}

			result := discovery.isClusterHealthy(obj)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestListClusters(t *testing.T) {
	// Create test ClusterProfiles
	clusterProfile1 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "multicluster.x-k8s.io/v1alpha1",
			"kind":       "ClusterProfile",
			"metadata": map[string]interface{}{
				"name":      "cluster1",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"displayName": "Cluster 1",
			},
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"type":   "ControlPlaneHealthy",
						"status": "True",
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	client := fake.NewSimpleDynamicClient(scheme, clusterProfile1)
	discovery := NewClusterProfileDiscovery(client, "default")

	clusters, err := discovery.ListClusters(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(clusters) != 1 {
		t.Errorf("expected 1 cluster, got %d", len(clusters))
	}

	if clusters[0].Name != "cluster1" {
		t.Errorf("expected cluster name 'cluster1', got '%s'", clusters[0].Name)
	}
}

func TestGetCluster(t *testing.T) {
	tests := []struct {
		name            string
		clusterProfile  *unstructured.Unstructured
		clusterName     string
		expectedName    string
		expectedDisplay string
		expectedHealthy bool
		expectError     bool
	}{
		{
			name: "get existing cluster",
			clusterProfile: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "multicluster.x-k8s.io/v1alpha1",
					"kind":       "ClusterProfile",
					"metadata": map[string]interface{}{
						"name":      "test-cluster",
						"namespace": "default",
					},
					"spec": map[string]interface{}{
						"displayName": "Test Cluster",
					},
					"status": map[string]interface{}{
						"version": map[string]interface{}{
							"kubernetes": "v1.30.0",
						},
						"conditions": []interface{}{
							map[string]interface{}{
								"type":   "ControlPlaneHealthy",
								"status": "True",
							},
						},
					},
				},
			},
			clusterName:     "test-cluster",
			expectedName:    "test-cluster",
			expectedDisplay: "Test Cluster",
			expectedHealthy: true,
			expectError:     false,
		},
		{
			name: "cluster without displayName uses name",
			clusterProfile: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "multicluster.x-k8s.io/v1alpha1",
					"kind":       "ClusterProfile",
					"metadata": map[string]interface{}{
						"name":      "prod-cluster",
						"namespace": "default",
					},
					"spec": map[string]interface{}{},
					"status": map[string]interface{}{
						"conditions": []interface{}{
							map[string]interface{}{
								"type":   "ControlPlaneHealthy",
								"status": "False",
							},
						},
					},
				},
			},
			clusterName:     "prod-cluster",
			expectedName:    "prod-cluster",
			expectedDisplay: "prod-cluster",
			expectedHealthy: false,
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			client := fake.NewSimpleDynamicClient(scheme, tt.clusterProfile)
			discovery := NewClusterProfileDiscovery(client, "default")

			cluster, err := discovery.GetCluster(context.Background(), tt.clusterName)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.expectError {
				return
			}

			if cluster.Name != tt.expectedName {
				t.Errorf("expected name %s, got %s", tt.expectedName, cluster.Name)
			}

			if cluster.DisplayName != tt.expectedDisplay {
				t.Errorf("expected display name %s, got %s", tt.expectedDisplay, cluster.DisplayName)
			}

			if cluster.Healthy != tt.expectedHealthy {
				t.Errorf("expected healthy %v, got %v", tt.expectedHealthy, cluster.Healthy)
			}
		})
	}
}

func TestGetCluster_NotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	client := fake.NewSimpleDynamicClient(scheme)
	discovery := NewClusterProfileDiscovery(client, "default")

	// Try to get a cluster that doesn't exist
	_, err := discovery.GetCluster(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent cluster, got nil")
	}
}

func TestListClusters_Empty(t *testing.T) {
	// Create an empty unstructured list for the fake client to return
	emptyList := &unstructured.UnstructuredList{}
	emptyList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "multicluster.x-k8s.io",
		Version: "v1alpha1",
		Kind:    "ClusterProfileList",
	})

	scheme := runtime.NewScheme()
	client := fake.NewSimpleDynamicClientWithCustomListKinds(scheme,
		map[schema.GroupVersionResource]string{
			{Group: "multicluster.x-k8s.io", Version: "v1alpha1", Resource: "clusterprofiles"}: "ClusterProfileList",
		})
	discovery := NewClusterProfileDiscovery(client, "default")

	clusters, err := discovery.ListClusters(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(clusters) != 0 {
		t.Errorf("expected 0 clusters, got %d", len(clusters))
	}
}

func TestListClusters_Multiple(t *testing.T) {
	clusterProfile1 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "multicluster.x-k8s.io/v1alpha1",
			"kind":       "ClusterProfile",
			"metadata": map[string]interface{}{
				"name":      "cluster1",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"displayName": "Cluster 1",
			},
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"type":   "ControlPlaneHealthy",
						"status": "True",
					},
				},
			},
		},
	}

	clusterProfile2 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "multicluster.x-k8s.io/v1alpha1",
			"kind":       "ClusterProfile",
			"metadata": map[string]interface{}{
				"name":      "cluster2",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"displayName": "Cluster 2",
			},
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"type":   "ControlPlaneHealthy",
						"status": "True",
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	client := fake.NewSimpleDynamicClient(scheme, clusterProfile1, clusterProfile2)
	discovery := NewClusterProfileDiscovery(client, "default")

	clusters, err := discovery.ListClusters(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(clusters) != 2 {
		t.Errorf("expected 2 clusters, got %d", len(clusters))
	}

	// Verify both clusters are present
	foundCluster1 := false
	foundCluster2 := false
	for _, cluster := range clusters {
		if cluster.Name == "cluster1" {
			foundCluster1 = true
		}
		if cluster.Name == "cluster2" {
			foundCluster2 = true
		}
	}

	if !foundCluster1 {
		t.Error("cluster1 not found in results")
	}
	if !foundCluster2 {
		t.Error("cluster2 not found in results")
	}
}
