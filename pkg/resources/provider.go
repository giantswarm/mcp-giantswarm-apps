package resources

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/mcp-giantswarm-apps/internal/k8s"
	"github.com/giantswarm/mcp-giantswarm-apps/pkg/app"
	"github.com/giantswarm/mcp-giantswarm-apps/pkg/appcatalogentry"
	"github.com/giantswarm/mcp-giantswarm-apps/pkg/catalog"
	"github.com/giantswarm/mcp-giantswarm-apps/pkg/config"
)

// Provider handles MCP resource operations
type Provider struct {
	k8sClient             *k8s.Client
	dynamicClient         *k8s.DynamicClient
	appClient             *app.Client
	catalogClient         *catalog.Client
	appCatalogEntryClient *appcatalogentry.Client
	configClient          *config.Client
}

// NewProvider creates a new resource provider
func NewProvider(k8sClient *k8s.Client, dynamicClient *k8s.DynamicClient) *Provider {
	return &Provider{
		k8sClient:             k8sClient,
		dynamicClient:         dynamicClient,
		appClient:             app.NewClient(dynamicClient),
		catalogClient:         catalog.NewClient(dynamicClient),
		appCatalogEntryClient: appcatalogentry.NewClient(dynamicClient),
		configClient:          config.NewClient(k8sClient),
	}
}

// ListResources returns a list of available resources
func (p *Provider) ListResources(ctx context.Context) ([]ResourceMetadata, error) {
	var resources []ResourceMetadata

	// List all apps across namespaces
	apps, err := p.appClient.List(ctx, "", labels.Everything().String())
	if err != nil {
		return nil, fmt.Errorf("failed to list apps: %w", err)
	}

	for _, app := range apps {
		// Add app resource
		resources = append(resources, ResourceMetadata{
			URI:         fmt.Sprintf("app://%s/%s", app.Namespace, app.Name),
			Name:        fmt.Sprintf("App: %s/%s", app.Namespace, app.Name),
			Description: fmt.Sprintf("Giant Swarm app %s in namespace %s", app.Name, app.Namespace),
			MimeType:    "application/json",
		})

		// Add config resource if app has configuration
		if app.Spec.Config != nil || app.Spec.UserConfig != nil {
			resources = append(resources, ResourceMetadata{
				URI:         fmt.Sprintf("config://%s/%s/values", app.Namespace, app.Name),
				Name:        fmt.Sprintf("Config: %s/%s", app.Namespace, app.Name),
				Description: fmt.Sprintf("Configuration values for app %s", app.Name),
				MimeType:    "application/json",
			})
		}
	}

	// List all catalogs
	catalogs, err := p.catalogClient.List(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list catalogs: %w", err)
	}

	for _, catalog := range catalogs {
		resources = append(resources, ResourceMetadata{
			URI:         fmt.Sprintf("catalog://%s", catalog.Name),
			Name:        fmt.Sprintf("Catalog: %s", catalog.Name),
			Description: fmt.Sprintf("Giant Swarm app catalog %s", catalog.Name),
			MimeType:    "application/json",
		})
	}

	// List app catalog entries for schema and changelog resources
	entries, err := p.appCatalogEntryClient.List(ctx, labels.Everything().String())
	if err != nil {
		return nil, fmt.Errorf("failed to list app catalog entries: %w", err)
	}

	// Group entries by app name and catalog
	appMap := make(map[string]*appcatalogentry.AppCatalogEntry)
	for _, entry := range entries {
		// Parse the name to get catalog and app name
		parts := strings.Split(entry.Name, "-")
		if len(parts) >= 2 {
			catalogName := parts[0]
			appName := strings.Join(parts[1:len(parts)-1], "-")

			// Add schema resource for each version
			if entry.Spec.Chart.Version != "" {
				resources = append(resources, ResourceMetadata{
					URI:         fmt.Sprintf("schema://%s/%s/%s", catalogName, appName, entry.Spec.Chart.Version),
					Name:        fmt.Sprintf("Schema: %s/%s@%s", catalogName, appName, entry.Spec.Chart.Version),
					Description: fmt.Sprintf("Configuration schema for %s version %s", appName, entry.Spec.Chart.Version),
					MimeType:    "application/json",
				})
			}

			// Keep track of unique apps for changelog
			key := fmt.Sprintf("%s/%s", catalogName, appName)
			if _, exists := appMap[key]; !exists {
				appMap[key] = entry
				resources = append(resources, ResourceMetadata{
					URI:         fmt.Sprintf("changelog://%s/%s", catalogName, appName),
					Name:        fmt.Sprintf("Changelog: %s/%s", catalogName, appName),
					Description: fmt.Sprintf("Version history for %s", appName),
					MimeType:    "application/json",
				})
			}
		}
	}

	return resources, nil
}

// GetResource fetches the content of a specific resource
func (p *Provider) GetResource(ctx context.Context, uri string) (interface{}, error) {
	resourceURI, err := ParseResourceURI(uri)
	if err != nil {
		return nil, err
	}

	switch resourceURI.Type {
	case ResourceTypeApp:
		return p.getAppResource(ctx, resourceURI)
	case ResourceTypeCatalog:
		return p.getCatalogResource(ctx, resourceURI)
	case ResourceTypeConfig:
		return p.getConfigResource(ctx, resourceURI)
	case ResourceTypeSchema:
		return p.getSchemaResource(ctx, resourceURI)
	case ResourceTypeChangelog:
		return p.getChangelogResource(ctx, resourceURI)
	default:
		return nil, fmt.Errorf("unknown resource type: %s", resourceURI.Type)
	}
}

func (p *Provider) getAppResource(ctx context.Context, uri *ResourceURI) (*AppResourceContent, error) {
	app, err := p.appClient.Get(ctx, uri.Namespace, uri.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get app: %w", err)
	}

	content := &AppResourceContent{
		Name:      uri.Name,
		Namespace: uri.Namespace,
		Version:   app.Spec.Version,
		Catalog:   app.Spec.Catalog,
		Status:    app.Status.Release.Status,
		Config:    make(map[string]interface{}),
		Metadata:  make(map[string]string),
	}

	// Get labels and timestamps from unstructured representation
	unstructuredApp := app.ToUnstructured()
	metadata := unstructuredApp.Object["metadata"].(map[string]interface{})

	// Extract labels
	if labels, ok := metadata["labels"].(map[string]interface{}); ok {
		for k, v := range labels {
			content.Metadata[k] = fmt.Sprintf("%v", v)
		}
	}

	// Extract configuration
	if app.Spec.Config != nil && app.Spec.Config.ConfigMap != nil {
		cm, err := p.configClient.GetConfigMap(ctx, uri.Namespace, app.Spec.Config.ConfigMap.Name)
		if err == nil && cm.Data != nil {
			for k, v := range cm.Data {
				content.Config[k] = v
			}
		}
	}

	// Extract timestamp
	if metadata["creationTimestamp"] != nil {
		content.LastUpdated = metadata["creationTimestamp"].(string)
	}

	return content, nil
}

func (p *Provider) getCatalogResource(ctx context.Context, uri *ResourceURI) (*CatalogResourceContent, error) {
	// Catalogs are cluster-scoped
	catalog, err := p.catalogClient.Get(ctx, "", uri.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get catalog: %w", err)
	}

	content := &CatalogResourceContent{
		Name:        uri.Name,
		Title:       catalog.Spec.Title,
		Description: catalog.Spec.Description,
		Type:        catalog.CatalogType(),
		Visibility:  catalog.CatalogVisibility(),
	}

	// Extract repository URL
	if catalog.Spec.Storage.Type == "helm" {
		content.URL = catalog.Spec.Storage.URL
	}

	// Count apps in this catalog
	entries, err := p.appCatalogEntryClient.List(ctx, labels.Everything().String())
	if err == nil {
		count := 0
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name, uri.Name+"-") {
				count++
			}
		}
		content.AppCount = count
	}

	// Get timestamp
	unstructuredCatalog := catalog.ToUnstructured()
	metadata := unstructuredCatalog.Object["metadata"].(map[string]interface{})
	if metadata["creationTimestamp"] != nil {
		content.LastUpdated = metadata["creationTimestamp"].(string)
	}

	return content, nil
}

func (p *Provider) getConfigResource(ctx context.Context, uri *ResourceURI) (*ConfigResourceContent, error) {
	// Get the app to find its config references
	app, err := p.appClient.Get(ctx, uri.Namespace, uri.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get app: %w", err)
	}

	content := &ConfigResourceContent{
		AppName:   uri.Name,
		Namespace: uri.Namespace,
		Values:    make(map[string]interface{}),
	}

	// Get user config
	if app.Spec.UserConfig != nil {
		if app.Spec.UserConfig.ConfigMap != nil {
			cm, err := p.configClient.GetConfigMap(ctx, uri.Namespace, app.Spec.UserConfig.ConfigMap.Name)
			if err == nil && cm.Data != nil {
				for k, v := range cm.Data {
					// Try to parse as YAML/JSON
					var parsed interface{}
					if err := yaml.Unmarshal([]byte(v), &parsed); err == nil {
						content.Values[k] = parsed
					} else {
						content.Values[k] = v
					}
				}
				content.Source = "configmap"
			}
		}
		if app.Spec.UserConfig.Secret != nil {
			secret, err := p.configClient.GetSecret(ctx, uri.Namespace, app.Spec.UserConfig.Secret.Name)
			if err == nil && secret.Data != nil {
				for k, v := range secret.Data {
					// Try to parse as YAML/JSON
					var parsed interface{}
					if err := yaml.Unmarshal([]byte(v), &parsed); err == nil {
						content.Values[k] = parsed
					} else {
						content.Values[k] = v
					}
				}
				content.Source = "secret"
			}
		}
	}

	// Get last update time
	unstructuredApp := app.ToUnstructured()
	metadata := unstructuredApp.Object["metadata"].(map[string]interface{})
	if metadata["creationTimestamp"] != nil {
		content.LastUpdate = metadata["creationTimestamp"].(string)
	}

	return content, nil
}

func (p *Provider) getSchemaResource(ctx context.Context, uri *ResourceURI) (*SchemaResourceContent, error) {
	// Find the app catalog entry for the specific version
	entries, err := p.appCatalogEntryClient.List(ctx, labels.Everything().String())
	if err != nil {
		return nil, fmt.Errorf("failed to list app catalog entries: %w", err)
	}

	var targetEntry *appcatalogentry.AppCatalogEntry
	searchName := fmt.Sprintf("%s-%s-%s", uri.Catalog, uri.Name, uri.Version)

	for _, entry := range entries {
		if entry.Name == searchName {
			targetEntry = entry
			break
		}
	}

	if targetEntry == nil {
		return nil, fmt.Errorf("app catalog entry not found for %s/%s@%s", uri.Catalog, uri.Name, uri.Version)
	}

	content := &SchemaResourceContent{
		AppName: uri.Name,
		Version: uri.Version,
		Schema:  make(map[string]interface{}),
	}

	// For now, generate a basic schema from the chart metadata
	// In a real implementation, this would fetch the actual values schema
	content.Schema = map[string]interface{}{
		"type":        "object",
		"title":       targetEntry.Spec.Chart.Name,
		"description": targetEntry.Spec.Chart.Description,
		"properties": map[string]interface{}{
			"replicaCount": map[string]interface{}{
				"type":        "integer",
				"description": "Number of replicas",
				"default":     1,
			},
			"image": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "Image repository",
					},
					"tag": map[string]interface{}{
						"type":        "string",
						"description": "Image tag",
					},
				},
			},
		},
	}

	return content, nil
}

func (p *Provider) getChangelogResource(ctx context.Context, uri *ResourceURI) (*ChangelogResourceContent, error) {
	// List all versions of this app
	entries, err := p.appCatalogEntryClient.List(ctx, labels.Everything().String())
	if err != nil {
		return nil, fmt.Errorf("failed to list app catalog entries: %w", err)
	}

	content := &ChangelogResourceContent{
		AppName: uri.Name,
		Catalog: uri.Catalog,
		Entries: []ChangelogEntry{},
	}

	// Filter entries for this app
	appPrefix := fmt.Sprintf("%s-%s-", uri.Catalog, uri.Name)
	var versions []struct {
		version     string
		date        string
		description string
		entry       *appcatalogentry.AppCatalogEntry
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name, appPrefix) {
			item := struct {
				version     string
				date        string
				description string
				entry       *appcatalogentry.AppCatalogEntry
			}{
				version: entry.Spec.Chart.Version,
				entry:   entry,
			}

			// Extract date
			if entry.Spec.DateCreated != nil {
				item.date = entry.Spec.DateCreated.Format("2006-01-02")
			}

			// Extract description
			if entry.Spec.Chart.Description != "" {
				item.description = entry.Spec.Chart.Description
			}

			versions = append(versions, item)
		}
	}

	// Sort versions by date (newest first)
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].date > versions[j].date
	})

	// Convert to changelog entries
	for i, v := range versions {
		entry := ChangelogEntry{
			Version:     v.version,
			Date:        v.date,
			Description: v.description,
		}

		// Check if it's a breaking change (major version bump)
		if i < len(versions)-1 {
			nextVersion := versions[i+1].version
			if isBreakingChange(v.version, nextVersion) {
				entry.Breaking = true
			}
		}

		content.Entries = append(content.Entries, entry)
	}

	return content, nil
}

// isBreakingChange checks if version change is breaking (major version bump)
func isBreakingChange(newVersion, oldVersion string) bool {
	// Simple check: if major version changed
	newParts := strings.Split(newVersion, ".")
	oldParts := strings.Split(oldVersion, ".")

	if len(newParts) > 0 && len(oldParts) > 0 {
		newMajor := strings.TrimPrefix(newParts[0], "v")
		oldMajor := strings.TrimPrefix(oldParts[0], "v")
		return newMajor != oldMajor
	}

	return false
}
