package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/giantswarm/mcp-giantswarm-apps/internal/k8s"
	internalServer "github.com/giantswarm/mcp-giantswarm-apps/internal/server"
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
		"kubernetes.contexts",
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
	// Placeholder for resource initialization
	// We'll add actual resources in subsequent commits
	return nil
}

func initializePrompts(s *server.MCPServer, ctx *internalServer.Context) error {
	// Placeholder for prompt initialization
	// We'll add actual prompts in subsequent commits
	return nil
} 