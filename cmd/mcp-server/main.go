package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/giantswarm/mcp-giantswarm-apps/internal/k8s"
	internalServer "github.com/giantswarm/mcp-giantswarm-apps/internal/server"
	"github.com/giantswarm/mcp-giantswarm-apps/pkg/prompts"
	"github.com/giantswarm/mcp-giantswarm-apps/pkg/resources"
	"github.com/giantswarm/mcp-giantswarm-apps/pkg/tools"
)

const (
	serverName    = "mcp-giantswarm-apps"
	serverVersion = "0.1.0"
)

func main() {
	// Initialize logger
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Printf("Starting %s v%s", serverName, serverVersion)

	// Initialize Kubernetes client
	ctx := context.Background()
	kubeContext := os.Getenv("KUBE_CONTEXT") // Allow overriding context via env var

	k8sClient, err := k8s.NewClient(ctx, kubeContext)
	if err != nil {
		log.Fatalf("Failed to initialize Kubernetes client: %v", err)
	}
	log.Printf("Connected to Kubernetes cluster (context: %s)", k8sClient.GetCurrentContext())

	// Initialize dynamic client for CRDs
	dynamicClient, err := k8s.NewDynamicClient(k8sClient)
	if err != nil {
		log.Fatalf("Failed to initialize dynamic client: %v", err)
	}

	// Check if Giant Swarm CRDs are available
	if err := dynamicClient.CheckCRDsExist(ctx, k8sClient); err != nil {
		log.Printf("Warning: %v", err)
		log.Println("Make sure you're connected to a Giant Swarm management cluster")
	}

	// Create server context
	serverCtx := internalServer.NewContext(k8sClient, dynamicClient)

	// Create MCP server
	s := server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true), // subscribe, list
		server.WithPromptCapabilities(true),
		server.WithLogging(),
	)

	// Initialize tools
	if err := initializeTools(s, serverCtx); err != nil {
		log.Fatalf("Failed to initialize tools: %v", err)
	}

	// Initialize resources
	if err := initializeResources(s, serverCtx); err != nil {
		log.Fatalf("Failed to initialize resources: %v", err)
	}

	// Initialize prompts
	if err := initializePrompts(s, serverCtx); err != nil {
		log.Fatalf("Failed to initialize prompts: %v", err)
	}

	// Start server with stdio transport
	log.Println("MCP server started on stdio transport")
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

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
			serverName, serverVersion,
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

	// Catalog resource template
	catalogTemplate := mcp.NewResourceTemplate(
		"catalog://{name}",
		"Catalog Resource",
		mcp.WithTemplateDescription("Giant Swarm catalog information"),
		mcp.WithTemplateMIMEType("application/json"),
	)

	s.AddResourceTemplate(catalogTemplate, func(rctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
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

	// Config resource template
	configTemplate := mcp.NewResourceTemplate(
		"config://{namespace}/{app}/values",
		"Config Resource",
		mcp.WithTemplateDescription("App configuration values"),
		mcp.WithTemplateMIMEType("application/json"),
	)

	s.AddResourceTemplate(configTemplate, func(rctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
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

	// Schema resource template
	schemaTemplate := mcp.NewResourceTemplate(
		"schema://{catalog}/{app}/{version}",
		"Schema Resource",
		mcp.WithTemplateDescription("App configuration schema"),
		mcp.WithTemplateMIMEType("application/json"),
	)

	s.AddResourceTemplate(schemaTemplate, func(rctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
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

	// Changelog resource template
	changelogTemplate := mcp.NewResourceTemplate(
		"changelog://{catalog}/{app}",
		"Changelog Resource",
		mcp.WithTemplateDescription("App version changelog"),
		mcp.WithTemplateMIMEType("application/json"),
	)

	s.AddResourceTemplate(changelogTemplate, func(rctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
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

	return nil
}

func initializePrompts(s *server.MCPServer, ctx *internalServer.Context) error {
	// Placeholder for prompt initialization
	// We'll add actual prompts in subsequent commits
	return nil
}
