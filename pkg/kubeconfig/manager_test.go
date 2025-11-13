package kubeconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetAndGetMapping(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	manager, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// Test setting a mapping
	err = manager.SetMapping("cluster1", "context1", "namespace1")
	if err != nil {
		t.Fatalf("failed to set mapping: %v", err)
	}

	// Test getting the mapping
	context, err := manager.GetContext("cluster1")
	if err != nil {
		t.Fatalf("failed to get context: %v", err)
	}

	if context != "context1" {
		t.Errorf("expected context 'context1', got '%s'", context)
	}

	// Test getting non-existent mapping
	_, err = manager.GetContext("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent cluster, got nil")
	}
}

func TestUpdateMapping(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	manager, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// Set initial mapping
	err = manager.SetMapping("cluster1", "context1", "namespace1")
	if err != nil {
		t.Fatalf("failed to set initial mapping: %v", err)
	}

	// Update the mapping
	err = manager.SetMapping("cluster1", "context2", "namespace2")
	if err != nil {
		t.Fatalf("failed to update mapping: %v", err)
	}

	// Verify the update
	context, err := manager.GetContext("cluster1")
	if err != nil {
		t.Fatalf("failed to get context: %v", err)
	}

	if context != "context2" {
		t.Errorf("expected context 'context2', got '%s'", context)
	}

	// Verify only one mapping exists
	mappings := manager.ListMappings()
	if len(mappings) != 1 {
		t.Errorf("expected 1 mapping, got %d", len(mappings))
	}
}

func TestListMappings(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	manager, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// Add multiple mappings
	clusters := []struct {
		name      string
		context   string
		namespace string
	}{
		{"cluster1", "context1", "ns1"},
		{"cluster2", "context2", "ns2"},
		{"cluster3", "context3", "ns3"},
	}

	for _, c := range clusters {
		err := manager.SetMapping(c.name, c.context, c.namespace)
		if err != nil {
			t.Fatalf("failed to set mapping for %s: %v", c.name, err)
		}
	}

	// List all mappings
	mappings := manager.ListMappings()
	if len(mappings) != len(clusters) {
		t.Errorf("expected %d mappings, got %d", len(clusters), len(mappings))
	}
}

func TestConfigPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	// Create first manager and set mappings
	manager1, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("failed to create first manager: %v", err)
	}

	err = manager1.SetMapping("cluster1", "context1", "namespace1")
	if err != nil {
		t.Fatalf("failed to set mapping: %v", err)
	}

	// Create second manager with same config path
	manager2, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("failed to create second manager: %v", err)
	}

	// Verify mapping persisted
	context, err := manager2.GetContext("cluster1")
	if err != nil {
		t.Fatalf("failed to get context from second manager: %v", err)
	}

	if context != "context1" {
		t.Errorf("expected context 'context1', got '%s'", context)
	}
}

func TestHubContextManagement(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	manager, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// Test setting hub context
	err = manager.SetHubContext("my-hub-context")
	if err != nil {
		t.Fatalf("failed to set hub context: %v", err)
	}

	// Test getting hub context
	hubContext := manager.GetHubContext()
	if hubContext != "my-hub-context" {
		t.Errorf("expected hub context 'my-hub-context', got '%s'", hubContext)
	}
}

func TestDefaultConfigPath(t *testing.T) {
	// Test that default config path uses home directory
	manager, err := NewManager("")
	if err != nil {
		t.Fatalf("failed to create manager with default path: %v", err)
	}

	// Verify config was initialized
	if manager.config == nil {
		t.Error("config should be initialized")
	}

	if manager.config.APIVersion != "kubectl-mc.k8s.io/v1alpha1" {
		t.Errorf("unexpected APIVersion: %s", manager.config.APIVersion)
	}

	if manager.config.Kind != "ClusterMapping" {
		t.Errorf("unexpected Kind: %s", manager.config.Kind)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	// Write invalid YAML
	err := os.WriteFile(configPath, []byte("invalid: yaml: content:"), 0644)
	if err != nil {
		t.Fatalf("failed to write invalid YAML: %v", err)
	}

	// Attempt to load
	_, err = NewManager(configPath)
	if err == nil {
		t.Error("expected error loading invalid YAML, got nil")
	}
}
