// Package organization provides utilities for working with Giant Swarm organization namespaces
// and multi-tenancy support.
//
// # Overview
//
// Giant Swarm uses an organization-based multi-tenancy model where each organization (customer)
// has dedicated namespaces following specific naming conventions:
//   - Organization namespaces: org-{organization-name} (e.g., org-giantswarm)
//   - Workload cluster namespaces: workload-{cluster-name} (e.g., workload-prod-cluster)
//
// This package provides functions to:
//   - Identify and validate organization namespaces
//   - List and filter namespaces by organization
//   - Get detailed namespace information including organization context
//   - Validate namespace access permissions
//
// # Usage
//
// Basic namespace identification:
//
//	if organization.IsOrganizationNamespace("org-giantswarm") {
//	    orgName, _ := organization.GetOrganizationFromNamespace("org-giantswarm")
//	    // orgName = "giantswarm"
//	}
//
// List all organizations:
//
//	orgs, err := organization.ListOrganizationNamespaces(ctx, k8sClient)
//	// Returns: ["org-giantswarm", "org-adidas", ...]
//
// Get all namespaces for an organization:
//
//	namespaces, err := organization.GetNamespacesByOrganization(ctx, k8sClient, "giantswarm")
//	// Returns: ["org-giantswarm", "workload-cluster1", ...]
//
// # Namespace Types
//
// The package recognizes four types of namespaces:
//   - Organization: Primary namespace for an organization (org-*)
//   - WorkloadCluster: Namespace for a workload cluster (workload-*)
//   - System: Core Kubernetes namespaces (kube-system, default, etc.)
//   - Other: Any other namespace
//
// # Labels
//
// The package uses these standard Giant Swarm labels:
//   - giantswarm.io/organization: Marks organization namespaces
//   - giantswarm.io/owner: Identifies the organization that owns a resource
//   - giantswarm.io/cluster: Identifies the cluster ID for workload namespaces
package organization
