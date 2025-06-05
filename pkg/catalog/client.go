package catalog

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/giantswarm/mcp-giantswarm-apps/internal/k8s"
)

// Client provides operations for Catalog resources
type Client struct {
	dynamicClient *k8s.DynamicClient
}

// NewClient creates a new catalog client
func NewClient(dynamicClient *k8s.DynamicClient) *Client {
	return &Client{
		dynamicClient: dynamicClient,
	}
}

// List lists catalogs in a namespace or across all namespaces
func (c *Client) List(ctx context.Context, namespace string) ([]*Catalog, error) {
	listOptions := metav1.ListOptions{}

	var list *unstructured.UnstructuredList
	var err error

	if namespace == "" {
		// List across all namespaces
		list, err = c.dynamicClient.Catalogs("").List(ctx, listOptions)
	} else {
		// List in specific namespace
		list, err = c.dynamicClient.Catalogs(namespace).List(ctx, listOptions)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list catalogs: %w", err)
	}

	catalogs := make([]*Catalog, 0, len(list.Items))
	for _, item := range list.Items {
		catalog, err := NewCatalogFromUnstructured(&item)
		if err != nil {
			continue // Skip invalid catalogs
		}
		catalogs = append(catalogs, catalog)
	}

	return catalogs, nil
}

// Get retrieves a specific catalog
func (c *Client) Get(ctx context.Context, namespace, name string) (*Catalog, error) {
	obj, err := c.dynamicClient.Catalogs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get catalog %s/%s: %w", namespace, name, err)
	}

	return NewCatalogFromUnstructured(obj)
}

// Create creates a new catalog
func (c *Client) Create(ctx context.Context, catalog *Catalog) (*Catalog, error) {
	unstructuredCatalog := catalog.ToUnstructured()

	created, err := c.dynamicClient.Catalogs(catalog.Namespace).Create(ctx, unstructuredCatalog, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create catalog %s/%s: %w", catalog.Namespace, catalog.Name, err)
	}

	return NewCatalogFromUnstructured(created)
}

// Update updates an existing catalog
func (c *Client) Update(ctx context.Context, catalog *Catalog) (*Catalog, error) {
	// Get current catalog to preserve metadata
	current, err := c.Get(ctx, catalog.Namespace, catalog.Name)
	if err != nil {
		return nil, err
	}

	// Convert to unstructured
	unstructuredCatalog := catalog.ToUnstructured()

	// Preserve resource version for update
	currentUnstructured := current.ToUnstructured()
	unstructuredCatalog.SetResourceVersion(currentUnstructured.GetResourceVersion())

	updated, err := c.dynamicClient.Catalogs(catalog.Namespace).Update(ctx, unstructuredCatalog, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to update catalog %s/%s: %w", catalog.Namespace, catalog.Name, err)
	}

	return NewCatalogFromUnstructured(updated)
}

// Delete deletes a catalog
func (c *Client) Delete(ctx context.Context, namespace, name string) error {
	err := c.dynamicClient.Catalogs(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete catalog %s/%s: %w", namespace, name, err)
	}

	return nil
}

// FilterByType filters catalogs by type (stable, testing, community)
func FilterByType(catalogs []*Catalog, catalogType string) []*Catalog {
	if catalogType == "" {
		return catalogs
	}

	filtered := make([]*Catalog, 0)
	for _, catalog := range catalogs {
		if catalog.CatalogType() == catalogType {
			filtered = append(filtered, catalog)
		}
	}
	return filtered
}

// FilterByVisibility filters catalogs by visibility (public, private)
func FilterByVisibility(catalogs []*Catalog, visibility string) []*Catalog {
	if visibility == "" {
		return catalogs
	}

	filtered := make([]*Catalog, 0)
	for _, catalog := range catalogs {
		if catalog.CatalogVisibility() == visibility {
			filtered = append(filtered, catalog)
		}
	}
	return filtered
}

// ValidateRepositoryURL validates that a repository URL is accessible
func ValidateRepositoryURL(url string) error {
	// Basic URL validation - in a real implementation, this could make HTTP requests
	// to verify the repository is accessible
	if url == "" {
		return fmt.Errorf("repository URL cannot be empty")
	}
	return nil
}
