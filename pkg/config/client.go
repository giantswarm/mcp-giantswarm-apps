package config

import (
	"context"
	"fmt"
	"regexp"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Client provides operations for ConfigMaps and Secrets
type Client struct {
	k8sClient kubernetes.Interface
}

// NewClient creates a new config client
func NewClient(k8sClient kubernetes.Interface) *Client {
	return &Client{
		k8sClient: k8sClient,
	}
}

// GetConfigMap retrieves a ConfigMap
func (c *Client) GetConfigMap(ctx context.Context, namespace, name string) (*Config, error) {
	cm, err := c.k8sClient.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get configmap %s/%s: %w", namespace, name, err)
	}

	return NewConfigFromConfigMap(cm), nil
}

// GetSecret retrieves a Secret
func (c *Client) GetSecret(ctx context.Context, namespace, name string) (*Config, error) {
	secret, err := c.k8sClient.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %s/%s: %w", namespace, name, err)
	}

	return NewConfigFromSecret(secret), nil
}

// Get retrieves a configuration (ConfigMap or Secret)
func (c *Client) Get(ctx context.Context, namespace, name string, configType ConfigType) (*Config, error) {
	switch configType {
	case ConfigTypeConfigMap:
		return c.GetConfigMap(ctx, namespace, name)
	case ConfigTypeSecret:
		return c.GetSecret(ctx, namespace, name)
	default:
		return nil, fmt.Errorf("unknown config type: %s", configType)
	}
}

// CreateConfigMap creates a new ConfigMap
func (c *Client) CreateConfigMap(ctx context.Context, config *Config) error {
	cm := config.ToConfigMap()
	_, err := c.k8sClient.CoreV1().ConfigMaps(config.Namespace).Create(ctx, cm, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create configmap %s/%s: %w", config.Namespace, config.Name, err)
	}
	return nil
}

// CreateSecret creates a new Secret
func (c *Client) CreateSecret(ctx context.Context, config *Config) error {
	secret := config.ToSecret()
	_, err := c.k8sClient.CoreV1().Secrets(config.Namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create secret %s/%s: %w", config.Namespace, config.Name, err)
	}
	return nil
}

// Create creates a new configuration (ConfigMap or Secret)
func (c *Client) Create(ctx context.Context, config *Config) error {
	switch config.Type {
	case ConfigTypeConfigMap:
		return c.CreateConfigMap(ctx, config)
	case ConfigTypeSecret:
		return c.CreateSecret(ctx, config)
	default:
		return fmt.Errorf("unknown config type: %s", config.Type)
	}
}

// UpdateConfigMap updates an existing ConfigMap
func (c *Client) UpdateConfigMap(ctx context.Context, config *Config) error {
	cm := config.ToConfigMap()

	// Get current to preserve metadata
	current, err := c.k8sClient.CoreV1().ConfigMaps(config.Namespace).Get(ctx, config.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get current configmap: %w", err)
	}

	cm.ResourceVersion = current.ResourceVersion

	_, err = c.k8sClient.CoreV1().ConfigMaps(config.Namespace).Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update configmap %s/%s: %w", config.Namespace, config.Name, err)
	}
	return nil
}

// UpdateSecret updates an existing Secret
func (c *Client) UpdateSecret(ctx context.Context, config *Config) error {
	secret := config.ToSecret()

	// Get current to preserve metadata
	current, err := c.k8sClient.CoreV1().Secrets(config.Namespace).Get(ctx, config.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get current secret: %w", err)
	}

	secret.ResourceVersion = current.ResourceVersion

	_, err = c.k8sClient.CoreV1().Secrets(config.Namespace).Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update secret %s/%s: %w", config.Namespace, config.Name, err)
	}
	return nil
}

// Update updates an existing configuration (ConfigMap or Secret)
func (c *Client) Update(ctx context.Context, config *Config) error {
	switch config.Type {
	case ConfigTypeConfigMap:
		return c.UpdateConfigMap(ctx, config)
	case ConfigTypeSecret:
		return c.UpdateSecret(ctx, config)
	default:
		return fmt.Errorf("unknown config type: %s", config.Type)
	}
}

// DeleteConfigMap deletes a ConfigMap
func (c *Client) DeleteConfigMap(ctx context.Context, namespace, name string) error {
	err := c.k8sClient.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete configmap %s/%s: %w", namespace, name, err)
	}
	return nil
}

// DeleteSecret deletes a Secret
func (c *Client) DeleteSecret(ctx context.Context, namespace, name string) error {
	err := c.k8sClient.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete secret %s/%s: %w", namespace, name, err)
	}
	return nil
}

// Delete deletes a configuration (ConfigMap or Secret)
func (c *Client) Delete(ctx context.Context, namespace, name string, configType ConfigType) error {
	switch configType {
	case ConfigTypeConfigMap:
		return c.DeleteConfigMap(ctx, namespace, name)
	case ConfigTypeSecret:
		return c.DeleteSecret(ctx, namespace, name)
	default:
		return fmt.Errorf("unknown config type: %s", configType)
	}
}

// ListConfigMaps lists ConfigMaps in a namespace
func (c *Client) ListConfigMaps(ctx context.Context, namespace string, labelSelector string) ([]*Config, error) {
	listOptions := metav1.ListOptions{}
	if labelSelector != "" {
		listOptions.LabelSelector = labelSelector
	}

	cmList, err := c.k8sClient.CoreV1().ConfigMaps(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list configmaps: %w", err)
	}

	configs := make([]*Config, 0, len(cmList.Items))
	for _, cm := range cmList.Items {
		configs = append(configs, NewConfigFromConfigMap(&cm))
	}

	return configs, nil
}

// ListSecrets lists Secrets in a namespace
func (c *Client) ListSecrets(ctx context.Context, namespace string, labelSelector string) ([]*Config, error) {
	listOptions := metav1.ListOptions{}
	if labelSelector != "" {
		listOptions.LabelSelector = labelSelector
	}

	secretList, err := c.k8sClient.CoreV1().Secrets(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	configs := make([]*Config, 0, len(secretList.Items))
	for _, secret := range secretList.Items {
		// Skip service account tokens and other system secrets
		if secret.Type == corev1.SecretTypeServiceAccountToken ||
			secret.Type == corev1.SecretTypeDockercfg ||
			secret.Type == corev1.SecretTypeDockerConfigJson {
			continue
		}
		configs = append(configs, NewConfigFromSecret(&secret))
	}

	return configs, nil
}

// Validate validates a configuration against a schema
func (c *Client) Validate(config *Config, schema *ConfigSchema) *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Errors: []string{},
	}

	// Check required keys
	for _, key := range schema.RequiredKeys {
		if _, exists := config.Data[key]; !exists {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("missing required key: %s", key))
		}
	}

	// Check key patterns
	for key, value := range config.Data {
		// Check if key is allowed
		isAllowed := false
		for _, reqKey := range schema.RequiredKeys {
			if key == reqKey {
				isAllowed = true
				break
			}
		}
		if !isAllowed {
			for _, optKey := range schema.OptionalKeys {
				if key == optKey {
					isAllowed = true
					break
				}
			}
		}

		if !isAllowed && len(schema.RequiredKeys) > 0 || len(schema.OptionalKeys) > 0 {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("unexpected key: %s", key))
		}

		// Validate value pattern if specified
		if pattern, exists := schema.KeyPatterns[key]; exists {
			matched, err := regexp.MatchString(pattern, value)
			if err != nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("invalid pattern for key %s: %v", key, err))
			} else if !matched {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("value for key %s does not match pattern %s", key, pattern))
			}
		}
	}

	return result
}

// GetAppConfig retrieves the configuration for a Giant Swarm app
// It looks for ConfigMaps/Secrets with specific labels
func (c *Client) GetAppConfig(ctx context.Context, namespace, appName string, configType ConfigType) (*Config, error) {
	labelSelector := fmt.Sprintf("app.kubernetes.io/name=%s", appName)

	switch configType {
	case ConfigTypeConfigMap:
		configs, err := c.ListConfigMaps(ctx, namespace, labelSelector)
		if err != nil {
			return nil, err
		}
		if len(configs) == 0 {
			return nil, fmt.Errorf("no configmap found for app %s", appName)
		}
		// Return the first one (there should typically be only one)
		return configs[0], nil

	case ConfigTypeSecret:
		configs, err := c.ListSecrets(ctx, namespace, labelSelector)
		if err != nil {
			return nil, err
		}
		if len(configs) == 0 {
			return nil, fmt.Errorf("no secret found for app %s", appName)
		}
		// Return the first one (there should typically be only one)
		return configs[0], nil

	default:
		return nil, fmt.Errorf("unknown config type: %s", configType)
	}
}

// MergeConfigs merges multiple configurations
// Later configs take precedence
func MergeConfigs(configs ...*Config) *Config {
	if len(configs) == 0 {
		return nil
	}

	merged := &Config{
		Name:      configs[0].Name,
		Namespace: configs[0].Namespace,
		Type:      configs[0].Type,
		Data:      make(map[string]string),
		Labels:    make(map[string]string),
	}

	for _, config := range configs {
		if config != nil {
			merged.MergeWith(config)
			// Also merge labels
			for k, v := range config.Labels {
				merged.Labels[k] = v
			}
		}
	}

	return merged
}
