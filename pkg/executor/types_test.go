package executor

import (
	"fmt"
	"testing"

	"github.com/suchpuppet/kubectl-mc/pkg/discovery"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.MaxConcurrency != 10 {
		t.Errorf("expected MaxConcurrency 10, got %d", config.MaxConcurrency)
	}

	if config.TimeoutSeconds != 30 {
		t.Errorf("expected TimeoutSeconds 30, got %d", config.TimeoutSeconds)
	}

	if !config.ContinueOnError {
		t.Error("expected ContinueOnError to be true")
	}
}

func TestNewAggregatedResults(t *testing.T) {
	clusters := []discovery.ClusterInfo{
		{Name: "cluster1"},
		{Name: "cluster2"},
		{Name: "cluster3"},
	}

	results := NewAggregatedResults(clusters)

	if results == nil {
		t.Fatal("expected results, got nil")
	}

	if results.Summary.Total != 3 {
		t.Errorf("expected Total 3, got %d", results.Summary.Total)
	}

	if results.Summary.Successful != 0 {
		t.Errorf("expected Successful 0, got %d", results.Summary.Successful)
	}

	if results.Summary.Failed != 0 {
		t.Errorf("expected Failed 0, got %d", results.Summary.Failed)
	}

	if len(results.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results.Results))
	}

	if results.Summary.Errors == nil {
		t.Error("expected Errors map to be initialized")
	}
}

func TestAddResult_Success(t *testing.T) {
	clusters := []discovery.ClusterInfo{{Name: "cluster1"}}
	results := NewAggregatedResults(clusters)

	item := unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name": "test-pod",
			},
		},
	}

	result := ClusterResult{
		ClusterName: "cluster1",
		Success:     true,
		Items:       []unstructured.Unstructured{item},
		Error:       nil,
	}

	results.AddResult(result)

	if len(results.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results.Results))
	}

	if results.Summary.Successful != 1 {
		t.Errorf("expected Successful 1, got %d", results.Summary.Successful)
	}

	if results.Summary.Failed != 0 {
		t.Errorf("expected Failed 0, got %d", results.Summary.Failed)
	}

	if len(results.Summary.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(results.Summary.Errors))
	}
}

func TestAddResult_Failure(t *testing.T) {
	clusters := []discovery.ClusterInfo{{Name: "cluster1"}}
	results := NewAggregatedResults(clusters)

	result := ClusterResult{
		ClusterName: "cluster1",
		Success:     false,
		Items:       nil,
		Error:       fmt.Errorf("connection timeout"),
	}

	results.AddResult(result)

	if len(results.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results.Results))
	}

	if results.Summary.Successful != 0 {
		t.Errorf("expected Successful 0, got %d", results.Summary.Successful)
	}

	if results.Summary.Failed != 1 {
		t.Errorf("expected Failed 1, got %d", results.Summary.Failed)
	}

	if len(results.Summary.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(results.Summary.Errors))
	}

	if _, exists := results.Summary.Errors["cluster1"]; !exists {
		t.Error("expected error for cluster1 in Errors map")
	}
}

func TestAddResult_Mixed(t *testing.T) {
	clusters := []discovery.ClusterInfo{
		{Name: "cluster1"},
		{Name: "cluster2"},
		{Name: "cluster3"},
	}
	results := NewAggregatedResults(clusters)

	// Add successful result
	results.AddResult(ClusterResult{
		ClusterName: "cluster1",
		Success:     true,
		Items:       []unstructured.Unstructured{{}},
	})

	// Add failed result
	results.AddResult(ClusterResult{
		ClusterName: "cluster2",
		Success:     false,
		Error:       fmt.Errorf("error"),
	})

	// Add another successful result
	results.AddResult(ClusterResult{
		ClusterName: "cluster3",
		Success:     true,
		Items:       []unstructured.Unstructured{{}, {}},
	})

	if len(results.Results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results.Results))
	}

	if results.Summary.Successful != 2 {
		t.Errorf("expected Successful 2, got %d", results.Summary.Successful)
	}

	if results.Summary.Failed != 1 {
		t.Errorf("expected Failed 1, got %d", results.Summary.Failed)
	}

	if len(results.Summary.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(results.Summary.Errors))
	}
}

func TestClusterResult_Basic(t *testing.T) {
	result := ClusterResult{
		ClusterName: "test-cluster",
		Success:     true,
		Items:       []unstructured.Unstructured{{}, {}},
		Error:       nil,
	}

	if result.ClusterName != "test-cluster" {
		t.Errorf("expected ClusterName 'test-cluster', got '%s'", result.ClusterName)
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}

	if len(result.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Items))
	}

	if result.Error != nil {
		t.Errorf("expected no error, got %v", result.Error)
	}
}

func TestExecutorConfig_CustomValues(t *testing.T) {
	config := ExecutorConfig{
		MaxConcurrency:  5,
		TimeoutSeconds:  60,
		ContinueOnError: false,
	}

	if config.MaxConcurrency != 5 {
		t.Errorf("expected MaxConcurrency 5, got %d", config.MaxConcurrency)
	}

	if config.TimeoutSeconds != 60 {
		t.Errorf("expected TimeoutSeconds 60, got %d", config.TimeoutSeconds)
	}

	if config.ContinueOnError {
		t.Error("expected ContinueOnError to be false")
	}
}
