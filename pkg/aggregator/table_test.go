package aggregator

import (
	"bytes"
	"strings"
	"testing"

	"github.com/suchpuppet/kubectl-mc/pkg/executor"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestNewTableAggregator(t *testing.T) {
	buf := &bytes.Buffer{}
	agg := NewTableAggregator(buf)

	if agg == nil {
		t.Fatal("expected aggregator, got nil")
	}

	if agg.writer != buf {
		t.Error("writer not set correctly")
	}
}

func TestAggregateGetResults_EmptyResults(t *testing.T) {
	buf := &bytes.Buffer{}
	agg := NewTableAggregator(buf)

	results := &executor.AggregatedResults{
		Results: []executor.ClusterResult{},
	}

	err := agg.AggregateGetResults(results, "pods")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No resources found") {
		t.Errorf("expected 'No resources found', got: %s", output)
	}
}

func TestAggregateGetResults_Pods(t *testing.T) {
	buf := &bytes.Buffer{}
	agg := NewTableAggregator(buf)

	pod1 := unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "nginx-1",
				"namespace": "default",
			},
			"status": map[string]interface{}{
				"phase": "Running",
				"containerStatuses": []interface{}{
					map[string]interface{}{
						"ready":        true,
						"restartCount": int64(0),
					},
				},
			},
		},
	}

	results := &executor.AggregatedResults{
		Results: []executor.ClusterResult{
			{
				ClusterName: "cluster1",
				Success:     true,
				Items:       []unstructured.Unstructured{pod1},
			},
		},
	}

	err := agg.AggregateGetResults(results, "pods")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Check for header
	if !strings.Contains(output, "NAMESPACE") {
		t.Error("missing NAMESPACE header")
	}
	if !strings.Contains(output, "NAME") {
		t.Error("missing NAME header")
	}
	if !strings.Contains(output, "CLUSTER") {
		t.Error("missing CLUSTER header")
	}
	if !strings.Contains(output, "READY") {
		t.Error("missing READY header")
	}
	if !strings.Contains(output, "STATUS") {
		t.Error("missing STATUS header")
	}

	// Check for data
	if !strings.Contains(output, "nginx-1") {
		t.Error("missing pod name in output")
	}
	if !strings.Contains(output, "default") {
		t.Error("missing namespace in output")
	}
	if !strings.Contains(output, "cluster1") {
		t.Error("missing cluster name in output")
	}
	if !strings.Contains(output, "Running") {
		t.Error("missing status in output")
	}
}

func TestAggregateGetResults_Deployments(t *testing.T) {
	buf := &bytes.Buffer{}
	agg := NewTableAggregator(buf)

	deployment := unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "nginx-deployment",
				"namespace": "default",
			},
			"status": map[string]interface{}{
				"replicas":          int64(3),
				"readyReplicas":     int64(3),
				"updatedReplicas":   int64(3),
				"availableReplicas": int64(3),
			},
		},
	}

	results := &executor.AggregatedResults{
		Results: []executor.ClusterResult{
			{
				ClusterName: "cluster1",
				Success:     true,
				Items:       []unstructured.Unstructured{deployment},
			},
		},
	}

	err := agg.AggregateGetResults(results, "deployments")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "nginx-deployment") {
		t.Error("missing deployment name in output")
	}
	if !strings.Contains(output, "3/3") {
		t.Error("missing ready replicas in output")
	}
	if !strings.Contains(output, "READY") {
		t.Error("missing READY header")
	}
}

func TestAggregateGetResults_Services(t *testing.T) {
	buf := &bytes.Buffer{}
	agg := NewTableAggregator(buf)

	service := unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "nginx-service",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"type":      "ClusterIP",
				"clusterIP": "10.96.0.1",
				"ports": []interface{}{
					map[string]interface{}{
						"port":     int64(80),
						"protocol": "TCP",
					},
				},
			},
		},
	}

	results := &executor.AggregatedResults{
		Results: []executor.ClusterResult{
			{
				ClusterName: "cluster1",
				Success:     true,
				Items:       []unstructured.Unstructured{service},
			},
		},
	}

	err := agg.AggregateGetResults(results, "services")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "nginx-service") {
		t.Error("missing service name in output")
	}
	if !strings.Contains(output, "ClusterIP") {
		t.Error("missing service type in output")
	}
	if !strings.Contains(output, "10.96.0.1") {
		t.Error("missing cluster IP in output")
	}
}

func TestAggregateGetResults_Generic(t *testing.T) {
	buf := &bytes.Buffer{}
	agg := NewTableAggregator(buf)

	resource := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "my-config",
				"namespace": "default",
			},
		},
	}
	resource.SetKind("ConfigMap")

	results := &executor.AggregatedResults{
		Results: []executor.ClusterResult{
			{
				ClusterName: "cluster1",
				Success:     true,
				Items:       []unstructured.Unstructured{resource},
			},
		},
	}

	err := agg.AggregateGetResults(results, "configmaps")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "my-config") {
		t.Error("missing resource name in output")
	}
	if !strings.Contains(output, "ConfigMap") {
		t.Error("missing resource kind in output")
	}
}

func TestAggregateGetResults_MultipleClustersSorting(t *testing.T) {
	buf := &bytes.Buffer{}
	agg := NewTableAggregator(buf)

	pod1 := unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "pod-a",
				"namespace": "default",
			},
			"status": map[string]interface{}{
				"phase": "Running",
			},
		},
	}

	pod2 := unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "pod-b",
				"namespace": "default",
			},
			"status": map[string]interface{}{
				"phase": "Running",
			},
		},
	}

	results := &executor.AggregatedResults{
		Results: []executor.ClusterResult{
			{
				ClusterName: "cluster2",
				Success:     true,
				Items:       []unstructured.Unstructured{pod2},
			},
			{
				ClusterName: "cluster1",
				Success:     true,
				Items:       []unstructured.Unstructured{pod1},
			},
		},
	}

	err := agg.AggregateGetResults(results, "pods")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	lines := strings.Split(output, "\n")

	// Should be sorted by cluster name (cluster1 before cluster2)
	cluster1Line := -1
	cluster2Line := -1

	for i, line := range lines {
		if strings.Contains(line, "cluster1") {
			cluster1Line = i
		}
		if strings.Contains(line, "cluster2") {
			cluster2Line = i
		}
	}

	if cluster1Line == -1 || cluster2Line == -1 {
		t.Error("missing cluster entries in output")
	}

	if cluster1Line > cluster2Line {
		t.Error("results not sorted by cluster name")
	}
}

func TestAggregateGetResults_FailedCluster(t *testing.T) {
	buf := &bytes.Buffer{}
	agg := NewTableAggregator(buf)

	pod := unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "nginx-1",
				"namespace": "default",
			},
			"status": map[string]interface{}{
				"phase": "Running",
			},
		},
	}

	results := &executor.AggregatedResults{
		Results: []executor.ClusterResult{
			{
				ClusterName: "cluster1",
				Success:     true,
				Items:       []unstructured.Unstructured{pod},
			},
			{
				ClusterName: "cluster2",
				Success:     false,
			},
		},
	}

	err := agg.AggregateGetResults(results, "pods")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Should only show results from successful cluster
	if !strings.Contains(output, "cluster1") {
		t.Error("missing successful cluster in output")
	}
	if strings.Contains(output, "cluster2") {
		t.Error("failed cluster should not appear in output")
	}
	if !strings.Contains(output, "nginx-1") {
		t.Error("missing pod from successful cluster")
	}
}
