package main

import (
	"context"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	serverName    = "mcp-giantswarm-apps"
	serverVersion = "0.1.0"
)

func main() {
	// Initialize logger
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Printf("Starting %s v%s", serverName, serverVersion)

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
	if err := initializeTools(s); err != nil {
		log.Fatalf("Failed to initialize tools: %v", err)
	}

	// Initialize resources
	if err := initializeResources(s); err != nil {
		log.Fatalf("Failed to initialize resources: %v", err)
	}

	// Initialize prompts
	if err := initializePrompts(s); err != nil {
		log.Fatalf("Failed to initialize prompts: %v", err)
	}

	// Start server with stdio transport
	log.Println("MCP server started on stdio transport")
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func initializeTools(s *server.MCPServer) error {
	// Placeholder for tool initialization
	// We'll add actual tools in subsequent commits

	// Example: Health check tool
	healthTool := mcp.NewTool(
		"health",
		mcp.WithDescription("Check MCP server health"),
		mcp.WithString("message", mcp.Required(), mcp.Description("Optional message")),
	)

	s.AddTool(healthTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		message := args["message"].(string)
		return mcp.NewToolResultText(fmt.Sprintf("Server is healthy! Message: %s", message)), nil
	})

	return nil
}

func initializeResources(s *server.MCPServer) error {
	// Placeholder for resource initialization
	// We'll add actual resources in subsequent commits
	return nil
}

func initializePrompts(s *server.MCPServer) error {
	// Placeholder for prompt initialization
	// We'll add actual prompts in subsequent commits
	return nil
} 