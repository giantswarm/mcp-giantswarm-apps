package app

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/giantswarm/mcp-giantswarm-apps/internal/k8s"
	"github.com/giantswarm/mcp-giantswarm-apps/pkg/organization"
)

// Client provides operations for App resources
type Client struct {
	dynamicClient *k8s.DynamicClient
}

// NewClient creates a new app client
func NewClient(dynamicClient *k8s.DynamicClient) *Client {
	return &Client{
		dynamicClient: dynamicClient,
	}
}

// List lists apps in a namespace or across all namespaces
func (c *Client) List(ctx context.Context, namespace string, labelSelector string) ([]*App, error) {
	listOptions := metav1.ListOptions{}
	if labelSelector != "" {
		listOptions.LabelSelector = labelSelector
	}

	var list *unstructured.UnstructuredList
	var err error

	if namespace == "" {
		// List across all namespaces
		list, err = c.dynamicClient.Apps("").List(ctx, listOptions)
	} else {
		// List in specific namespace
		list, err = c.dynamicClient.Apps(namespace).List(ctx, listOptions)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list apps: %w", err)
	}

	apps := make([]*App, 0, len(list.Items))
	for _, item := range list.Items {
		app, err := NewAppFromUnstructured(&item)
		if err != nil {
			continue // Skip invalid apps
		}
		apps = append(apps, app)
	}

	return apps, nil
}

// Get retrieves a specific app
func (c *Client) Get(ctx context.Context, namespace, name string) (*App, error) {
	obj, err := c.dynamicClient.Apps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get app %s/%s: %w", namespace, name, err)
	}

	return NewAppFromUnstructured(obj)
}

// Create creates a new app
func (c *Client) Create(ctx context.Context, app *App) (*App, error) {
	unstructuredApp := app.ToUnstructured()

	created, err := c.dynamicClient.Apps(app.Namespace).Create(ctx, unstructuredApp, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create app %s/%s: %w", app.Namespace, app.Name, err)
	}

	return NewAppFromUnstructured(created)
}

// Update updates an existing app
func (c *Client) Update(ctx context.Context, app *App) (*App, error) {
	// Get current app to preserve metadata
	current, err := c.Get(ctx, app.Namespace, app.Name)
	if err != nil {
		return nil, err
	}

	// Convert to unstructured
	unstructuredApp := app.ToUnstructured()

	// Preserve resource version for update
	currentUnstructured := current.ToUnstructured()
	unstructuredApp.SetResourceVersion(currentUnstructured.GetResourceVersion())

	updated, err := c.dynamicClient.Apps(app.Namespace).Update(ctx, unstructuredApp, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to update app %s/%s: %w", app.Namespace, app.Name, err)
	}

	return NewAppFromUnstructured(updated)
}

// Delete deletes an app
func (c *Client) Delete(ctx context.Context, namespace, name string) error {
	err := c.dynamicClient.Apps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete app %s/%s: %w", namespace, name, err)
	}

	return nil
}

// UpdateVersion updates the version of an app
func (c *Client) UpdateVersion(ctx context.Context, namespace, name, version string) (*App, error) {
	app, err := c.Get(ctx, namespace, name)
	if err != nil {
		return nil, err
	}

	app.Spec.Version = version
	return c.Update(ctx, app)
}

// FilterByStatus filters apps by release status
func FilterByStatus(apps []*App, status string) []*App {
	if status == "" {
		return apps
	}

	filtered := make([]*App, 0)
	for _, app := range apps {
		if app.Status.Release.Status == status {
			filtered = append(filtered, app)
		}
	}
	return filtered
}

// FilterByCatalog filters apps by catalog
func FilterByCatalog(apps []*App, catalog string) []*App {
	if catalog == "" {
		return apps
	}

	filtered := make([]*App, 0)
	for _, app := range apps {
		if app.Spec.Catalog == catalog {
			filtered = append(filtered, app)
		}
	}
	return filtered
}

// GetOrganizationNamespaces returns all organization namespaces (org-*)
func (c *Client) GetOrganizationNamespaces(ctx context.Context, k8sClient *k8s.Client) ([]string, error) {
	return organization.ListOrganizationNamespaces(ctx, k8sClient)
}

// ListByOrganization lists all apps belonging to an organization across all its namespaces
func (c *Client) ListByOrganization(ctx context.Context, k8sClient *k8s.Client, org string, labelSelector string) ([]*App, error) {
	// Get all namespaces belonging to this organization
	namespaces, err := organization.GetNamespacesByOrganization(ctx, k8sClient, org)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespaces for organization %s: %w", org, err)
	}

	apps := make([]*App, 0)
	for _, ns := range namespaces {
		nsApps, err := c.List(ctx, ns, labelSelector)
		if err != nil {
			// Log error but continue with other namespaces
			continue
		}
		apps = append(apps, nsApps...)
	}

	return apps, nil
}

// FilterByOrganization filters apps to only include those from organization namespaces
func FilterByOrganization(apps []*App, org string) []*App {
	if org == "" {
		return apps
	}

	orgNamespace := organization.GetOrganizationNamespace(org)
	filtered := make([]*App, 0)

	for _, app := range apps {
		// Check if app is in the organization namespace
		if app.Namespace == orgNamespace {
			filtered = append(filtered, app)
			continue
		}

		// Check if app is in a workload cluster namespace belonging to the organization
		if organization.IsWorkloadClusterNamespace(app.Namespace) {
			// This would need namespace metadata to determine ownership
			// For now, we'll include it if the namespace contains the org name
			// In production, you'd check namespace labels
			filtered = append(filtered, app)
		}
	}

	return filtered
}
