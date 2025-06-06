# Multi-namespace and Organization Support

This document describes the multi-namespace and organization support in the MCP Giant Swarm Apps server.

## Overview

Giant Swarm uses an organization-based multi-tenancy model where each organization (customer) has:
- A dedicated organization namespace (e.g., `org-giantswarm`, `org-adidas`)
- Optional workload cluster namespaces (e.g., `workload-prod-cluster`)
- RBAC policies controlling access to resources

## Namespace Types

### Organization Namespaces
- **Pattern**: `org-{organization-name}`
- **Purpose**: Primary namespace for organization resources
- **Example**: `org-giantswarm`, `org-my-company`

### Workload Cluster Namespaces  
- **Pattern**: `workload-{cluster-name}`
- **Purpose**: Resources for specific workload clusters
- **Labels**: 
  - `giantswarm.io/owner`: Organization that owns the cluster
  - `giantswarm.io/cluster`: Cluster identifier

### System Namespaces
- **Examples**: `kube-system`, `default`, `giantswarm`, `flux-system`
- **Purpose**: Core Kubernetes and platform services

## Organization Tools

### organization.list
List all organizations in the cluster.

```bash
# List organizations
mcp organization.list

# List with detailed namespace information
mcp organization.list --detailed
```

### organization.namespaces
List all namespaces belonging to an organization.

```bash
# List namespaces for an organization
mcp organization.namespaces --organization giantswarm

# Include namespace details
mcp organization.namespaces --organization giantswarm --include-details
```

### organization.info
Get detailed information about a namespace and its organization context.

```bash
# Get namespace information
mcp organization.info --namespace org-giantswarm
```

### organization.validate-access
Validate access to a namespace or organization.

```bash
# Validate namespace access
mcp organization.validate-access --namespace org-giantswarm

# Validate organization access
mcp organization.validate-access --organization giantswarm
```

## Organization-Aware App Management

### Listing Apps

```bash
# List apps from a specific organization
mcp app.list --organization giantswarm

# List apps from all organization namespaces
mcp app.list --all-orgs

# Include workload cluster namespaces
mcp app.list --organization giantswarm --include-workload-clusters
```

### Creating Apps

```bash
# Create app in organization namespace
mcp app.create \
  --name my-app \
  --namespace org-giantswarm \
  --catalog giantswarm \
  --app nginx-ingress-controller \
  --version 2.0.0
```

## Organization-Aware Catalog Management

### Listing Catalogs

```bash
# List catalogs from a specific organization
mcp catalog.list --organization giantswarm

# List catalogs from all organization namespaces
mcp catalog.list --all-orgs
```

## API Reference

### Organization Package Functions

#### IsOrganizationNamespace(namespace string) bool
Checks if a namespace is an organization namespace.

#### IsWorkloadClusterNamespace(namespace string) bool
Checks if a namespace is a workload cluster namespace.

#### GetOrganizationFromNamespace(namespace string) (string, error)
Extracts the organization name from an organization namespace.

#### GetOrganizationNamespace(organization string) string
Returns the namespace name for an organization.

#### ListOrganizationNamespaces(ctx, k8sClient) ([]string, error)
Returns all organization namespaces in the cluster.

#### GetNamespacesByOrganization(ctx, k8sClient, organization) ([]string, error)
Returns all namespaces belonging to an organization.

#### GetNamespaceInfo(ctx, k8sClient, namespace) (*NamespaceInfo, error)
Returns detailed information about a namespace.

#### ValidateNamespaceAccess(ctx, k8sClient, namespace) error
Validates if the current context has access to a namespace.

## Best Practices

1. **Always specify organization or namespace** when working with resources to ensure proper isolation.

2. **Use organization parameter** instead of hardcoding namespace names for better maintainability.

3. **Check access permissions** before attempting operations on organization resources.

4. **Use --all-orgs carefully** as it may return a large number of resources.

5. **Consider workload clusters** when managing apps that need to be deployed across multiple clusters.

## RBAC Considerations

- Users typically have access only to their organization's namespaces
- Cross-organization operations require elevated permissions
- The MCP server respects Kubernetes RBAC policies

## Examples

### List all apps across an organization's clusters
```bash
mcp app.list --organization giantswarm --include-workload-clusters
```

### Deploy an app to a specific organization
```bash
mcp app.create \
  --name monitoring-stack \
  --organization giantswarm \
  --catalog giantswarm \
  --app prometheus-operator \
  --version 11.0.0
```

### Check organization access
```bash
mcp organization.validate-access --organization giantswarm
```

### Get organization namespace details
```bash
mcp organization.info --namespace org-giantswarm
``` 