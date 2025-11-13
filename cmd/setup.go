package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/suchpuppet/kubectl-mc/pkg/client"
	"github.com/suchpuppet/kubectl-mc/pkg/discovery"
	"github.com/suchpuppet/kubectl-mc/pkg/kubeconfig"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup cluster-to-context mappings",
	Long: `Create or update mappings between ClusterProfile names and kubeconfig contexts.

This command discovers clusters from the hub and prompts you to map each cluster
to a kubeconfig context name.

Example:
  kubectl mc setup`,
	RunE: runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
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
		fmt.Println("No clusters discovered from hub")
		return nil
	}

	fmt.Printf("Discovered %d cluster(s)\n\n", len(clusters))

	// Load existing mappings
	mappingManager, err := kubeconfig.NewManager("")
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig mappings: %w", err)
	}

	// Interactive setup
	reader := bufio.NewReader(os.Stdin)

	for _, cluster := range clusters {
		// Check if mapping already exists
		existingContext, err := mappingManager.GetContext(cluster.Name)
		if err == nil {
			fmt.Printf("Cluster '%s' is already mapped to context '%s'\n", cluster.Name, existingContext)
			fmt.Print("Update mapping? [y/N]: ")
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				continue
			}
		}

		// Prompt for context name
		fmt.Printf("\nCluster: %s (namespace: %s)\n", cluster.DisplayName, cluster.Namespace)
		if cluster.KubernetesVersion != "" {
			fmt.Printf("  Kubernetes version: %s\n", cluster.KubernetesVersion)
		}
		fmt.Printf("  Healthy: %v\n", cluster.Healthy)

		// Suggest a context name based on cluster name
		suggestedContext := fmt.Sprintf("kind-%s", cluster.Name)
		fmt.Printf("Enter kubeconfig context name [%s]: ", suggestedContext)

		contextName, _ := reader.ReadString('\n')
		contextName = strings.TrimSpace(contextName)

		if contextName == "" {
			contextName = suggestedContext
		}

		// Save mapping
		if err := mappingManager.SetMapping(cluster.Name, contextName, cluster.Namespace); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to save mapping for %s: %v\n", cluster.Name, err)
			continue
		}

		fmt.Printf("✓ Mapped '%s' to context '%s'\n", cluster.Name, contextName)
	}

	fmt.Println("\nSetup complete!")
	fmt.Println("Mappings saved to:", "~/.kube/kubectl-mc-clusters.yaml")
	fmt.Println("\nYou can now use: kubectl mc get pods")

	return nil
}
