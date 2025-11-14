package client

import (
	"fmt"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Factory provides Kubernetes clients for a specific context
type Factory struct {
	context     string
	kubeconfig  string
	configFlags *genericclioptions.ConfigFlags
}

// NewFactory creates a new client factory for the specified context
func NewFactory(context string, configFlags *genericclioptions.ConfigFlags) (*Factory, error) {
	return &Factory{
		context:     context,
		configFlags: configFlags,
	}, nil
}

// RESTConfig returns a REST config for the specified context
func (f *Factory) RESTConfig() (*rest.Config, error) {
	// If context is specified, use it; otherwise use current context
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}

	if f.context != "" {
		configOverrides.CurrentContext = f.context
	}

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	return clientConfig.ClientConfig()
}

// DynamicClient returns a dynamic client
func (f *Factory) DynamicClient() (dynamic.Interface, error) {
	config, err := f.RESTConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get REST config: %w", err)
	}

	return dynamic.NewForConfig(config)
}

// Clientset returns a typed Kubernetes clientset
func (f *Factory) Clientset() (*kubernetes.Clientset, error) {
	config, err := f.RESTConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get REST config: %w", err)
	}

	return kubernetes.NewForConfig(config)
}

// DiscoveryClient returns a discovery client
func (f *Factory) DiscoveryClient() (discovery.DiscoveryInterface, error) {
	config, err := f.RESTConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get REST config: %w", err)
	}

	return discovery.NewDiscoveryClientForConfig(config)
}
