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
- `app.list` - List apps with filtering options
- `app.get` - Get detailed app information
- `app.create` - Create a new app from catalog
- `app.update` - Update app configuration or version
- `app.delete` - Delete an app
- `app.rollback` - Rollback to previous version

### Catalog Management
- `catalog.list` - List available catalogs
- `catalog.get` - Get catalog details
- `catalog.search` - Search apps across catalogs
- `catalog.browse` - Browse apps in a specific catalog

### Configuration
- `config.get` - Get app configuration
- `config.set` - Update app configuration
- `config.validate` - Validate configuration

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