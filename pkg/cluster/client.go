package cluster

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/mcp-giantswarm-apps/internal/k8s"
	"github.com/giantswarm/mcp-giantswarm-apps/pkg/app"
	"github.com/giantswarm/mcp-giantswarm-apps/pkg/organization"
)

// ClusterGVR is the GroupVersionResource for CAPI Cluster resources
var ClusterGVR = schema.GroupVersionResource{
	Group:    "cluster.x-k8s.io",
	Version:  "v1beta1",
	Resource: "clusters",
}

// Client provides operations for CAPI Cluster resources
type Client struct {
	dynamicClient dynamic.Interface
	k8sClient     kubernetes.Interface
	appClient     *app.Client
}

// NewClient creates a new cluster client
func NewClient(dynamicClient *k8s.DynamicClient, k8sClient kubernetes.Interface, appClient *app.Client) *Client {
	return &Client{
		dynamicClient: dynamicClient.GetInterface(),
		k8sClient:     k8sClient,
		appClient:     appClient,
	}
}

// List lists clusters in a namespace or across all namespaces
func (c *Client) List(ctx context.Context, namespace string, labelSelector string) ([]*Cluster, error) {
	listOptions := metav1.ListOptions{}
	if labelSelector != "" {
		listOptions.LabelSelector = labelSelector
	}

	var list *unstructured.UnstructuredList
	var err error

	if namespace == "" {
		// List across all namespaces
		list, err = c.dynamicClient.Resource(ClusterGVR).List(ctx, listOptions)
	} else {
		// List in specific namespace
		list, err = c.dynamicClient.Resource(ClusterGVR).Namespace(namespace).List(ctx, listOptions)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	clusters := make([]*Cluster, 0, len(list.Items))
	for _, item := range list.Items {
		cluster, err := NewClusterFromUnstructured(&item)
		if err != nil {
			continue // Skip invalid clusters
		}
		clusters = append(clusters, cluster)
	}

	return clusters, nil
}

// Get retrieves a specific cluster
func (c *Client) Get(ctx context.Context, namespace, name string) (*Cluster, error) {
	obj, err := c.dynamicClient.Resource(ClusterGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster %s/%s: %w", namespace, name, err)
	}

	return NewClusterFromUnstructured(obj)
}

// ListByOrganization lists all clusters belonging to an organization
func (c *Client) ListByOrganization(ctx context.Context, org string) ([]*Cluster, error) {
	// First, get all namespaces for the organization
	namespaces, err := organization.GetNamespacesByOrganization(ctx, c.k8sClient, org)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization namespaces: %w", err)
	}

	allClusters := make([]*Cluster, 0)

	// Look for clusters in each namespace
	for _, ns := range namespaces {
		clusters, err := c.List(ctx, ns, "")
		if err != nil {
			continue // Skip namespaces with errors
		}

		// Also filter by organization label
		for _, cluster := range clusters {
			if cluster.GetOrganization() == org {
				allClusters = append(allClusters, cluster)
			}
		}
	}

	return allClusters, nil
}

// GetKubeconfig retrieves the kubeconfig for a workload cluster
func (c *Client) GetKubeconfig(ctx context.Context, cluster *Cluster) ([]byte, error) {
	// Look for kubeconfig secret in the same namespace as the cluster
	// The secret name follows the pattern: {cluster-name}-kubeconfig
	secretName := fmt.Sprintf("%s-kubeconfig", cluster.Name)

	secret, err := c.k8sClient.CoreV1().Secrets(cluster.Namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig secret: %w", err)
	}

	// The kubeconfig is usually stored in the "value" key
	if kubeconfig, ok := secret.Data["value"]; ok {
		return kubeconfig, nil
	}

	// Fallback to "kubeconfig" key
	if kubeconfig, ok := secret.Data["kubeconfig"]; ok {
		return kubeconfig, nil
	}

	return nil, fmt.Errorf("kubeconfig not found in secret")
}

// ListApps lists all apps deployed to a specific cluster
func (c *Client) ListApps(ctx context.Context, cluster *Cluster) ([]*app.App, error) {
	// Apps for a workload cluster are typically in the workload cluster namespace
	workloadNamespace := fmt.Sprintf("workload-%s", cluster.Name)

	// List apps in the workload namespace
	apps, err := c.appClient.List(ctx, workloadNamespace, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list apps in cluster %s: %w", cluster.Name, err)
	}

	// Also check for apps that target this cluster via kubeconfig
	// This requires checking apps in the organization namespace
	if cluster.GetOrganization() != "" {
		orgNamespace := organization.GetOrganizationNamespace(cluster.GetOrganization())
		orgApps, err := c.appClient.List(ctx, orgNamespace, "")
		if err == nil {
			// Filter apps that target this cluster
			for _, app := range orgApps {
				// Check if app targets this cluster
				// This would need to be implemented based on how apps reference target clusters
				if !app.Spec.KubeConfig.InCluster {
					// TODO: Check if app references this cluster's kubeconfig
					apps = append(apps, app)
				}
			}
		}
	}

	return apps, nil
}

// IsWorkloadCluster checks if this is a workload cluster (not the management cluster)
func (c *Client) IsWorkloadCluster(cluster *Cluster) bool {
	// Management clusters typically have specific labels or are in specific namespaces
	if cluster.Namespace == "default" || cluster.Namespace == "giantswarm" {
		return false
	}

	// Check for workload cluster labels
	if clusterType, ok := cluster.Labels["giantswarm.io/cluster-type"]; ok {
		return clusterType == "workload"
	}

	// By default, assume clusters in organization namespaces are workload clusters
	return true
}

// FilterByProvider filters clusters by infrastructure provider
func FilterByProvider(clusters []*Cluster, provider string) []*Cluster {
	if provider == "" {
		return clusters
	}

	filtered := make([]*Cluster, 0)
	for _, cluster := range clusters {
		if cluster.GetProvider() == provider {
			filtered = append(filtered, cluster)
		}
	}
	return filtered
}

// FilterByStatus filters clusters by their status
func FilterByStatus(clusters []*Cluster, ready bool) []*Cluster {
	filtered := make([]*Cluster, 0)
	for _, cluster := range clusters {
		if cluster.IsReady() == ready {
			filtered = append(filtered, cluster)
		}
	}
	return filtered
}

// GetClusterNamespace returns the expected namespace for a workload cluster
func GetClusterNamespace(clusterName string) string {
	return fmt.Sprintf("workload-%s", clusterName)
}

// CreateClusterLabels creates standard labels for a cluster
func CreateClusterLabels(org string, clusterType string) map[string]string {
	labels := make(map[string]string)

	if org != "" {
		labels["giantswarm.io/organization"] = org
	}

	if clusterType != "" {
		labels["giantswarm.io/cluster-type"] = clusterType
	}

	return labels
}
