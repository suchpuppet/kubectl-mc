package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var (
	// cfgFile is the path to the kubectl-mc configuration file
	cfgFile string

	// kubeConfigFlags provides Kubernetes configuration flags
	kubeConfigFlags *genericclioptions.ConfigFlags
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kubectl-mc",
	Short: "Multi-cluster kubectl plugin using sig-multicluster standards",
	Long: `kubectl-mc is a kubectl plugin that provides seamless multi-cluster operations
using sig-multicluster standards (ClusterProfile API).

It extends the familiar kubectl experience to work across multiple clusters,
automatically discovering clusters from a hub and aggregating results.

Example:
  kubectl mc get pods
  kubectl mc get deployments -n default
  kubectl mc describe pod nginx`,
	SilenceUsage: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Initialize kubeconfig flags
	kubeConfigFlags = genericclioptions.NewConfigFlags(true)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.kube/kubectl-mc-config.yaml)")
	rootCmd.PersistentFlags().String("hub-context", "", "kubernetes context for the hub cluster")
	rootCmd.PersistentFlags().String("hub-namespace", "open-cluster-management", "namespace where ClusterProfile resources are located")

	// Add standard kubectl flags
	kubeConfigFlags.AddFlags(rootCmd.PersistentFlags())
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag if specified
		// TODO: Implement config file loading
	} else {
		// Use default config location
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return
		}
		cfgFile = home + "/.kube/kubectl-mc-config.yaml"
	}
}
