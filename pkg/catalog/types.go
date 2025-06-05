package catalog

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Catalog represents a Giant Swarm Catalog resource
type Catalog struct {
	Name      string
	Namespace string
	Spec      CatalogSpec
	Labels    map[string]string
}

// CatalogSpec represents the spec of a Catalog
type CatalogSpec struct {
	Title        string
	Description  string
	LogoURL      string
	Storage      Storage
	Repositories []Repository
	Config       *CatalogConfig
}

// Storage represents the storage configuration
type Storage struct {
	Type string
	URL  string
}

// Repository represents a catalog repository
type Repository struct {
	Type string
	URL  string
}

// CatalogConfig represents catalog configuration
type CatalogConfig struct {
	ConfigMap *ConfigMapReference
	Secret    *SecretReference
}

// ConfigMapReference references a configmap
type ConfigMapReference struct {
	Name      string
	Namespace string
}

// SecretReference references a secret
type SecretReference struct {
	Name      string
	Namespace string
}

// CatalogType represents the type of catalog (stable, testing, community)
func (c *Catalog) CatalogType() string {
	if catalogType, ok := c.Labels["application.giantswarm.io/catalog-type"]; ok {
		return catalogType
	}
	return "unknown"
}

// CatalogVisibility represents the visibility of catalog (public, private)
func (c *Catalog) CatalogVisibility() string {
	if visibility, ok := c.Labels["application.giantswarm.io/catalog-visibility"]; ok {
		return visibility
	}
	return "unknown"
}

// NewCatalogFromUnstructured converts an unstructured object to a Catalog
func NewCatalogFromUnstructured(obj *unstructured.Unstructured) (*Catalog, error) {
	catalog := &Catalog{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
		Labels:    obj.GetLabels(),
	}

	// Extract spec
	spec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil || !found {
		return catalog, err
	}

	// Title
	if title, ok := spec["title"].(string); ok {
		catalog.Spec.Title = title
	}

	// Description
	if description, ok := spec["description"].(string); ok {
		catalog.Spec.Description = description
	}

	// LogoURL
	if logoURL, ok := spec["logoURL"].(string); ok {
		catalog.Spec.LogoURL = logoURL
	}

	// Storage
	if storage, ok := spec["storage"].(map[string]interface{}); ok {
		if storageType, ok := storage["type"].(string); ok {
			catalog.Spec.Storage.Type = storageType
		}
		if url, ok := storage["URL"].(string); ok {
			catalog.Spec.Storage.URL = url
		}
	}

	// Repositories
	if repositories, ok := spec["repositories"].([]interface{}); ok {
		catalog.Spec.Repositories = make([]Repository, 0, len(repositories))
		for _, repo := range repositories {
			if repoMap, ok := repo.(map[string]interface{}); ok {
				repository := Repository{}
				if repoType, ok := repoMap["type"].(string); ok {
					repository.Type = repoType
				}
				if url, ok := repoMap["URL"].(string); ok {
					repository.URL = url
				}
				catalog.Spec.Repositories = append(catalog.Spec.Repositories, repository)
			}
		}
	}

	// Config
	if config, ok := spec["config"].(map[string]interface{}); ok {
		catalog.Spec.Config = parseCatalogConfig(config)
	}

	return catalog, nil
}

// parseCatalogConfig parses catalog config from unstructured data
func parseCatalogConfig(config map[string]interface{}) *CatalogConfig {
	cc := &CatalogConfig{}

	if configMap, ok := config["configMap"].(map[string]interface{}); ok {
		cc.ConfigMap = &ConfigMapReference{}
		if name, ok := configMap["name"].(string); ok {
			cc.ConfigMap.Name = name
		}
		if namespace, ok := configMap["namespace"].(string); ok {
			cc.ConfigMap.Namespace = namespace
		}
	}

	if secret, ok := config["secret"].(map[string]interface{}); ok {
		cc.Secret = &SecretReference{}
		if name, ok := secret["name"].(string); ok {
			cc.Secret.Name = name
		}
		if namespace, ok := secret["namespace"].(string); ok {
			cc.Secret.Namespace = namespace
		}
	}

	return cc
}

// ToUnstructured converts a Catalog to an unstructured object
func (c *Catalog) ToUnstructured() *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "application.giantswarm.io/v1alpha1",
			"kind":       "Catalog",
			"metadata": map[string]interface{}{
				"name":      c.Name,
				"namespace": c.Namespace,
				"labels":    c.Labels,
			},
			"spec": map[string]interface{}{
				"title":       c.Spec.Title,
				"description": c.Spec.Description,
				"logoURL":     c.Spec.LogoURL,
				"storage": map[string]interface{}{
					"type": c.Spec.Storage.Type,
					"URL":  c.Spec.Storage.URL,
				},
			},
		},
	}

	// Add repositories
	if len(c.Spec.Repositories) > 0 {
		spec := obj.Object["spec"].(map[string]interface{})
		repos := make([]interface{}, 0, len(c.Spec.Repositories))
		for _, repo := range c.Spec.Repositories {
			repos = append(repos, map[string]interface{}{
				"type": repo.Type,
				"URL":  repo.URL,
			})
		}
		spec["repositories"] = repos
	}

	// Add config if present
	if c.Spec.Config != nil {
		spec := obj.Object["spec"].(map[string]interface{})
		config := make(map[string]interface{})

		if c.Spec.Config.ConfigMap != nil {
			config["configMap"] = map[string]interface{}{
				"name":      c.Spec.Config.ConfigMap.Name,
				"namespace": c.Spec.Config.ConfigMap.Namespace,
			}
		}

		if c.Spec.Config.Secret != nil {
			config["secret"] = map[string]interface{}{
				"name":      c.Spec.Config.Secret.Name,
				"namespace": c.Spec.Config.Secret.Namespace,
			}
		}

		spec["config"] = config
	}

	return obj
}
