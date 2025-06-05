package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Client wraps the Kubernetes client with Giant Swarm specific functionality
type Client struct {
	kubernetes.Interface
	RestConfig *rest.Config
	Context    string
}

// NewClient creates a new Kubernetes client
func NewClient(ctx context.Context, kubeContext string) (*Client, error) {
	config, currentContext, err := getConfig(kubeContext)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Test connection
	_, err = clientset.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to kubernetes cluster: %w", err)
	}

	return &Client{
		Interface:  clientset,
		RestConfig: config,
		Context:    currentContext,
	}, nil
}

// getConfig returns the kubernetes config and current context
func getConfig(kubeContext string) (*rest.Config, string, error) {
	// Try in-cluster config first
	if config, err := rest.InClusterConfig(); err == nil {
		return config, "in-cluster", nil
	}

	// Fall back to kubeconfig
	kubeconfigPath := getKubeconfigPath()
	
	// Build config from kubeconfig file
	configLoadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configLoadingRules.ExplicitPath = kubeconfigPath

	configOverrides := &clientcmd.ConfigOverrides{}
	if kubeContext != "" {
		configOverrides.CurrentContext = kubeContext
	}

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		configLoadingRules,
		configOverrides,
	)

	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, "", fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	// Get current context
	rawConfig, err := kubeConfig.RawConfig()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get raw config: %w", err)
	}

	currentContext := rawConfig.CurrentContext
	if kubeContext != "" {
		currentContext = kubeContext
	}

	return config, currentContext, nil
}

// getKubeconfigPath returns the path to the kubeconfig file
func getKubeconfigPath() string {
	// Check KUBECONFIG env var first
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return kubeconfig
	}

	// Default to ~/.kube/config
	if home := homedir.HomeDir(); home != "" {
		return filepath.Join(home, ".kube", "config")
	}

	return ""
}

// GetCurrentContext returns the current kubernetes context
func (c *Client) GetCurrentContext() string {
	return c.Context
}

// ListContexts returns all available contexts from kubeconfig
func ListContexts() ([]string, string, error) {
	kubeconfigPath := getKubeconfigPath()
	
	configLoadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configLoadingRules.ExplicitPath = kubeconfigPath

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		configLoadingRules,
		&clientcmd.ConfigOverrides{},
	)

	rawConfig, err := kubeConfig.RawConfig()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get raw config: %w", err)
	}

	contexts := make([]string, 0, len(rawConfig.Contexts))
	for name := range rawConfig.Contexts {
		contexts = append(contexts, name)
	}

	return contexts, rawConfig.CurrentContext, nil
} 