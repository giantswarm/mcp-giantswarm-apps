package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/giantswarm/mcp-giantswarm-apps/internal/k8s"
	internalServer "github.com/giantswarm/mcp-giantswarm-apps/internal/server"
	"github.com/giantswarm/mcp-giantswarm-apps/pkg/prompts"
	"github.com/giantswarm/mcp-giantswarm-apps/pkg/resources"
	"github.com/giantswarm/mcp-giantswarm-apps/pkg/tools"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

const (
	serverName    = "mcp-giantswarm-apps"
	serverVersion = "0.1.0"
)

// newServeCmd creates the Cobra command for starting the MCP server.
func newServeCmd() *cobra.Command {
	var (
		kubeContext string

		// Transport options
		transport       string
		httpAddr        string
		sseEndpoint     string
		messageEndpoint string
		httpEndpoint    string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the MCP Giant Swarm Apps server",
		Long: `Start the MCP Giant Swarm Apps server to provide tools for interacting
with Giant Swarm app management, catalogs, and configurations via the Model Context Protocol.

Supports multiple transport types:
  - stdio: Standard input/output (default)
  - sse: Server-Sent Events over HTTP
  - streamable-http: Streamable HTTP transport`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(kubeContext, transport, httpAddr, sseEndpoint, messageEndpoint, httpEndpoint)
		},
	}

	// Add flags for configuring the server
	cmd.Flags().StringVar(&kubeContext, "kube-context", "", "Kubernetes context to use (defaults to current context)")

	// Transport flags
	cmd.Flags().StringVar(&transport, "transport", "stdio", "Transport type: stdio, sse, or streamable-http")
	cmd.Flags().StringVar(&httpAddr, "http-addr", ":8080", "HTTP server address (for sse and streamable-http transports)")
	cmd.Flags().StringVar(&sseEndpoint, "sse-endpoint", "/sse", "SSE endpoint path (for sse transport)")
	cmd.Flags().StringVar(&messageEndpoint, "message-endpoint", "/message", "Message endpoint path (for sse transport)")
	cmd.Flags().StringVar(&httpEndpoint, "http-endpoint", "/mcp", "HTTP endpoint path (for streamable-http transport)")

	return cmd
}

// runServe contains the main server logic with support for multiple transports
func runServe(kubeContext, transport, httpAddr, sseEndpoint, messageEndpoint, httpEndpoint string) error {
	// Initialize logger
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Printf("Starting %s v%s", serverName, rootCmd.Version)

	// Setup graceful shutdown - listen for both SIGINT and SIGTERM
	shutdownCtx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Initialize Kubernetes client
	ctx := context.Background()
	if kubeContext == "" {
		kubeContext = os.Getenv("KUBE_CONTEXT") // Allow overriding context via env var
	}

	k8sClient, err := k8s.NewClient(ctx, kubeContext)
	if err != nil {
		return fmt.Errorf("failed to initialize Kubernetes client: %v", err)
	}
	log.Printf("Connected to Kubernetes cluster (context: %s)", k8sClient.GetCurrentContext())

	// Initialize dynamic client for CRDs
	dynamicClient, err := k8s.NewDynamicClient(k8sClient)
	if err != nil {
		return fmt.Errorf("failed to initialize dynamic client: %v", err)
	}

	// Check if Giant Swarm CRDs are available
	if err := dynamicClient.CheckCRDsExist(ctx, k8sClient); err != nil {
		log.Printf("Warning: %v", err)
		log.Println("Make sure you're connected to a Giant Swarm management cluster")
	}

	// Create server context
	serverCtx := internalServer.NewContext(k8sClient, dynamicClient)

	// Create MCP server
	mcpSrv := server.NewMCPServer(
		serverName,
		rootCmd.Version, // Use version from root command
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true), // subscribe, list
		server.WithPromptCapabilities(true),
		server.WithLogging(),
	)

	// Initialize tools
	if err := initializeTools(mcpSrv, serverCtx); err != nil {
		return fmt.Errorf("failed to initialize tools: %v", err)
	}

	// Initialize resources
	if err := initializeResources(mcpSrv, serverCtx); err != nil {
		return fmt.Errorf("failed to initialize resources: %v", err)
	}

	// Initialize prompts
	if err := initializePrompts(mcpSrv, serverCtx); err != nil {
		return fmt.Errorf("failed to initialize prompts: %v", err)
	}

	fmt.Printf("Starting MCP Giant Swarm Apps server with %s transport...\n", transport)

	// Start the appropriate server based on transport type
	switch transport {
	case "stdio":
		return runStdioServer(mcpSrv)
	case "sse":
		return runSSEServer(mcpSrv, httpAddr, sseEndpoint, messageEndpoint, shutdownCtx)
	case "streamable-http":
		return runStreamableHTTPServer(mcpSrv, httpAddr, httpEndpoint, shutdownCtx)
	default:
		return fmt.Errorf("unsupported transport type: %s (supported: stdio, sse, streamable-http)", transport)
	}
}

// runStdioServer runs the server with STDIO transport
func runStdioServer(mcpSrv *mcpserver.MCPServer) error {
	// Start the server in a goroutine so we can handle shutdown signals
	serverDone := make(chan error, 1)
	go func() {
		defer close(serverDone)
		if err := mcpserver.ServeStdio(mcpSrv); err != nil {
			serverDone <- err
		}
	}()

	// Wait for server completion
	select {
	case err := <-serverDone:
		if err != nil {
			return fmt.Errorf("server stopped with error: %w", err)
		} else {
			fmt.Println("Server stopped normally")
		}
	}

	fmt.Println("Server gracefully stopped")
	return nil
}

// runSSEServer runs the server with SSE transport
func runSSEServer(mcpSrv *mcpserver.MCPServer, addr, sseEndpoint, messageEndpoint string, ctx context.Context) error {
	// Create SSE server with custom endpoints
	sseServer := mcpserver.NewSSEServer(mcpSrv,
		mcpserver.WithSSEEndpoint(sseEndpoint),
		mcpserver.WithMessageEndpoint(messageEndpoint),
	)

	fmt.Printf("SSE server starting on %s\n", addr)
	fmt.Printf("  SSE endpoint: %s\n", sseEndpoint)
	fmt.Printf("  Message endpoint: %s\n", messageEndpoint)

	// Start server in goroutine
	serverDone := make(chan error, 1)
	go func() {
		defer close(serverDone)
		if err := sseServer.Start(addr); err != nil {
			serverDone <- err
		}
	}()

	// Wait for either shutdown signal or server completion
	select {
	case <-ctx.Done():
		fmt.Println("Shutdown signal received, stopping SSE server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30)
		defer cancel()
		if err := sseServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("error shutting down SSE server: %w", err)
		}
	case err := <-serverDone:
		if err != nil {
			return fmt.Errorf("SSE server stopped with error: %w", err)
		} else {
			fmt.Println("SSE server stopped normally")
		}
	}

	fmt.Println("SSE server gracefully stopped")
	return nil
}

// runStreamableHTTPServer runs the server with Streamable HTTP transport
func runStreamableHTTPServer(mcpSrv *mcpserver.MCPServer, addr, endpoint string, ctx context.Context) error {
	// Create Streamable HTTP server with custom endpoint
	httpServer := mcpserver.NewStreamableHTTPServer(mcpSrv,
		mcpserver.WithEndpointPath(endpoint),
	)

	fmt.Printf("Streamable HTTP server starting on %s\n", addr)
	fmt.Printf("  HTTP endpoint: %s\n", endpoint)

	// Start server in goroutine
	serverDone := make(chan error, 1)
	go func() {
		defer close(serverDone)
		if err := httpServer.Start(addr); err != nil {
			serverDone <- err
		}
	}()

	// Wait for either shutdown signal or server completion
	select {
	case <-ctx.Done():
		fmt.Println("Shutdown signal received, stopping HTTP server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("error shutting down HTTP server: %w", err)
		}
	case err := <-serverDone:
		if err != nil {
			return fmt.Errorf("HTTP server stopped with error: %w", err)
		} else {
			fmt.Println("HTTP server stopped normally")
		}
	}

	fmt.Println("HTTP server gracefully stopped")
	return nil
}

// initializeTools registers all MCP tools with the server (moved from original main.go)
func initializeTools(s *server.MCPServer, ctx *internalServer.Context) error {
	// Register app management tools
	if err := tools.RegisterAppTools(s, ctx); err != nil {
		return fmt.Errorf("failed to register app tools: %w", err)
	}

	// Register catalog management tools
	if err := tools.RegisterCatalogTools(s, ctx); err != nil {
		return fmt.Errorf("failed to register catalog tools: %w", err)
	}

	// Register app catalog entry tools
	if err := tools.RegisterAppCatalogEntryTools(s, ctx); err != nil {
		return fmt.Errorf("failed to register app catalog entry tools: %w", err)
	}

	// Register configuration management tools
	if err := tools.RegisterConfigTools(s, ctx); err != nil {
		return fmt.Errorf("failed to register config tools: %w", err)
	}

	// Register organization management tools
	if err := tools.RegisterOrganizationTools(s, ctx); err != nil {
		return fmt.Errorf("failed to register organization tools: %w", err)
	}

	// Register cluster management tools for CAPI integration
	if err := tools.RegisterClusterTools(s, ctx); err != nil {
		return fmt.Errorf("failed to register cluster tools: %w", err)
	}

	// Register prompts
	if err := prompts.RegisterPrompts(s, ctx); err != nil {
		return fmt.Errorf("failed to register prompts: %w", err)
	}

	// Health check tool
	healthTool := mcp.NewTool(
		"health",
		mcp.WithDescription("Check MCP server and Kubernetes connection health"),
	)

	s.AddTool(healthTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Check Kubernetes connection
		version, err := ctx.K8sClient.Discovery().ServerVersion()
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Kubernetes connection failed: %v", err)), nil
		}

		// Check Giant Swarm CRDs
		crdStatus := "available"
		if err := ctx.DynamicClient.CheckCRDsExist(toolCtx, ctx.K8sClient); err != nil {
			crdStatus = fmt.Sprintf("not available: %v", err)
		}

		healthStatus := fmt.Sprintf(`MCP Server Health Check:
- Server: %s v%s (healthy)
- Kubernetes: connected to %s
  - Version: %s
  - Context: %s
- Giant Swarm CRDs: %s`,
			serverName, rootCmd.Version,
			version.GitVersion,
			version.GitVersion,
			ctx.K8sClient.GetCurrentContext(),
			crdStatus,
		)

		return mcp.NewToolResultText(healthStatus), nil
	})

	// List contexts tool
	listContextsTool := mcp.NewTool(
		"kubernetes_contexts",
		mcp.WithDescription("List available Kubernetes contexts"),
	)

	s.AddTool(listContextsTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		contexts, current, err := k8s.ListContexts()
		if err != nil {
			return nil, fmt.Errorf("failed to list contexts: %w", err)
		}

		result := "Available Kubernetes contexts:\n"
		for _, context := range contexts {
			marker := "  "
			if context == current {
				marker = "* "
			}
			result += fmt.Sprintf("%s%s\n", marker, context)
		}

		return mcp.NewToolResultText(result), nil
	})

	return nil
}

// initializeResources registers all MCP resources with the server (moved from original main.go)
func initializeResources(s *server.MCPServer, ctx *internalServer.Context) error {
	// Create resource provider
	provider := resources.NewProvider(ctx.K8sClient, ctx.DynamicClient)

	// Register resource templates for dynamic resources
	// App resource template
	appTemplate := mcp.NewResourceTemplate(
		"app://{namespace}/{name}",
		"App Resource",
		mcp.WithTemplateDescription("Giant Swarm app details and status"),
		mcp.WithTemplateMIMEType("application/json"),
	)

	s.AddResourceTemplate(appTemplate, func(rctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		content, err := provider.GetResource(rctx, request.Params.URI)
		if err != nil {
			return nil, fmt.Errorf("failed to get resource %s: %w", request.Params.URI, err)
		}

		// Convert to JSON
		jsonData, err := json.MarshalIndent(content, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal resource content: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     string(jsonData),
			},
		}, nil
	})

	// Add remaining resource templates (simplified for now)
	// Full implementation would include catalog, config, schema, changelog templates

	return nil
}

// initializePrompts registers all MCP prompts with the server (moved from original main.go)
func initializePrompts(s *server.MCPServer, ctx *internalServer.Context) error {
	// Placeholder for prompt initialization
	// We'll add actual prompts in subsequent commits
	return nil
}
