package aggregator

import (
	"bytes"
	"strings"
	"testing"

	"github.com/suchpuppet/kubectl-mc/pkg/discovery"
	"github.com/suchpuppet/kubectl-mc/pkg/executor"
)

const (
	testNginxOutput    = "Name:         nginx\nNamespace:    default\nLabels:       app=nginx\n"
	testUnexpectedErr  = "unexpected error: %v"
)

func TestDescribeAggregator(t *testing.T) {
	tests := []struct {
		name     string
		results  *executor.AggregatedResults
		wantText []string // Strings that should appear in output
	}{
		{
			name: "single cluster with output",
			results: &executor.AggregatedResults{
				Results: []executor.ClusterResult{
					{
						ClusterName: "cluster1",
						Success:     true,
						Output:      testNginxOutput,
					},
				},
			},
			wantText: []string{
				"CLUSTER: cluster1",
				"Name:         nginx",
				"Namespace:    default",
				"Labels:       app=nginx",
			},
		},
		{
			name: "multiple clusters with output",
			results: &executor.AggregatedResults{
				Results: []executor.ClusterResult{
					{
						ClusterName: "cluster1",
						Success:     true,
						Output:      "Name:         nginx\nNamespace:    default\n",
					},
					{
						ClusterName: "cluster2",
						Success:     true,
						Output:      "Name:         nginx\nNamespace:    production\n",
					},
				},
			},
			wantText: []string{
				"CLUSTER: cluster1",
				"Name:         nginx",
				"Namespace:    default",
				"CLUSTER: cluster2",
				"Namespace:    production",
				"========", // Separator between clusters
			},
		},
		{
			name: "cluster with error",
			results: &executor.AggregatedResults{
				Results: []executor.ClusterResult{
					{
						ClusterName: "cluster1",
						Success:     false,
						Error:       nil,
					},
					{
						ClusterName: "cluster2",
						Success:     true,
						Output:      "Name:         nginx\n",
					},
				},
			},
			wantText: []string{
				"CLUSTER: cluster2",
				"Name:         nginx",
			},
		},
		{
			name: "no results",
			results: &executor.AggregatedResults{
				Results: []executor.ClusterResult{},
				Summary: executor.ResultSummary{
					Total:  0,
					Failed: 0,
					Errors: make(map[string]error),
				},
			},
			wantText: []string{
				"No resources found",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			agg := NewDescribeAggregator(&buf)

			err := agg.AggregateDescribeResults(tt.results, "pod")
			if err != nil {
				t.Fatalf(testUnexpectedErr, err)
			}

			output := buf.String()
			for _, want := range tt.wantText {
				if !strings.Contains(output, want) {
					t.Errorf("output missing expected text %q\nGot:\n%s", want, output)
				}
			}
		})
	}
}

func TestDescribeAggregator_SortsByClusterName(t *testing.T) {
	results := &executor.AggregatedResults{
		Results: []executor.ClusterResult{
			{ClusterName: "zebra", Success: true, Output: "Resource from zebra\n"},
			{ClusterName: "alpha", Success: true, Output: "Resource from alpha\n"},
			{ClusterName: "beta", Success: true, Output: "Resource from beta\n"},
		},
	}

	var buf bytes.Buffer
	agg := NewDescribeAggregator(&buf)
	err := agg.AggregateDescribeResults(results, "pod")
	if err != nil {
		t.Fatalf(testUnexpectedErr, err)
	}

	output := buf.String()

	// Check that clusters appear in alphabetical order
	alphaIdx := strings.Index(output, "CLUSTER: alpha")
	betaIdx := strings.Index(output, "CLUSTER: beta")
	zebraIdx := strings.Index(output, "CLUSTER: zebra")

	if alphaIdx == -1 || betaIdx == -1 || zebraIdx == -1 {
		t.Fatalf("missing cluster headers in output:\n%s", output)
	}

	if !(alphaIdx < betaIdx && betaIdx < zebraIdx) {
		t.Errorf("clusters not in alphabetical order. alpha=%d, beta=%d, zebra=%d", alphaIdx, betaIdx, zebraIdx)
	}
}

func TestNewDescribeAggregator(t *testing.T) {
	var buf bytes.Buffer
	agg := NewDescribeAggregator(&buf)

	if agg == nil {
		t.Fatal("expected non-nil aggregator")
	}

	if agg.writer != &buf {
		t.Error("writer not set correctly")
	}
}

// Test with real-world cluster info structure
func TestDescribeAggregator_WithClusterInfo(t *testing.T) {
	clusters := []discovery.ClusterInfo{
		{Name: "prod-cluster"},
		{Name: "staging-cluster"},
	}

	results := executor.NewAggregatedResults(clusters)
	results.AddResult(executor.ClusterResult{
		ClusterName: "prod-cluster",
		Success:     true,
		Output:      "Name:         production-app\nNamespace:    default\n",
	})
	results.AddResult(executor.ClusterResult{
		ClusterName: "staging-cluster",
		Success:     true,
		Output:      "Name:         staging-app\nNamespace:    staging\n",
	})

	var buf bytes.Buffer
	agg := NewDescribeAggregator(&buf)
	err := agg.AggregateDescribeResults(results, "deployment")
	if err != nil {
		t.Fatalf(testUnexpectedErr, err)
	}

	output := buf.String()

	expectedStrings := []string{
		"CLUSTER: prod-cluster",
		"production-app",
		"CLUSTER: staging-cluster",
		"staging-app",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("output missing expected text %q\nGot:\n%s", expected, output)
		}
	}
}
