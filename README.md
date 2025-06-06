# MCP Server for Giant Swarm App Platform

An MCP (Model Context Protocol) server that provides tools and resources for managing Giant Swarm App Platform deployments.

## Features

- **App Management**: Create, update, list, and delete Giant Swarm apps
- **Catalog Support**: Browse and search app catalogs and available app versions
- **Configuration Management**: Handle app configurations via ConfigMaps and Secrets
- **Multi-namespace Support**: Work with organization-based namespaces
- **CAPI Integration**: Support for workload cluster app deployments
- **Prompts**: Interactive guides for common operations

## Prerequisites

- Go 1.21 or later
- Access to a Giant Swarm management cluster
- Kubernetes credentials configured (`kubectl` access)

## Installation

### Prerequisites

- Go 1.21 or later
- Access to a Giant Swarm management cluster
- Valid kubeconfig for authentication

### Building from Source

```bash
# Clone the repository
git clone https://github.com/giantswarm/mcp-giantswarm-apps.git
cd mcp-giantswarm-apps

# Install dependencies
make deps

# Build the server
make build

# Install to $GOPATH/bin
make install
```

## Usage

### Running the Server

The server runs using stdio transport by default:

```bash
# Run directly
./build/mcp-giantswarm-apps

# Or if installed
mcp-giantswarm-apps
```

### Configuration

The server uses your current kubeconfig context by default. You can specify a different context:

```bash
export KUBECONFIG=/path/to/kubeconfig
mcp-giantswarm-apps
```

### Integration with AI Assistants

To use with Claude Desktop or other MCP-compatible clients, add to your configuration:

```json
{
  "mcpServers": {
    "giantswarm-apps": {
      "command": "mcp-giantswarm-apps",
      "args": []
    }
  }
}
```

## Available Tools

### App Management

- `app_list` - List Giant Swarm apps with filtering options
- `app_get` - Get detailed information about a specific app
- `app_create` - Create a new Giant Swarm app
- `app_update` - Update an existing app
- `app_delete` - Delete an app

### Catalog Management

- `catalog_list` - List available app catalogs
- `catalog_get` - Get detailed catalog information
- `catalog_refresh` - Refresh catalog entries
- `catalog_search` - Search for apps across catalogs

### App Catalog Entries

- `appcatalogentry_list` - List apps from catalogs
- `appcatalogentry_get` - Get detailed app information
- `appcatalogentry_versions` - List available versions
- `appcatalogentry_search` - Search catalog entries

### Configuration Management

- `config_get` - Get app configuration
- `config_create` - Create new configuration
- `config_update` - Update configuration
- `config_values` - Get configuration values

### Organization Management  

- `organization_list` - List organizations
- `organization_namespaces` - List organization namespaces
- `organization_info` - Get namespace details
- `organization_validate_access` - Check access permissions

### Cluster Management (CAPI)

- `cluster_list` - List available workload clusters
- `cluster_get` - Get detailed cluster information
- `cluster_apps` - List apps deployed to a specific cluster

### System Tools

- `health` - Check server and connection health
- `kubernetes_contexts` - List available contexts

## Available Resources

The server exposes various resources:
- `app://{namespace}/{name}` - App details and status
- `catalog://{name}` - Catalog information
- `config://{namespace}/{app}/values` - App configuration

## Usage Examples

### List workload clusters

```bash
# List all clusters
mcp cluster_list

# List clusters for an organization
mcp cluster_list --organization giantswarm

# List only ready clusters
mcp cluster_list --ready-only
```

### Deploy app to workload cluster

```bash
# Deploy app to a specific workload cluster
mcp app_create \
  --name nginx-ingress \
  --namespace org-giantswarm \
  --catalog giantswarm \
  --app nginx-ingress-controller \
  --version 2.1.0 \
  --cluster prod-cluster
```

### List apps in a workload cluster

```bash
# List all apps in a specific cluster
mcp cluster_apps --cluster prod-cluster --organization giantswarm
```

## Development

### Project Structure

```
.
├── cmd/mcp-server/      # Main server entry point
├── pkg/
│   ├── app/            # App management logic
│   ├── catalog/        # Catalog handling
│   └── config/         # Configuration management
├── internal/
│   └── k8s/           # Kubernetes client utilities
└── Makefile           # Build automation
```

### Running Tests

```bash
make test
```

### Contributing

Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Support

For issues and feature requests, please use the [GitHub issue tracker](https://github.com/giantswarm/mcp-giantswarm-apps/issues). 