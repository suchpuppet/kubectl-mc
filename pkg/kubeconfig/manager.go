package kubeconfig

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Manager handles cluster-to-context mappings
type Manager struct {
	configPath string
	config     *MappingConfig
}

// NewManager creates a new kubeconfig mapping manager
func NewManager(configPath string) (*Manager, error) {
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = filepath.Join(home, ".kube", "kubectl-mc-clusters.yaml")
	}

	m := &Manager{
		configPath: configPath,
	}

	// Load existing config or create empty one
	if err := m.load(); err != nil {
		// If file doesn't exist, initialize with empty config
		if os.IsNotExist(err) {
			m.config = &MappingConfig{
				APIVersion: "kubectl-mc.k8s.io/v1alpha1",
				Kind:       "ClusterMapping",
				Clusters:   []ClusterMapping{},
			}
		} else {
			return nil, err
		}
	}

	return m, nil
}

// GetContext returns the kubeconfig context for a cluster name
func (m *Manager) GetContext(clusterName string) (string, error) {
	for _, mapping := range m.config.Clusters {
		if mapping.Name == clusterName {
			return mapping.Context, nil
		}
	}
	return "", fmt.Errorf("no context mapping found for cluster %s", clusterName)
}

// SetMapping adds or updates a cluster-to-context mapping
func (m *Manager) SetMapping(clusterName, context, namespace string) error {
	// Check if mapping already exists
	for i, mapping := range m.config.Clusters {
		if mapping.Name == clusterName {
			m.config.Clusters[i].Context = context
			m.config.Clusters[i].Namespace = namespace
			return m.save()
		}
	}

	// Add new mapping
	m.config.Clusters = append(m.config.Clusters, ClusterMapping{
		Name:      clusterName,
		Context:   context,
		Namespace: namespace,
	})

	return m.save()
}

// ListMappings returns all cluster mappings
func (m *Manager) ListMappings() []ClusterMapping {
	return m.config.Clusters
}

// load reads the mapping config from disk
func (m *Manager) load() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return err
	}

	m.config = &MappingConfig{}
	if err := yaml.Unmarshal(data, m.config); err != nil {
		return fmt.Errorf("failed to parse mapping config: %w", err)
	}

	return nil
}

// save writes the mapping config to disk
func (m *Manager) save() error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(m.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(m.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetHubContext returns the configured hub context if set
func (m *Manager) GetHubContext() string {
	return m.config.HubContext
}

// SetHubContext sets the default hub context
func (m *Manager) SetHubContext(context string) error {
	m.config.HubContext = context
	return m.save()
}
