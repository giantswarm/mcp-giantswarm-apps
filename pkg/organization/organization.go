// Package organization provides utilities for working with Giant Swarm organization namespaces
package organization

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

const (
	// OrganizationNamespacePrefix is the standard prefix for organization namespaces
	OrganizationNamespacePrefix = "org-"

	// OrganizationLabel is the label used to identify organization namespaces
	OrganizationLabel = "giantswarm.io/organization"

	// WorkloadClusterNamespacePrefix is the prefix for workload cluster namespaces
	WorkloadClusterNamespacePrefix = "workload-"
)

// IsOrganizationNamespace checks if a namespace is an organization namespace
func IsOrganizationNamespace(namespace string) bool {
	return strings.HasPrefix(namespace, OrganizationNamespacePrefix)
}

// IsWorkloadClusterNamespace checks if a namespace is a workload cluster namespace
func IsWorkloadClusterNamespace(namespace string) bool {
	return strings.HasPrefix(namespace, WorkloadClusterNamespacePrefix)
}

// GetOrganizationFromNamespace extracts the organization name from an organization namespace
// e.g., "org-giantswarm" -> "giantswarm"
func GetOrganizationFromNamespace(namespace string) (string, error) {
	if !IsOrganizationNamespace(namespace) {
		return "", fmt.Errorf("namespace %s is not an organization namespace", namespace)
	}
	return strings.TrimPrefix(namespace, OrganizationNamespacePrefix), nil
}

// GetOrganizationNamespace returns the namespace name for an organization
// e.g., "giantswarm" -> "org-giantswarm"
func GetOrganizationNamespace(organization string) string {
	if strings.HasPrefix(organization, OrganizationNamespacePrefix) {
		return organization // Already has prefix
	}
	return OrganizationNamespacePrefix + organization
}

// ListOrganizationNamespaces returns all organization namespaces in the cluster
func ListOrganizationNamespaces(ctx context.Context, k8sClient kubernetes.Interface) ([]string, error) {
	// First try to list by label
	namespaceList, err := k8sClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
		LabelSelector: labels.Set{
			OrganizationLabel: "true",
		}.String(),
	})

	if err == nil && len(namespaceList.Items) > 0 {
		// Found namespaces with the organization label
		namespaces := make([]string, 0, len(namespaceList.Items))
		for _, ns := range namespaceList.Items {
			namespaces = append(namespaces, ns.Name)
		}
		return namespaces, nil
	}

	// Fallback to listing all namespaces and filtering by prefix
	// This handles both error cases and when no labeled namespaces are found
	allNamespaces, err := k8sClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	orgNamespaces := make([]string, 0)
	for _, ns := range allNamespaces.Items {
		if IsOrganizationNamespace(ns.Name) {
			orgNamespaces = append(orgNamespaces, ns.Name)
		}
	}
	return orgNamespaces, nil
}

// GetNamespacesByOrganization returns all namespaces belonging to an organization
// This includes the organization namespace and any workload cluster namespaces
func GetNamespacesByOrganization(ctx context.Context, k8sClient kubernetes.Interface, organization string) ([]string, error) {
	orgNamespace := GetOrganizationNamespace(organization)

	// List all namespaces
	namespaceList, err := k8sClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	namespaces := make([]string, 0)
	for _, ns := range namespaceList.Items {
		// Include the organization namespace itself
		if ns.Name == orgNamespace {
			namespaces = append(namespaces, ns.Name)
			continue
		}

		// Check if namespace belongs to this organization via labels
		if orgLabel, exists := ns.Labels[OrganizationLabel]; exists && orgLabel == organization {
			namespaces = append(namespaces, ns.Name)
			continue
		}

		// Check for workload cluster namespaces that belong to this organization
		if IsWorkloadClusterNamespace(ns.Name) {
			// Check if the workload cluster belongs to this organization
			if owner, exists := ns.Labels["giantswarm.io/owner"]; exists && owner == organization {
				namespaces = append(namespaces, ns.Name)
			}
		}
	}

	return namespaces, nil
}

// NamespaceInfo contains information about a namespace
type NamespaceInfo struct {
	Name         string
	Type         NamespaceType
	Organization string
	ClusterID    string
	Labels       map[string]string
}

// NamespaceType represents the type of namespace
type NamespaceType string

const (
	// NamespaceTypeOrganization represents an organization namespace
	NamespaceTypeOrganization NamespaceType = "organization"

	// NamespaceTypeWorkloadCluster represents a workload cluster namespace
	NamespaceTypeWorkloadCluster NamespaceType = "workload-cluster"

	// NamespaceTypeSystem represents a system namespace
	NamespaceTypeSystem NamespaceType = "system"

	// NamespaceTypeOther represents any other namespace
	NamespaceTypeOther NamespaceType = "other"
)

// GetNamespaceInfo returns detailed information about a namespace
func GetNamespaceInfo(ctx context.Context, k8sClient kubernetes.Interface, namespace string) (*NamespaceInfo, error) {
	ns, err := k8sClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace %s: %w", namespace, err)
	}

	info := &NamespaceInfo{
		Name:   namespace,
		Labels: ns.Labels,
	}

	// Determine namespace type
	if IsOrganizationNamespace(namespace) {
		info.Type = NamespaceTypeOrganization
		info.Organization, _ = GetOrganizationFromNamespace(namespace)
	} else if IsWorkloadClusterNamespace(namespace) {
		info.Type = NamespaceTypeWorkloadCluster
		if clusterID, exists := ns.Labels["giantswarm.io/cluster"]; exists {
			info.ClusterID = clusterID
		}
		if owner, exists := ns.Labels["giantswarm.io/owner"]; exists {
			info.Organization = owner
		}
	} else if isSystemNamespace(namespace) {
		info.Type = NamespaceTypeSystem
	} else {
		info.Type = NamespaceTypeOther
		// Check if it has an organization label
		if org, exists := ns.Labels[OrganizationLabel]; exists {
			info.Organization = org
		}
	}

	return info, nil
}

// isSystemNamespace checks if a namespace is a system namespace
func isSystemNamespace(namespace string) bool {
	systemNamespaces := []string{
		"kube-system",
		"kube-public",
		"kube-node-lease",
		"default",
		"giantswarm",
		"flux-system",
		"monitoring",
	}

	for _, sysNs := range systemNamespaces {
		if namespace == sysNs {
			return true
		}
	}

	return false
}

// ValidateNamespaceAccess validates if the current context has access to a namespace
// This is a placeholder for RBAC validation
func ValidateNamespaceAccess(ctx context.Context, k8sClient kubernetes.Interface, namespace string) error {
	// Try to get the namespace - if we can't, we don't have access
	_, err := k8sClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("no access to namespace %s: %w", namespace, err)
	}
	return nil
}

// GetOrganizationFromCurrentContext attempts to determine the organization from the current context
// This is a placeholder that would need to be implemented based on your authentication setup
func GetOrganizationFromCurrentContext(ctx context.Context, k8sClient kubernetes.Interface) (string, error) {
	// This would typically check:
	// 1. Service account namespace if running in-cluster
	// 2. RBAC permissions to determine organization
	// 3. Context name patterns
	// 4. Default to a configured organization

	// For now, return empty to indicate no default organization
	return "", nil
}
