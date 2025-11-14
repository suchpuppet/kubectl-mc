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
	// describeCmd represents the describe command
	describeCmd = &cobra.Command{
		Use:   "describe [resource] [name]",
		Short: "Describe resources across multiple clusters",
		Long: `Describe resources across all discovered clusters and display detailed information.

Examples:
  # Describe a specific pod across all clusters
  kubectl mc describe pod nginx

  # Describe all pods in a namespace
  kubectl mc describe pods -n default

  # Describe a deployment
  kubectl mc describe deployment my-app`,
		Args: cobra.MinimumNArgs(1),
		RunE: runDescribe,
	}
)

func init() {
	rootCmd.AddCommand(describeCmd)

	// Add cluster filtering flags (reuse same flags as get)
	describeCmd.Flags().StringSliceVar(&clustersFlag, "clusters", []string{}, "comma-separated list of cluster names or patterns")
	describeCmd.Flags().StringSliceVar(&excludeFlag, "exclude", []string{}, "comma-separated list of cluster names or patterns to exclude")
	describeCmd.Flags().BoolVar(&allClusters, "all-clusters", false, "target all clusters (explicit confirmation)")
	
	// Add all-namespaces flag (kubectl standard -A)
	describeCmd.Flags().BoolP("all-namespaces", "A", false, "query resources across all namespaces")
}

func runDescribe(cmd *cobra.Command, args []string) error {
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

	// Determine namespace to use
	var namespace string
	allNamespaces, _ := cmd.Flags().GetBool("all-namespaces")
	
	if allNamespaces {
		// -A flag: query all namespaces
		namespace = ""
	} else if cmd.Flags().Changed("namespace") {
		// -n flag explicitly set: use that namespace
		namespace, _ = cmd.Flags().GetString("namespace")
	} else {
		// Neither flag set: use kubeconfig default namespace
		// Pass the namespace from kubeConfigFlags which respects kubeconfig context
		if kubeConfigFlags.Namespace != nil && *kubeConfigFlags.Namespace != "" {
			namespace = *kubeConfigFlags.Namespace
		} else {
			// No namespace in kubeconfig either, default to "default"
			namespace = "default"
		}
	}

	// Execute describe across all clusters
	results, err := exec.Describe(ctx, filteredClusters, resource, resourceName, namespace)
	if err != nil {
		return fmt.Errorf("failed to execute describe: %w", err)
	}

	// Aggregate and format results
	agg := aggregator.NewDescribeAggregator(os.Stdout)
	if err := agg.AggregateDescribeResults(results, resource); err != nil {
		return fmt.Errorf("failed to aggregate results: %w", err)
	}

	// Only print errors if ALL clusters failed (when at least one succeeded, silently ignore failures)
	if results.Summary.Failed > 0 && results.Summary.Successful == 0 {
		fmt.Fprintf(os.Stderr, "\nError: Failed to query all %d clusters\n", results.Summary.Total)
		for cluster, err := range results.Summary.Errors {
			fmt.Fprintf(os.Stderr, "  - %s: %v\n", cluster, err)
		}
		return fmt.Errorf("all clusters failed")
	}

	return nil
}
