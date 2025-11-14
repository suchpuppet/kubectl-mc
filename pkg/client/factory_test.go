package client

import (
	"testing"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func TestNewFactory(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)

	tests := []struct {
		name        string
		context     string
		expectError bool
	}{
		{
			name:        "empty context",
			context:     "",
			expectError: false,
		},
		{
			name:        "with context",
			context:     "test-context",
			expectError: false,
		},
		{
			name:        "with special characters",
			context:     "kind-ocm-hub",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory, err := NewFactory(tt.context, configFlags)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError && factory == nil {
				t.Error("expected factory but got nil")
			}

			if factory != nil && factory.context != tt.context {
				t.Errorf("expected context %s, got %s", tt.context, factory.context)
			}

			// Verify internal fields are set
			if factory != nil && factory.configFlags != configFlags {
				t.Error("configFlags not set correctly")
			}
		})
	}
}

func TestFactoryContext(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)

	tests := []struct {
		name            string
		context         string
		expectedContext string
	}{
		{
			name:            "empty context",
			context:         "",
			expectedContext: "",
		},
		{
			name:            "test context",
			context:         "test-ctx",
			expectedContext: "test-ctx",
		},
		{
			name:            "production context",
			context:         "prod-cluster",
			expectedContext: "prod-cluster",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory, err := NewFactory(tt.context, configFlags)
			if err != nil {
				t.Fatalf("failed to create factory: %v", err)
			}

			if factory.context != tt.expectedContext {
				t.Errorf("expected context %s, got %s", tt.expectedContext, factory.context)
			}
		})
	}
}

func TestFactoryNilConfigFlags(t *testing.T) {
	// NewFactory doesn't validate nil configFlags - it will just panic later
	// This is acceptable since it's an internal API and configFlags should always be provided
	factory, err := NewFactory("test-context", nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if factory == nil {
		t.Error("expected factory, got nil")
	}
}

func TestFactoryRESTConfig(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	factory, err := NewFactory("", configFlags)
	if err != nil {
		t.Fatalf("failed to create factory: %v", err)
	}

	// RESTConfig will likely fail without a real kubeconfig, but we can test it exists
	_, err = factory.RESTConfig()
	// We expect an error since there's no valid kubeconfig in test environment
	// Just verify the method doesn't panic
	if err == nil {
		t.Log("RESTConfig returned successfully (unexpected in test environment)")
	}
}

func TestFactoryDynamicClient(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	factory, err := NewFactory("nonexistent-context", configFlags)
	if err != nil {
		t.Fatalf("failed to create factory: %v", err)
	}

	// DynamicClient should fail gracefully without valid kubeconfig
	client, err := factory.DynamicClient()
	if err == nil && client == nil {
		t.Error("expected either error or client, got neither")
	}
	// Error is expected in test environment - just verify it doesn't panic
}

func TestFactoryClientset(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	factory, err := NewFactory("nonexistent-context", configFlags)
	if err != nil {
		t.Fatalf("failed to create factory: %v", err)
	}

	// Clientset should fail gracefully without valid kubeconfig
	clientset, err := factory.Clientset()
	if err == nil && clientset == nil {
		t.Error("expected either error or clientset, got neither")
	}
	// Error is expected in test environment - just verify it doesn't panic
}

func TestFactoryDiscoveryClient(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	factory, err := NewFactory("nonexistent-context", configFlags)
	if err != nil {
		t.Fatalf("failed to create factory: %v", err)
	}

	// DiscoveryClient should fail gracefully without valid kubeconfig
	discoveryClient, err := factory.DiscoveryClient()
	if err == nil && discoveryClient == nil {
		t.Error("expected either error or discovery client, got neither")
	}
	// Error is expected in test environment - just verify it doesn't panic
}

func TestFactoryMultipleContexts(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)

	contexts := []string{"ctx1", "ctx2", "ctx3"}
	factories := make([]*Factory, len(contexts))

	for i, ctx := range contexts {
		factory, err := NewFactory(ctx, configFlags)
		if err != nil {
			t.Errorf("failed to create factory for context %s: %v", ctx, err)
		}
		factories[i] = factory
	}

	// Verify each factory has the correct context
	for i, factory := range factories {
		if factory.context != contexts[i] {
			t.Errorf("factory %d: expected context %s, got %s", i, contexts[i], factory.context)
		}
	}
}
