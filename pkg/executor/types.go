package executor

import (
	"github.com/suchpuppet/kubectl-mc/pkg/discovery"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ClusterResult represents the result from a single cluster
type ClusterResult struct {
	ClusterName string
	Success     bool
	Items       []unstructured.Unstructured
	Error       error
}

// AggregatedResults contains results from all clusters
type AggregatedResults struct {
	Results []ClusterResult
	Summary ResultSummary
}

// ResultSummary provides statistics about the multi-cluster operation
type ResultSummary struct {
	Total      int
	Successful int
	Failed     int
	Errors     map[string]error // cluster name -> error
}

// ExecutorConfig configures the executor behavior
type ExecutorConfig struct {
	MaxConcurrency  int  // Maximum number of concurrent cluster queries
	TimeoutSeconds  int  // Timeout for each cluster operation
	ContinueOnError bool // Continue if some clusters fail
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() ExecutorConfig {
	return ExecutorConfig{
		MaxConcurrency:  10,
		TimeoutSeconds:  30,
		ContinueOnError: true,
	}
}

// NewAggregatedResults creates an initialized AggregatedResults
func NewAggregatedResults(clusters []discovery.ClusterInfo) *AggregatedResults {
	return &AggregatedResults{
		Results: make([]ClusterResult, 0, len(clusters)),
		Summary: ResultSummary{
			Total:  len(clusters),
			Errors: make(map[string]error),
		},
	}
}

// AddResult adds a cluster result and updates the summary
func (ar *AggregatedResults) AddResult(result ClusterResult) {
	ar.Results = append(ar.Results, result)

	if result.Success {
		ar.Summary.Successful++
	} else {
		ar.Summary.Failed++
		if result.Error != nil {
			ar.Summary.Errors[result.ClusterName] = result.Error
		}
	}
}
