# MCP Giant Swarm Apps Server

An MCP (Model Context Protocol) server for managing Giant Swarm App Platform applications. This server enables AI assistants to interact with Giant Swarm apps, catalogs, and configurations.

## Features

- **App Management**: Create, update, delete, and manage apps deployed through Giant Swarm App Platform
- **Catalog Browsing**: Browse and search available app catalogs and their entries
- **Configuration Management**: Handle app configurations through ConfigMaps and Secrets
- **Multi-tenancy Support**: Work with Giant Swarm's organization-based namespace model
- **CAPI Integration**: Support for workload cluster app deployments

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
- `app_list` - List apps with filtering options
- `app_get` - Get detailed app information
- `app_create` - Create a new app from catalog
- `app_update` - Update app configuration or version
- `app_delete` - Delete an app

### Catalog Management
- `catalog_list` - List available catalogs
- `catalog_get` - Get catalog details
- `catalog_create` - Create a new catalog
- `catalog_update` - Update catalog settings
- `catalog_delete` - Delete a catalog

### App Catalog Entry Management
- `appcatalogentry_list` - List app catalog entries
- `appcatalogentry_get` - Get app catalog entry details
- `appcatalogentry_search` - Search for apps in catalogs
- `appcatalogentry_versions` - List all versions of an app

### Configuration
- `config_get` - Get app configuration
- `config_set` - Update app configuration
- `config_validate` - Validate configuration
- `config_diff` - Compare configurations
- `config_merge` - Merge multiple configurations
- `secret_create` - Create a new secret
- `secret_update` - Update an existing secret

### Organization Management
- `organization_list` - List all organizations
- `organization_namespaces` - List namespaces for an organization
- `organization_info` - Get namespace organization context
- `organization_validate_access` - Check access permissions

### Kubernetes
- `health` - Check server and Kubernetes health
- `kubernetes_contexts` - List available Kubernetes contexts

### Interactive Prompts
- `deploy-app` - Step-by-step guide to deploy an app
- `upgrade-app` - Guide for upgrading apps safely
- `troubleshoot-app` - Comprehensive troubleshooting guide
- `create-catalog` - Create custom app catalogs
- `configure-app` - Interactive configuration wizard

## Resources

The server exposes various resources:
- `app://{namespace}/{name}` - App details and status
- `catalog://{name}` - Catalog information
- `config://{namespace}/{app}/values` - App configuration

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