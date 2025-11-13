package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/suchpuppet/kubectl-mc/pkg/aggregator"
	"github.com/suchpuppet/kubectl-mc/pkg/client"
	"github.com/suchpuppet/kubectl-mc/pkg/discovery"
	"github.com/suchpuppet/kubectl-mc/pkg/executor"
	"github.com/suchpuppet/kubectl-mc/pkg/kubeconfig"
)

var (
	// getCmd represents the get command
	getCmd = &cobra.Command{
		Use:   "get [resource] [name]",
		Short: "Get resources across multiple clusters",
		Long: `Get resources across all discovered clusters and aggregate the results.

Examples:
  # List all pods across all clusters
  kubectl mc get pods

  # List deployments in a specific namespace
  kubectl mc get deployments -n default

  # Get a specific pod
  kubectl mc get pod nginx`,
		Args: cobra.MinimumNArgs(1),
		RunE: runGet,
	}

	// Cluster filtering flags
	clustersFlag []string
	excludeFlag  []string
	allClusters  bool
)

func init() {
	rootCmd.AddCommand(getCmd)

	// Add cluster filtering flags
	getCmd.Flags().StringSliceVar(&clustersFlag, "clusters", []string{}, "comma-separated list of cluster names or patterns")
	getCmd.Flags().StringSliceVar(&excludeFlag, "exclude", []string{}, "comma-separated list of cluster names or patterns to exclude")
	getCmd.Flags().BoolVar(&allClusters, "all-clusters", false, "target all clusters (explicit confirmation)")
}

func runGet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get hub context
	hubContext, err := cmd.Flags().GetString("hub-context")
	if err != nil {
		return fmt.Errorf("failed to get hub-context flag: %w", err)
	}

	hubNamespace, err := cmd.Flags().GetString("hub-namespace")
	if err != nil {
		return fmt.Errorf("failed to get hub-namespace flag: %w", err)
	}

	// Create hub client
	hubClientFactory, err := client.NewFactory(hubContext, kubeConfigFlags)
	if err != nil {
		return fmt.Errorf("failed to create hub client factory: %w", err)
	}

	dynamicClient, err := hubClientFactory.DynamicClient()
	if err != nil {
		return fmt.Errorf("failed to create dynamic client for hub: %w", err)
	}

	// Create discovery client
	discoveryClient := discovery.NewClusterProfileDiscovery(dynamicClient, hubNamespace)

	// Discover clusters
	clusters, err := discoveryClient.ListClusters(ctx)
	if err != nil {
		return fmt.Errorf("failed to discover clusters: %w", err)
	}

	if len(clusters) == 0 {
		fmt.Fprintf(os.Stderr, "No clusters discovered from hub\n")
		return nil
	}

	fmt.Fprintf(os.Stderr, "Discovered %d cluster(s)\n", len(clusters))

	// Load kubeconfig mappings
	mappingManager, err := kubeconfig.NewManager("")
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig mappings: %w", err)
	}

	// Filter clusters based on flags
	filteredClusters := filterClusters(clusters, clustersFlag, excludeFlag)

	// Create executor
	exec := executor.NewExecutor(mappingManager, kubeConfigFlags)

	// Extract resource type and name from args
	resource := args[0]
	var resourceName string
	if len(args) > 1 {
		resourceName = args[1]
	}

	// Get namespace flag
	namespace, err := cmd.Flags().GetString("namespace")
	if err != nil {
		namespace = ""
	}

	// Execute get across all clusters
	results, err := exec.Get(ctx, filteredClusters, resource, resourceName, namespace)
	if err != nil {
		return fmt.Errorf("failed to execute get: %w", err)
	}

	// Aggregate and format results
	agg := aggregator.NewTableAggregator(os.Stdout)
	if err := agg.AggregateGetResults(results, resource); err != nil {
		return fmt.Errorf("failed to aggregate results: %w", err)
	}

	// Print summary if there were errors
	if results.Summary.Failed > 0 {
		fmt.Fprintf(os.Stderr, "\nWarning: Failed to query %d/%d clusters\n",
			results.Summary.Failed, results.Summary.Total)
		for cluster, err := range results.Summary.Errors {
			fmt.Fprintf(os.Stderr, "  - %s: %v\n", cluster, err)
		}
	}

	return nil
}

// filterClusters applies cluster filtering based on --clusters and --exclude flags
func filterClusters(clusters []discovery.ClusterInfo, include, exclude []string) []discovery.ClusterInfo {
	// If no filtering specified, return all clusters
	if len(include) == 0 && len(exclude) == 0 {
		return clusters
	}

	filtered := make([]discovery.ClusterInfo, 0)

	for _, cluster := range clusters {
		// Skip if in exclude list
		if matchesAny(cluster.Name, exclude) {
			continue
		}

		// Include if no include list specified, or if matches include list
		if len(include) == 0 || matchesAny(cluster.Name, include) {
			filtered = append(filtered, cluster)
		}
	}

	return filtered
}

// matchesAny checks if a string matches any of the patterns
// For now, this is a simple string match. Could be enhanced with wildcards later.
func matchesAny(str string, patterns []string) bool {
	for _, pattern := range patterns {
		if str == pattern {
			return true
		}
		// TODO: Add wildcard support (e.g., "prod-*")
	}
	return false
}
