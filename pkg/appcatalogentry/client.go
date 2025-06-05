package appcatalogentry

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/giantswarm/mcp-giantswarm-apps/internal/k8s"
)

// Client provides operations for AppCatalogEntry resources
type Client struct {
	dynamicClient *k8s.DynamicClient
}

// NewClient creates a new AppCatalogEntry client
func NewClient(dynamicClient *k8s.DynamicClient) *Client {
	return &Client{
		dynamicClient: dynamicClient,
	}
}

// List lists AppCatalogEntries in a namespace or across all namespaces
func (c *Client) List(ctx context.Context, namespace string) ([]*AppCatalogEntry, error) {
	listOptions := metav1.ListOptions{}

	var list *unstructured.UnstructuredList
	var err error

	if namespace == "" {
		// List across all namespaces
		list, err = c.dynamicClient.AppCatalogEntries("").List(ctx, listOptions)
	} else {
		// List in specific namespace
		list, err = c.dynamicClient.AppCatalogEntries(namespace).List(ctx, listOptions)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list app catalog entries: %w", err)
	}

	entries := make([]*AppCatalogEntry, 0, len(list.Items))
	for _, item := range list.Items {
		entry, err := NewAppCatalogEntryFromUnstructured(&item)
		if err != nil {
			continue // Skip invalid entries
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// ListByCatalog lists AppCatalogEntries for a specific catalog
func (c *Client) ListByCatalog(ctx context.Context, catalogName, catalogNamespace string) ([]*AppCatalogEntry, error) {
	// List all entries and filter by catalog
	entries, err := c.List(ctx, "")
	if err != nil {
		return nil, err
	}

	filtered := make([]*AppCatalogEntry, 0)
	for _, entry := range entries {
		if entry.Spec.Catalog.Name == catalogName &&
			(catalogNamespace == "" || entry.Spec.Catalog.Namespace == catalogNamespace) {
			filtered = append(filtered, entry)
		}
	}

	return filtered, nil
}

// Get retrieves a specific AppCatalogEntry
func (c *Client) Get(ctx context.Context, namespace, name string) (*AppCatalogEntry, error) {
	obj, err := c.dynamicClient.AppCatalogEntries(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get app catalog entry %s/%s: %w", namespace, name, err)
	}

	return NewAppCatalogEntryFromUnstructured(obj)
}

// Search searches for AppCatalogEntries by app name or keywords
func (c *Client) Search(ctx context.Context, query string) ([]*AppCatalogEntry, error) {
	entries, err := c.List(ctx, "")
	if err != nil {
		return nil, err
	}

	query = strings.ToLower(query)
	results := make([]*AppCatalogEntry, 0)

	for _, entry := range entries {
		// Search in app name
		if strings.Contains(strings.ToLower(entry.Spec.AppName), query) {
			results = append(results, entry)
			continue
		}

		// Search in chart name
		if strings.Contains(strings.ToLower(entry.Spec.Chart.Name), query) {
			results = append(results, entry)
			continue
		}

		// Search in description
		if strings.Contains(strings.ToLower(entry.Spec.Chart.Description), query) {
			results = append(results, entry)
			continue
		}

		// Search in keywords
		for _, keyword := range entry.Spec.Chart.Keywords {
			if strings.Contains(strings.ToLower(keyword), query) {
				results = append(results, entry)
				break
			}
		}
	}

	return results, nil
}

// GetVersions gets all available versions for an app
func (c *Client) GetVersions(ctx context.Context, appName string) ([]*AppCatalogEntry, error) {
	entries, err := c.List(ctx, "")
	if err != nil {
		return nil, err
	}

	versions := make([]*AppCatalogEntry, 0)
	for _, entry := range entries {
		if entry.Spec.AppName == appName || entry.Spec.Chart.Name == appName {
			versions = append(versions, entry)
		}
	}

	return versions, nil
}

// FilterByLabels filters entries by label selector
func (c *Client) FilterByLabels(ctx context.Context, labelSelector string) ([]*AppCatalogEntry, error) {
	selector, err := labels.Parse(labelSelector)
	if err != nil {
		return nil, fmt.Errorf("invalid label selector: %w", err)
	}

	entries, err := c.List(ctx, "")
	if err != nil {
		return nil, err
	}

	filtered := make([]*AppCatalogEntry, 0)
	for _, entry := range entries {
		if selector.Matches(labels.Set(entry.Labels)) {
			filtered = append(filtered, entry)
		}
	}

	return filtered, nil
}

// FilterByRestrictions filters entries by restrictions
func FilterByRestrictions(entries []*AppCatalogEntry, clusterApp bool) []*AppCatalogEntry {
	filtered := make([]*AppCatalogEntry, 0)
	for _, entry := range entries {
		if entry.IsClusterApp() == clusterApp {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// SortByDate sorts entries by date (newest first)
func SortByDate(entries []*AppCatalogEntry) []*AppCatalogEntry {
	// Simple bubble sort for now - in production, use sort.Slice
	sorted := make([]*AppCatalogEntry, len(entries))
	copy(sorted, entries)

	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			date1 := sorted[j].Spec.DateUpdated
			if date1 == nil {
				date1 = sorted[j].Spec.DateCreated
			}
			date2 := sorted[j+1].Spec.DateUpdated
			if date2 == nil {
				date2 = sorted[j+1].Spec.DateCreated
			}

			if date1 != nil && date2 != nil && date1.Before(*date2) {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	return sorted
}

// GroupByApp groups entries by app name
func GroupByApp(entries []*AppCatalogEntry) map[string][]*AppCatalogEntry {
	grouped := make(map[string][]*AppCatalogEntry)

	for _, entry := range entries {
		appName := entry.Spec.AppName
		if appName == "" {
			appName = entry.Spec.Chart.Name
		}

		if _, exists := grouped[appName]; !exists {
			grouped[appName] = make([]*AppCatalogEntry, 0)
		}
		grouped[appName] = append(grouped[appName], entry)
	}

	return grouped
}
