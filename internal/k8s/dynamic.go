package k8s

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// Giant Swarm CRD Group Version Resources
var (
	AppGVR = schema.GroupVersionResource{
		Group:    "application.giantswarm.io",
		Version:  "v1alpha1",
		Resource: "apps",
	}

	CatalogGVR = schema.GroupVersionResource{
		Group:    "application.giantswarm.io",
		Version:  "v1alpha1",
		Resource: "catalogs",
	}

	AppCatalogEntryGVR = schema.GroupVersionResource{
		Group:    "application.giantswarm.io",
		Version:  "v1alpha1",
		Resource: "appcatalogentries",
	}

	ReleaseGVR = schema.GroupVersionResource{
		Group:    "release.giantswarm.io",
		Version:  "v1alpha1",
		Resource: "releases",
	}
)

// DynamicClient wraps the dynamic client for Giant Swarm resources
type DynamicClient struct {
	client dynamic.Interface
}

// NewDynamicClient creates a new dynamic client
func NewDynamicClient(client *Client) (*DynamicClient, error) {
	dynamicClient, err := dynamic.NewForConfig(client.RestConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &DynamicClient{
		client: dynamicClient,
	}, nil
}

// Apps returns the interface for working with App resources
func (d *DynamicClient) Apps(namespace string) dynamic.ResourceInterface {
	if namespace == "" {
		return d.client.Resource(AppGVR)
	}
	return d.client.Resource(AppGVR).Namespace(namespace)
}

// Catalogs returns the interface for working with Catalog resources
func (d *DynamicClient) Catalogs(namespace string) dynamic.ResourceInterface {
	if namespace == "" {
		return d.client.Resource(CatalogGVR)
	}
	return d.client.Resource(CatalogGVR).Namespace(namespace)
}

// AppCatalogEntries returns the interface for working with AppCatalogEntry resources
func (d *DynamicClient) AppCatalogEntries(namespace string) dynamic.ResourceInterface {
	if namespace == "" {
		return d.client.Resource(AppCatalogEntryGVR)
	}
	return d.client.Resource(AppCatalogEntryGVR).Namespace(namespace)
}

// Releases returns the interface for working with Release resources
func (d *DynamicClient) Releases(namespace string) dynamic.ResourceInterface {
	if namespace == "" {
		return d.client.Resource(ReleaseGVR)
	}
	return d.client.Resource(ReleaseGVR).Namespace(namespace)
}

// CheckCRDsExist verifies that Giant Swarm CRDs are installed
func (d *DynamicClient) CheckCRDsExist(ctx context.Context, client *Client) error {
	apiResourceList, err := client.Discovery().ServerResourcesForGroupVersion("application.giantswarm.io/v1alpha1")
	if err != nil {
		return fmt.Errorf("Giant Swarm CRDs not found: %w", err)
	}

	requiredResources := map[string]bool{
		"apps":              false,
		"catalogs":          false,
		"appcatalogentries": false,
	}

	for _, resource := range apiResourceList.APIResources {
		if _, ok := requiredResources[resource.Name]; ok {
			requiredResources[resource.Name] = true
		}
	}

	for resource, found := range requiredResources {
		if !found {
			return fmt.Errorf("required CRD %s.application.giantswarm.io not found", resource)
		}
	}

	return nil
} 