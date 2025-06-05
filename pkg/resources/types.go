package resources

import (
	"fmt"
	"strings"
)

// ResourceType represents the type of resource
type ResourceType string

const (
	ResourceTypeApp       ResourceType = "app"
	ResourceTypeCatalog   ResourceType = "catalog"
	ResourceTypeConfig    ResourceType = "config"
	ResourceTypeSchema    ResourceType = "schema"
	ResourceTypeChangelog ResourceType = "changelog"
)

// ResourceURI represents a parsed resource URI
type ResourceURI struct {
	Type      ResourceType
	Namespace string
	Name      string
	Catalog   string
	Version   string
	SubPath   string
}

// ParseResourceURI parses a resource URI into its components
func ParseResourceURI(uri string) (*ResourceURI, error) {
	// Remove the scheme part
	parts := strings.SplitN(uri, "://", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid resource URI format: %s", uri)
	}

	scheme := parts[0]
	path := parts[1]

	var resourceType ResourceType
	switch scheme {
	case "app":
		resourceType = ResourceTypeApp
	case "catalog":
		resourceType = ResourceTypeCatalog
	case "config":
		resourceType = ResourceTypeConfig
	case "schema":
		resourceType = ResourceTypeSchema
	case "changelog":
		resourceType = ResourceTypeChangelog
	default:
		return nil, fmt.Errorf("unknown resource type: %s", scheme)
	}

	result := &ResourceURI{
		Type: resourceType,
	}

	// Parse the path based on resource type
	pathParts := strings.Split(path, "/")

	switch resourceType {
	case ResourceTypeApp:
		// app://{namespace}/{name}
		if len(pathParts) != 2 {
			return nil, fmt.Errorf("invalid app resource path: expected namespace/name")
		}
		result.Namespace = pathParts[0]
		result.Name = pathParts[1]

	case ResourceTypeCatalog:
		// catalog://{name}
		if len(pathParts) != 1 {
			return nil, fmt.Errorf("invalid catalog resource path: expected name")
		}
		result.Name = pathParts[0]

	case ResourceTypeConfig:
		// config://{namespace}/{app}/values
		if len(pathParts) < 2 {
			return nil, fmt.Errorf("invalid config resource path: expected namespace/app/...")
		}
		result.Namespace = pathParts[0]
		result.Name = pathParts[1]
		if len(pathParts) > 2 {
			result.SubPath = strings.Join(pathParts[2:], "/")
		}

	case ResourceTypeSchema:
		// schema://{catalog}/{app}/{version}
		if len(pathParts) != 3 {
			return nil, fmt.Errorf("invalid schema resource path: expected catalog/app/version")
		}
		result.Catalog = pathParts[0]
		result.Name = pathParts[1]
		result.Version = pathParts[2]

	case ResourceTypeChangelog:
		// changelog://{catalog}/{app}
		if len(pathParts) != 2 {
			return nil, fmt.Errorf("invalid changelog resource path: expected catalog/app")
		}
		result.Catalog = pathParts[0]
		result.Name = pathParts[1]
	}

	return result, nil
}

// String returns the URI string representation
func (r *ResourceURI) String() string {
	switch r.Type {
	case ResourceTypeApp:
		return fmt.Sprintf("app://%s/%s", r.Namespace, r.Name)
	case ResourceTypeCatalog:
		return fmt.Sprintf("catalog://%s", r.Name)
	case ResourceTypeConfig:
		if r.SubPath != "" {
			return fmt.Sprintf("config://%s/%s/%s", r.Namespace, r.Name, r.SubPath)
		}
		return fmt.Sprintf("config://%s/%s/values", r.Namespace, r.Name)
	case ResourceTypeSchema:
		return fmt.Sprintf("schema://%s/%s/%s", r.Catalog, r.Name, r.Version)
	case ResourceTypeChangelog:
		return fmt.Sprintf("changelog://%s/%s", r.Catalog, r.Name)
	default:
		return ""
	}
}

// ResourceMetadata contains metadata about a resource
type ResourceMetadata struct {
	URI         string
	Name        string
	Description string
	MimeType    string
}

// AppResourceContent represents the content of an app resource
type AppResourceContent struct {
	Name        string                 `json:"name"`
	Namespace   string                 `json:"namespace"`
	Version     string                 `json:"version"`
	Catalog     string                 `json:"catalog"`
	Status      string                 `json:"status"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Metadata    map[string]string      `json:"metadata,omitempty"`
	LastUpdated string                 `json:"lastUpdated,omitempty"`
}

// CatalogResourceContent represents the content of a catalog resource
type CatalogResourceContent struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Visibility  string `json:"visibility"`
	URL         string `json:"url"`
	AppCount    int    `json:"appCount"`
	LastUpdated string `json:"lastUpdated,omitempty"`
}

// ConfigResourceContent represents the content of a config resource
type ConfigResourceContent struct {
	AppName    string                 `json:"appName"`
	Namespace  string                 `json:"namespace"`
	Values     map[string]interface{} `json:"values"`
	Source     string                 `json:"source"` // configmap or secret
	LastUpdate string                 `json:"lastUpdate,omitempty"`
}

// SchemaResourceContent represents the content of a schema resource
type SchemaResourceContent struct {
	AppName     string                 `json:"appName"`
	Version     string                 `json:"version"`
	Schema      map[string]interface{} `json:"schema"`
	Required    []string               `json:"required,omitempty"`
	Definitions map[string]interface{} `json:"definitions,omitempty"`
}

// ChangelogEntry represents a single changelog entry
type ChangelogEntry struct {
	Version     string   `json:"version"`
	Date        string   `json:"date"`
	Description string   `json:"description"`
	Changes     []string `json:"changes,omitempty"`
	Breaking    bool     `json:"breaking,omitempty"`
}

// ChangelogResourceContent represents the content of a changelog resource
type ChangelogResourceContent struct {
	AppName string           `json:"appName"`
	Catalog string           `json:"catalog"`
	Entries []ChangelogEntry `json:"entries"`
}
