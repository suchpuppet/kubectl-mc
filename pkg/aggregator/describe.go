package aggregator

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/suchpuppet/kubectl-mc/pkg/executor"
)

// DescribeAggregator formats multi-cluster describe results
type DescribeAggregator struct {
	writer io.Writer
}

// NewDescribeAggregator creates a new describe aggregator
func NewDescribeAggregator(writer io.Writer) *DescribeAggregator {
	return &DescribeAggregator{
		writer: writer,
	}
}

// AggregateDescribeResults aggregates and formats describe results across clusters
// Returns error only if ALL clusters failed. If at least one cluster returns results, it's considered success.
func (a *DescribeAggregator) AggregateDescribeResults(results *executor.AggregatedResults, resourceType string) error {
	// Sort results by cluster name for consistent output
	sortedResults := make([]executor.ClusterResult, len(results.Results))
	copy(sortedResults, results.Results)
	sort.Slice(sortedResults, func(i, j int) bool {
		return sortedResults[i].ClusterName < sortedResults[j].ClusterName
	})

	// Track if we've printed any results
	hasOutput := false

	// Print results from each cluster
	for _, result := range sortedResults {
		// Skip failed results silently - we only care if we get at least one success
		if !result.Success {
			continue
		}

		if result.Output == "" {
			continue
		}

		// Add separator between clusters
		if hasOutput {
			fmt.Fprintln(a.writer, "\n"+strings.Repeat("=", 80))
		}

		// Print cluster header
		fmt.Fprintf(a.writer, "\n")
		fmt.Fprintf(a.writer, "CLUSTER: %s\n", result.ClusterName)
		fmt.Fprintf(a.writer, "%s\n", strings.Repeat("-", 80))

		// Print the describe output for this cluster
		fmt.Fprint(a.writer, result.Output)

		hasOutput = true
	}

	// Only return error if NO cluster had any results AND there were failures
	if !hasOutput {
		if results.Summary.Total > 0 && results.Summary.Failed == results.Summary.Total {
			return fmt.Errorf("failed to describe resource in all %d clusters", results.Summary.Total)
		}
		fmt.Fprintln(a.writer, "No resources found")
	}

	return nil
}
