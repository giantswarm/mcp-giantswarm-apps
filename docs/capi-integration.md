# CAPI Integration and Workload Cluster Support

This document describes the Cluster API (CAPI) integration and workload cluster support in the MCP Giant Swarm Apps server.

## Overview

The MCP server integrates with Cluster API to provide comprehensive workload cluster management capabilities. This allows you to:

- Discover and list workload clusters across organizations
- Deploy apps to specific workload clusters
- Manage cross-cluster app deployments
- Monitor cluster lifecycle events

## Architecture

### Cluster Resources

The integration works with standard CAPI resources:

- **Cluster** (`cluster.x-k8s.io/v1beta1`): The main cluster resource
- **Machine**, **MachineSet**, **MachineDeployment**: Node management resources
- **Infrastructure Resources**: Provider-specific resources (AWSCluster, AzureCluster, etc.)

### Namespace Convention

Workload clusters follow Giant Swarm's namespace conventions:

- **Cluster Resources**: Located in organization namespaces (`org-*`)
- **Workload Apps**: Deployed to workload namespaces (`workload-{cluster-name}`)
- **Kubeconfig Secrets**: Named `{cluster-name}-kubeconfig`

## Cluster Management Tools

### cluster_list

List available workload clusters with filtering options.

```bash
# List all clusters
mcp cluster_list

# List clusters for a specific organization
mcp cluster_list --organization giantswarm

# Filter by provider
mcp cluster_list --provider aws

# Show only ready clusters
mcp cluster_list --ready-only

# Filter by labels
mcp cluster_list --labels "environment=production,team=platform"
```

**Output includes:**
- Cluster name and namespace
- Organization ownership
- Infrastructure provider
- Readiness status
- Infrastructure and control plane status
- Current conditions

### cluster_get

Get detailed information about a specific cluster.

```bash
# Get cluster by name (searches all namespaces)
mcp cluster_get --name prod-cluster

# Get cluster in specific namespace
mcp cluster_get --name prod-cluster --namespace org-giantswarm

# Get cluster for organization
mcp cluster_get --name prod-cluster --organization giantswarm
```

**Output includes:**
- Full cluster specification
- Network configuration (service/pod CIDRs)
- Infrastructure and control plane references
- Detailed status and conditions
- Kubeconfig availability
- Associated labels

### cluster_apps

List all apps deployed to a specific workload cluster.

```bash
# List apps in a cluster
mcp cluster_apps --cluster prod-cluster

# Specify organization for faster lookup
mcp cluster_apps --cluster prod-cluster --organization giantswarm

# Specify exact namespace
mcp cluster_apps --cluster prod-cluster --namespace org-giantswarm
```

**Output includes:**
- Apps in the workload namespace
- Apps targeting the cluster via kubeconfig
- App status and version information

## Workload Cluster App Deployment

### Deploying to Workload Clusters

When creating apps, you can target specific workload clusters:

```bash
# Deploy app to workload cluster
mcp app_create \
  --name nginx-ingress \
  --namespace org-giantswarm \
  --catalog giantswarm \
  --app nginx-ingress-controller \
  --version 2.1.0 \
  --cluster prod-cluster
```

The `--cluster` parameter:
- Overrides the `--in-cluster` flag
- Indicates the app should be deployed to the specified workload cluster
- Requires the app operator to have access to the cluster's kubeconfig

### App Deployment Patterns

There are two main patterns for deploying apps to workload clusters:

1. **In-Cluster Apps**: Deployed directly to the management cluster
   ```bash
   mcp app_create --name my-app --in-cluster ...
   ```

2. **Cross-Cluster Apps**: Deployed to workload clusters
   ```bash
   mcp app_create --name my-app --cluster workload-1 ...
   ```

## Cluster Lifecycle Management

### Cluster Status

Clusters go through several phases:

- **Pending**: Cluster is being provisioned
- **Provisioning**: Infrastructure is being created
- **Provisioned**: Cluster is ready for use
- **Deleting**: Cluster is being deleted
- **Failed**: Cluster provisioning failed

### Readiness Checks

A cluster is considered ready when:
- Phase is "Provisioned"
- Infrastructure is ready
- Control plane is ready

Use the `--ready-only` flag to filter for ready clusters:
```bash
mcp cluster_list --ready-only
```

## Labels and Filtering

### Standard Labels

Clusters use these standard labels:

- `giantswarm.io/organization`: Organization that owns the cluster
- `giantswarm.io/cluster-type`: Type of cluster (workload, management)
- `cluster.x-k8s.io/provider`: Infrastructure provider
- `environment`: Environment designation (dev, staging, prod)
- `team`: Team ownership

### Filtering Examples

```bash
# Find production clusters
mcp cluster_list --labels "environment=production"

# Find AWS clusters owned by platform team
mcp cluster_list --labels "provider=aws,team=platform"

# Find all clusters for an organization
mcp cluster_list --organization giantswarm
```

## Kubeconfig Management

### Accessing Workload Clusters

Kubeconfig secrets follow the naming pattern `{cluster-name}-kubeconfig`:

```bash
# Check if kubeconfig is available
mcp cluster_get --name prod-cluster

# The output will show:
# Kubeconfig: Available
# Secret: prod-cluster-kubeconfig
```

### Security Considerations

- Kubeconfig secrets are stored in the cluster's namespace
- Access is controlled by Kubernetes RBAC
- The MCP server respects RBAC permissions

## Best Practices

1. **Always specify organization** when working with clusters for better performance
2. **Use labels** to organize and filter clusters effectively
3. **Check cluster readiness** before deploying apps
4. **Monitor cluster conditions** for potential issues
5. **Verify kubeconfig availability** before cross-cluster deployments

## Troubleshooting

### Cluster Not Found

If a cluster is not found:
1. Check you have access to the namespace
2. Verify the cluster name is correct
3. Try listing all clusters to see available options

### Apps Not Deploying to Workload Cluster

If apps aren't deploying to the workload cluster:
1. Verify the cluster is ready
2. Check kubeconfig secret exists
3. Ensure app operator has necessary permissions
4. Check the workload namespace exists

### No Clusters Listed

If no clusters appear:
1. Verify CAPI CRDs are installed
2. Check namespace access permissions
3. Ensure you're connected to a management cluster

## Examples

### Full Workflow: Deploy App to New Workload Cluster

```bash
# 1. List available clusters
mcp cluster_list --organization giantswarm

# 2. Check specific cluster details
mcp cluster_get --name prod-cluster --organization giantswarm

# 3. Verify cluster is ready
# (Check output for Ready: true)

# 4. Deploy app to the cluster
mcp app_create \
  --name monitoring-stack \
  --namespace org-giantswarm \
  --catalog giantswarm \
  --app prometheus-operator \
  --version 11.0.0 \
  --cluster prod-cluster

# 5. Verify deployment
mcp cluster_apps --cluster prod-cluster --organization giantswarm
```

### Monitor Multiple Clusters

```bash
# List all production clusters across organizations
mcp cluster_list --labels "environment=production"

# Check ready status for each
for cluster in $(mcp cluster_list --labels "environment=production" | grep "Name:" | cut -d' ' -f2); do
  echo "Checking $cluster..."
  mcp cluster_get --name $cluster | grep "Ready:"
done
```

## Integration with CI/CD

The cluster tools can be integrated into CI/CD pipelines:

```yaml
# Example: Deploy to cluster based on branch
- name: Deploy to cluster
  run: |
    if [ "$BRANCH" = "main" ]; then
      CLUSTER="prod-cluster"
    else
      CLUSTER="staging-cluster"
    fi
    
    mcp app_create \
      --name my-app \
      --namespace org-giantswarm \
      --catalog internal \
      --app my-app \
      --version $VERSION \
      --cluster $CLUSTER
```

## Future Enhancements

Planned improvements for CAPI integration:

1. **Direct kubeconfig management**: Tools to retrieve and use kubeconfig
2. **Cluster creation**: Support for creating new workload clusters
3. **Machine management**: Tools for node operations
4. **Cluster upgrades**: Automated cluster version upgrades
5. **Multi-cluster deployments**: Deploy apps to multiple clusters simultaneously 