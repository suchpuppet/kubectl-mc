package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/suchpuppet/kubectl-mc/pkg/client"
	"github.com/suchpuppet/kubectl-mc/pkg/discovery"
	"github.com/suchpuppet/kubectl-mc/pkg/kubeconfig"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8sdiscovery "k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
)

// Executor handles multi-cluster command execution
type Executor struct {
	mappingManager *kubeconfig.Manager
	configFlags    *genericclioptions.ConfigFlags
	config         ExecutorConfig
}

// NewExecutor creates a new multi-cluster executor
func NewExecutor(mappingManager *kubeconfig.Manager, configFlags *genericclioptions.ConfigFlags) *Executor {
	return &Executor{
		mappingManager: mappingManager,
		configFlags:    configFlags,
		config:         DefaultConfig(),
	}
}

// Get executes a get command across multiple clusters
func (e *Executor) Get(ctx context.Context, clusters []discovery.ClusterInfo, resource, name, namespace string) (*AggregatedResults, error) {
	results := NewAggregatedResults(clusters)

	// Create a channel for results
	resultChan := make(chan ClusterResult, len(clusters))

	// Create semaphore for concurrency control
	sem := make(chan struct{}, e.config.MaxConcurrency)

	// WaitGroup to wait for all goroutines
	var wg sync.WaitGroup

	// Execute get on each cluster in parallel
	for _, cluster := range clusters {
		wg.Add(1)
		go func(c discovery.ClusterInfo) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Create context with timeout
			ctx, cancel := context.WithTimeout(ctx, time.Duration(e.config.TimeoutSeconds)*time.Second)
			defer cancel()

			// Execute get on this cluster
			result := e.getFromCluster(ctx, c, resource, name, namespace)
			resultChan <- result
		}(cluster)
	}

	// Wait for all goroutines to complete and close the channel
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for result := range resultChan {
		results.AddResult(result)
	}

	return results, nil
}

// getFromCluster executes a get command on a single cluster
func (e *Executor) getFromCluster(ctx context.Context, cluster discovery.ClusterInfo, resource, name, namespace string) ClusterResult {
	result := ClusterResult{
		ClusterName: cluster.Name,
		Items:       []unstructured.Unstructured{},
	}

	// Get the kubeconfig context for this cluster
	contextName, err := e.mappingManager.GetContext(cluster.Name)
	if err != nil {
		result.Error = fmt.Errorf("no kubeconfig context mapped for cluster %s", cluster.Name)
		return result
	}

	// Create client factory for this cluster's context
	factory, err := client.NewFactory(contextName, e.configFlags)
	if err != nil {
		result.Error = fmt.Errorf("failed to create client factory: %w", err)
		return result
	}

	// Get dynamic client
	dynamicClient, err := factory.DynamicClient()
	if err != nil {
		result.Error = fmt.Errorf("failed to create dynamic client: %w", err)
		return result
	}

	// Get discovery client to resolve resource types
	discoveryClient, err := factory.DiscoveryClient()
	if err != nil {
		result.Error = fmt.Errorf("failed to create discovery client: %w", err)
		return result
	}

	// Resolve the GVR for the resource
	gvr, err := e.resolveGVR(discoveryClient, resource)
	if err != nil {
		result.Error = fmt.Errorf("failed to resolve resource type: %w", err)
		return result
	}

	// Execute the get operation
	var resourceInterface dynamic.ResourceInterface
	if namespace != "" {
		resourceInterface = dynamicClient.Resource(gvr).Namespace(namespace)
	} else {
		resourceInterface = dynamicClient.Resource(gvr)
	}

	if name != "" {
		// Get specific resource
		item, err := resourceInterface.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			result.Error = fmt.Errorf("failed to get resource: %w", err)
			return result
		}
		result.Items = append(result.Items, *item)
	} else {
		// List resources
		list, err := resourceInterface.List(ctx, metav1.ListOptions{})
		if err != nil {
			result.Error = fmt.Errorf("failed to list resources: %w", err)
			return result
		}
		result.Items = append(result.Items, list.Items...)
	}

	result.Success = true
	return result
}

// resolveGVR resolves a resource name to its GroupVersionResource
func (e *Executor) resolveGVR(discoveryClient k8sdiscovery.DiscoveryInterface, resource string) (schema.GroupVersionResource, error) {
	// This is a simplified implementation.
	// A production version would use kubectl's resource mapper for better resolution.

	// Common resource mappings (simplified)
	commonResources := map[string]schema.GroupVersionResource{
		"pods":        {Group: "", Version: "v1", Resource: "pods"},
		"pod":         {Group: "", Version: "v1", Resource: "pods"},
		"services":    {Group: "", Version: "v1", Resource: "services"},
		"service":     {Group: "", Version: "v1", Resource: "services"},
		"deployments": {Group: "apps", Version: "v1", Resource: "deployments"},
		"deployment":  {Group: "apps", Version: "v1", Resource: "deployments"},
		"configmaps":  {Group: "", Version: "v1", Resource: "configmaps"},
		"configmap":   {Group: "", Version: "v1", Resource: "configmaps"},
		"secrets":     {Group: "", Version: "v1", Resource: "secrets"},
		"secret":      {Group: "", Version: "v1", Resource: "secrets"},
		"namespaces":  {Group: "", Version: "v1", Resource: "namespaces"},
		"namespace":   {Group: "", Version: "v1", Resource: "namespaces"},
	}

	gvr, ok := commonResources[resource]
	if !ok {
		return schema.GroupVersionResource{}, fmt.Errorf("unknown resource type: %s", resource)
	}

	return gvr, nil
}
