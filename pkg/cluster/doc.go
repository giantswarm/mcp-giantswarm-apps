// Package cluster provides utilities for working with Cluster API (CAPI) resources
// and workload cluster management in Giant Swarm.
//
// # Overview
//
// This package integrates with Cluster API to provide:
//   - Discovery and listing of workload clusters
//   - Kubeconfig management for workload clusters
//   - Cross-cluster app deployment support
//   - Cluster lifecycle event handling
//
// # CAPI Resources
//
// The package works with standard CAPI resources:
//   - Cluster (cluster.x-k8s.io/v1beta1)
//   - Machine, MachineSet, MachineDeployment
//   - Infrastructure provider resources (AWSCluster, AzureCluster, etc.)
//
// # Usage
//
// List all clusters:
//
//	client := cluster.NewClient(dynamicClient, k8sClient, appClient)
//	clusters, err := client.List(ctx, "", "")
//
// Get clusters for an organization:
//
//	clusters, err := client.ListByOrganization(ctx, "giantswarm")
//
// Get kubeconfig for a workload cluster:
//
//	kubeconfig, err := client.GetKubeconfig(ctx, cluster)
//
// List apps in a workload cluster:
//
//	apps, err := client.ListApps(ctx, cluster)
//
// # Cluster Namespacing
//
// Workload clusters follow these conventions:
//   - Cluster resources are in organization namespaces (org-*)
//   - Workload cluster apps are in workload namespaces (workload-{cluster-name})
//   - Kubeconfig secrets are named {cluster-name}-kubeconfig
//
// # Labels
//
// Standard labels used:
//   - giantswarm.io/organization: Organization that owns the cluster
//   - giantswarm.io/cluster-type: Type of cluster (workload, management)
//   - cluster.x-k8s.io/provider: Infrastructure provider
package cluster
