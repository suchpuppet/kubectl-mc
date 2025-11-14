package kubeconfig

// ClusterMapping defines the mapping between ClusterProfile names and kubeconfig contexts
type ClusterMapping struct {
	// Name is the ClusterProfile name
	Name string `yaml:"name"`

	// Context is the kubeconfig context name
	Context string `yaml:"context"`

	// Namespace where the ClusterProfile exists
	Namespace string `yaml:"namespace,omitempty"`
}

// MappingConfig is the configuration file format for cluster mappings
type MappingConfig struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`

	// HubContext is the optional default hub context
	HubContext string `yaml:"hubContext,omitempty"`

	// Clusters is the list of cluster mappings
	Clusters []ClusterMapping `yaml:"clusters"`
}
