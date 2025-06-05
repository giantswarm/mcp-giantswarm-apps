package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// ConfigType represents the type of configuration
type ConfigType string

const (
	ConfigTypeConfigMap ConfigType = "configmap"
	ConfigTypeSecret    ConfigType = "secret"
)

// Config represents a configuration (ConfigMap or Secret)
type Config struct {
	Name      string
	Namespace string
	Type      ConfigType
	Data      map[string]string
	Labels    map[string]string
}

// ConfigDiff represents differences between two configurations
type ConfigDiff struct {
	Added    map[string]string
	Modified map[string]DiffEntry
	Removed  map[string]string
}

// DiffEntry represents a modified configuration entry
type DiffEntry struct {
	Old string
	New string
}

// ValidationResult represents the result of configuration validation
type ValidationResult struct {
	Valid  bool
	Errors []string
}

// ConfigSchema represents a simple schema for configuration validation
type ConfigSchema struct {
	RequiredKeys []string
	OptionalKeys []string
	KeyPatterns  map[string]string // Regular expression patterns for key validation
}

// IsSecret returns true if this is a secret configuration
func (c *Config) IsSecret() bool {
	return c.Type == ConfigTypeSecret
}

// GetValue retrieves a configuration value
func (c *Config) GetValue(key string) (string, bool) {
	value, exists := c.Data[key]
	return value, exists
}

// SetValue sets a configuration value
func (c *Config) SetValue(key, value string) {
	if c.Data == nil {
		c.Data = make(map[string]string)
	}
	c.Data[key] = value
}

// RemoveValue removes a configuration value
func (c *Config) RemoveValue(key string) {
	delete(c.Data, key)
}

// ToYAML converts the configuration data to YAML
func (c *Config) ToYAML() (string, error) {
	yamlData, err := yaml.Marshal(c.Data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal to YAML: %w", err)
	}
	return string(yamlData), nil
}

// ToJSON converts the configuration data to JSON
func (c *Config) ToJSON() (string, error) {
	jsonData, err := json.MarshalIndent(c.Data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal to JSON: %w", err)
	}
	return string(jsonData), nil
}

// FromYAML populates the configuration data from YAML
func (c *Config) FromYAML(yamlData string) error {
	if c.Data == nil {
		c.Data = make(map[string]string)
	}
	
	var data map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlData), &data); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %w", err)
	}
	
	// Convert to string map
	for k, v := range data {
		c.Data[k] = fmt.Sprintf("%v", v)
	}
	
	return nil
}

// FromJSON populates the configuration data from JSON
func (c *Config) FromJSON(jsonData string) error {
	if c.Data == nil {
		c.Data = make(map[string]string)
	}
	
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	
	// Convert to string map
	for k, v := range data {
		c.Data[k] = fmt.Sprintf("%v", v)
	}
	
	return nil
}

// MergeWith merges another configuration into this one
// The other configuration takes precedence for conflicting keys
func (c *Config) MergeWith(other *Config) {
	if c.Data == nil {
		c.Data = make(map[string]string)
	}
	
	for k, v := range other.Data {
		c.Data[k] = v
	}
}

// Diff compares this configuration with another
func (c *Config) Diff(other *Config) *ConfigDiff {
	diff := &ConfigDiff{
		Added:    make(map[string]string),
		Modified: make(map[string]DiffEntry),
		Removed:  make(map[string]string),
	}
	
	// Check for added and modified keys
	for k, v := range other.Data {
		if oldV, exists := c.Data[k]; exists {
			if oldV != v {
				diff.Modified[k] = DiffEntry{Old: oldV, New: v}
			}
		} else {
			diff.Added[k] = v
		}
	}
	
	// Check for removed keys
	for k, v := range c.Data {
		if _, exists := other.Data[k]; !exists {
			diff.Removed[k] = v
		}
	}
	
	return diff
}

// HasChanges returns true if there are any differences
func (d *ConfigDiff) HasChanges() bool {
	return len(d.Added) > 0 || len(d.Modified) > 0 || len(d.Removed) > 0
}

// NewConfigFromConfigMap creates a Config from a Kubernetes ConfigMap
func NewConfigFromConfigMap(cm *corev1.ConfigMap) *Config {
	return &Config{
		Name:      cm.Name,
		Namespace: cm.Namespace,
		Type:      ConfigTypeConfigMap,
		Data:      cm.Data,
		Labels:    cm.Labels,
	}
}

// NewConfigFromSecret creates a Config from a Kubernetes Secret
func NewConfigFromSecret(secret *corev1.Secret) *Config {
	config := &Config{
		Name:      secret.Name,
		Namespace: secret.Namespace,
		Type:      ConfigTypeSecret,
		Data:      make(map[string]string),
		Labels:    secret.Labels,
	}
	
	// Decode secret data
	for k, v := range secret.Data {
		config.Data[k] = string(v)
	}
	
	return config
}

// ToConfigMap converts a Config to a Kubernetes ConfigMap
func (c *Config) ToConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Name,
			Namespace: c.Namespace,
			Labels:    c.Labels,
		},
		Data: c.Data,
	}
}

// ToSecret converts a Config to a Kubernetes Secret
func (c *Config) ToSecret() *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Name,
			Namespace: c.Namespace,
			Labels:    c.Labels,
		},
		Type: corev1.SecretTypeOpaque,
		Data: make(map[string][]byte),
	}
	
	// Encode secret data
	for k, v := range c.Data {
		secret.Data[k] = []byte(v)
	}
	
	return secret
}

// EncodeSecretData base64 encodes all values (for display purposes)
func (c *Config) EncodeSecretData() {
	if c.Type != ConfigTypeSecret {
		return
	}
	
	encoded := make(map[string]string)
	for k, v := range c.Data {
		encoded[k] = base64.StdEncoding.EncodeToString([]byte(v))
	}
	c.Data = encoded
}

// DecodeSecretData base64 decodes all values
func (c *Config) DecodeSecretData() error {
	if c.Type != ConfigTypeSecret {
		return nil
	}
	
	decoded := make(map[string]string)
	for k, v := range c.Data {
		data, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return fmt.Errorf("failed to decode key %s: %w", k, err)
		}
		decoded[k] = string(data)
	}
	c.Data = decoded
	return nil
} 