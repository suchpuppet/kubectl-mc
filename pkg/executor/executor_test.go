package executor

import (
	"context"
	"testing"

	"github.com/suchpuppet/kubectl-mc/pkg/discovery"
	"github.com/suchpuppet/kubectl-mc/pkg/kubeconfig"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func TestNewExecutor(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	manager, err := kubeconfig.NewManager("")
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	executor := NewExecutor(manager, configFlags)

	if executor == nil {
		t.Fatal("expected executor, got nil")
	}

	if executor.mappingManager != manager {
		t.Error("mapping manager not set correctly")
	}

	if executor.configFlags != configFlags {
		t.Error("config flags not set correctly")
	}

	// Verify default config is set
	if executor.config.MaxConcurrency != 10 {
		t.Errorf("expected MaxConcurrency 10, got %d", executor.config.MaxConcurrency)
	}

	if executor.config.TimeoutSeconds != 30 {
		t.Errorf("expected TimeoutSeconds 30, got %d", executor.config.TimeoutSeconds)
	}

	if !executor.config.ContinueOnError {
		t.Error("expected ContinueOnError to be true")
	}
}

func TestResolveGVR(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	manager, _ := kubeconfig.NewManager("")
	executor := NewExecutor(manager, configFlags)

	tests := []struct {
		name        string
		resource    string
		expectedGVR schema.GroupVersionResource
		expectError bool
	}{
		{
			name:     "pods",
			resource: "pods",
			expectedGVR: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "pods",
			},
			expectError: false,
		},
		{
			name:     "pod singular",
			resource: "pod",
			expectedGVR: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "pods",
			},
			expectError: false,
		},
		{
			name:     "deployments",
			resource: "deployments",
			expectedGVR: schema.GroupVersionResource{
				Group:    "apps",
				Version:  "v1",
				Resource: "deployments",
			},
			expectError: false,
		},
		{
			name:     "deployment singular",
			resource: "deployment",
			expectedGVR: schema.GroupVersionResource{
				Group:    "apps",
				Version:  "v1",
				Resource: "deployments",
			},
			expectError: false,
		},
		{
			name:     "services",
			resource: "services",
			expectedGVR: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "services",
			},
			expectError: false,
		},
		{
			name:     "service singular",
			resource: "service",
			expectedGVR: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "services",
			},
			expectError: false,
		},
		{
			name:        "unknown resource",
			resource:    "unknownresource",
			expectError: true,
		},
		{
			name:        "empty resource",
			resource:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gvr, err := executor.resolveGVR(nil, tt.resource)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if gvr.Group != tt.expectedGVR.Group {
				t.Errorf("expected group %s, got %s", tt.expectedGVR.Group, gvr.Group)
			}

			if gvr.Version != tt.expectedGVR.Version {
				t.Errorf("expected version %s, got %s", tt.expectedGVR.Version, gvr.Version)
			}

			if gvr.Resource != tt.expectedGVR.Resource {
				t.Errorf("expected resource %s, got %s", tt.expectedGVR.Resource, gvr.Resource)
			}
		})
	}
}

func TestExecutorGet_EmptyClusters(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	manager, _ := kubeconfig.NewManager("")
	executor := NewExecutor(manager, configFlags)

	ctx := context.Background()
	clusters := []discovery.ClusterInfo{}

	results, err := executor.Get(ctx, clusters, "pods", "", "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results == nil {
		t.Fatal("expected results, got nil")
	}

	if len(results.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results.Results))
	}

	if results.Summary.Total != 0 {
		t.Errorf("expected Total 0, got %d", results.Summary.Total)
	}
}

func TestExecutorGet_SingleCluster_NoMapping(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	manager, _ := kubeconfig.NewManager("")
	executor := NewExecutor(manager, configFlags)

	ctx := context.Background()
	clusters := []discovery.ClusterInfo{
		{
			Name:      "test-cluster",
			Namespace: "default",
		},
	}

	// This should fail because there's no mapping for "test-cluster"
	results, err := executor.Get(ctx, clusters, "pods", "", "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results == nil {
		t.Fatal("expected results, got nil")
	}

	// Should have 1 result (failed)
	if len(results.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results.Results))
	}

	if results.Results[0].Success {
		t.Error("expected failure due to missing mapping")
	}

	if results.Summary.Failed != 1 {
		t.Errorf("expected 1 failed cluster, got %d", results.Summary.Failed)
	}
}

func TestExecutorGet_MultipleClusters_NoMappings(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	manager, _ := kubeconfig.NewManager("")
	executor := NewExecutor(manager, configFlags)

	ctx := context.Background()
	clusters := []discovery.ClusterInfo{
		{Name: "cluster1", Namespace: "ns1"},
		{Name: "cluster2", Namespace: "ns2"},
		{Name: "cluster3", Namespace: "ns3"},
	}

	results, err := executor.Get(ctx, clusters, "pods", "", "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results == nil {
		t.Fatal("expected results, got nil")
	}

	// Should have 3 results (all failed)
	if len(results.Results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results.Results))
	}

	// All should fail
	if results.Summary.Successful != 0 {
		t.Errorf("expected 0 successful, got %d", results.Summary.Successful)
	}

	if results.Summary.Failed != 3 {
		t.Errorf("expected 3 failed, got %d", results.Summary.Failed)
	}

	// Verify all have errors
	if len(results.Summary.Errors) != 3 {
		t.Errorf("expected 3 errors, got %d", len(results.Summary.Errors))
	}
}

func TestExecutorGet_ContextCancellation(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	manager, _ := kubeconfig.NewManager("")
	executor := NewExecutor(manager, configFlags)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	clusters := []discovery.ClusterInfo{
		{Name: "cluster1", Namespace: "ns1"},
	}

	results, err := executor.Get(ctx, clusters, "pods", "", "default")

	// Should not error even with cancelled context
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results == nil {
		t.Fatal("expected results, got nil")
	}

	// Result should indicate failure
	if len(results.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results.Results))
	}
}

func TestExecutorGet_DifferentResourceTypes(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	manager, _ := kubeconfig.NewManager("")
	executor := NewExecutor(manager, configFlags)

	ctx := context.Background()
	clusters := []discovery.ClusterInfo{
		{Name: "test-cluster", Namespace: "default"},
	}

	resourceTypes := []string{"pods", "deployments", "services"}

	for _, resource := range resourceTypes {
		t.Run(resource, func(t *testing.T) {
			results, err := executor.Get(ctx, clusters, resource, "", "default")
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", resource, err)
			}

			if results == nil {
				t.Fatalf("expected results for %s, got nil", resource)
			}

			// Should have 1 result (will fail due to no mapping, but shouldn't panic)
			if len(results.Results) != 1 {
				t.Errorf("expected 1 result for %s, got %d", resource, len(results.Results))
			}
		})
	}
}

func TestExecutorGet_WithSpecificName(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	manager, _ := kubeconfig.NewManager("")
	executor := NewExecutor(manager, configFlags)

	ctx := context.Background()
	clusters := []discovery.ClusterInfo{
		{Name: "test-cluster", Namespace: "default"},
	}

	// Test with specific pod name
	results, err := executor.Get(ctx, clusters, "pods", "nginx-pod", "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results == nil {
		t.Fatal("expected results, got nil")
	}

	// Should have 1 result (will fail due to no mapping)
	if len(results.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results.Results))
	}
}

func TestExecutorGet_AllNamespaces(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	manager, _ := kubeconfig.NewManager("")
	executor := NewExecutor(manager, configFlags)

	ctx := context.Background()
	clusters := []discovery.ClusterInfo{
		{Name: "test-cluster", Namespace: "default"},
	}

	// Test with empty namespace (all namespaces)
	results, err := executor.Get(ctx, clusters, "pods", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results == nil {
		t.Fatal("expected results, got nil")
	}

	// Should have 1 result
	if len(results.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results.Results))
	}
}

func TestExecutorConfigDefaults(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	manager, _ := kubeconfig.NewManager("")
	executor := NewExecutor(manager, configFlags)

	// Verify executor uses default config
	if executor.config.MaxConcurrency != DefaultConfig().MaxConcurrency {
		t.Errorf("expected MaxConcurrency %d, got %d",
			DefaultConfig().MaxConcurrency, executor.config.MaxConcurrency)
	}

	if executor.config.TimeoutSeconds != DefaultConfig().TimeoutSeconds {
		t.Errorf("expected TimeoutSeconds %d, got %d",
			DefaultConfig().TimeoutSeconds, executor.config.TimeoutSeconds)
	}

	if executor.config.ContinueOnError != DefaultConfig().ContinueOnError {
		t.Errorf("expected ContinueOnError %v, got %v",
			DefaultConfig().ContinueOnError, executor.config.ContinueOnError)
	}
}
