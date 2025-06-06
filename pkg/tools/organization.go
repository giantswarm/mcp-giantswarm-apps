package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/giantswarm/mcp-giantswarm-apps/internal/server"
	"github.com/giantswarm/mcp-giantswarm-apps/pkg/organization"
)

// RegisterOrganizationTools registers all organization management tools
func RegisterOrganizationTools(s *mcpserver.MCPServer, ctx *server.Context) error {
	// organization_list tool
	listTool := mcp.NewTool(
		"organization_list",
		mcp.WithDescription("List all organizations in the cluster"),
		mcp.WithBoolean("detailed", mcp.Description("Include detailed namespace information")),
	)

	s.AddTool(listTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		detailed := getBoolArg(args, "detailed")

		// Get all organization namespaces
		orgNamespaces, err := organization.ListOrganizationNamespaces(toolCtx, ctx.K8sClient)
		if err != nil {
			return nil, fmt.Errorf("failed to list organization namespaces: %w", err)
		}

		if len(orgNamespaces) == 0 {
			return mcp.NewToolResultText("No organizations found"), nil
		}

		var output strings.Builder
		output.WriteString(fmt.Sprintf("Found %d organizations:\n\n", len(orgNamespaces)))

		for _, ns := range orgNamespaces {
			orgName, _ := organization.GetOrganizationFromNamespace(ns)
			output.WriteString(fmt.Sprintf("- %s (namespace: %s)\n", orgName, ns))

			if detailed {
				// Get namespace info
				info, err := organization.GetNamespaceInfo(toolCtx, ctx.K8sClient, ns)
				if err == nil && len(info.Labels) > 0 {
					output.WriteString("  Labels:\n")
					for k, v := range info.Labels {
						output.WriteString(fmt.Sprintf("    %s: %s\n", k, v))
					}
				}

				// List workload cluster namespaces for this org
				allNs, err := organization.GetNamespacesByOrganization(toolCtx, ctx.K8sClient, orgName)
				if err == nil && len(allNs) > 1 {
					output.WriteString("  Related namespaces:\n")
					for _, relatedNs := range allNs {
						if relatedNs != ns {
							output.WriteString(fmt.Sprintf("    - %s\n", relatedNs))
						}
					}
				}
			}
		}

		return mcp.NewToolResultText(output.String()), nil
	})

	// organization_namespaces tool
	namespacesTool := mcp.NewTool(
		"organization_namespaces",
		mcp.WithDescription("List all namespaces belonging to an organization"),
		mcp.WithString("organization", mcp.Required(), mcp.Description("Organization name (e.g., 'giantswarm')")),
		mcp.WithBoolean("include-details", mcp.Description("Include namespace details and type")),
	)

	s.AddTool(namespacesTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		orgName := args["organization"].(string)
		includeDetails := getBoolArg(args, "include-details")

		// Get all namespaces for this organization
		namespaces, err := organization.GetNamespacesByOrganization(toolCtx, ctx.K8sClient, orgName)
		if err != nil {
			return nil, fmt.Errorf("failed to get namespaces for organization %s: %w", orgName, err)
		}

		if len(namespaces) == 0 {
			return mcp.NewToolResultText(fmt.Sprintf("No namespaces found for organization %s", orgName)), nil
		}

		var output strings.Builder
		output.WriteString(fmt.Sprintf("Namespaces for organization '%s':\n\n", orgName))

		for _, ns := range namespaces {
			if includeDetails {
				info, err := organization.GetNamespaceInfo(toolCtx, ctx.K8sClient, ns)
				if err == nil {
					output.WriteString(fmt.Sprintf("- %s (type: %s)\n", ns, info.Type))
					if info.ClusterID != "" {
						output.WriteString(fmt.Sprintf("  Cluster ID: %s\n", info.ClusterID))
					}
				} else {
					output.WriteString(fmt.Sprintf("- %s\n", ns))
				}
			} else {
				output.WriteString(fmt.Sprintf("- %s\n", ns))
			}
		}

		return mcp.NewToolResultText(output.String()), nil
	})

	// organization_info tool
	infoTool := mcp.NewTool(
		"organization_info",
		mcp.WithDescription("Get detailed information about a namespace and its organization context"),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace name")),
	)

	s.AddTool(infoTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		namespace := args["namespace"].(string)

		// Get namespace info
		info, err := organization.GetNamespaceInfo(toolCtx, ctx.K8sClient, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to get namespace info: %w", err)
		}

		var output strings.Builder
		output.WriteString(fmt.Sprintf("Namespace: %s\n", info.Name))
		output.WriteString(fmt.Sprintf("Type: %s\n", info.Type))

		if info.Organization != "" {
			output.WriteString(fmt.Sprintf("Organization: %s\n", info.Organization))
		}

		if info.ClusterID != "" {
			output.WriteString(fmt.Sprintf("Cluster ID: %s\n", info.ClusterID))
		}

		if len(info.Labels) > 0 {
			output.WriteString("\nLabels:\n")
			for k, v := range info.Labels {
				output.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
			}
		}

		// Check access
		err = organization.ValidateNamespaceAccess(toolCtx, ctx.K8sClient, namespace)
		if err != nil {
			output.WriteString(fmt.Sprintf("\nAccess: No (%v)\n", err))
		} else {
			output.WriteString("\nAccess: Yes\n")
		}

		return mcp.NewToolResultText(output.String()), nil
	})

	// organization_validate_access tool
	validateTool := mcp.NewTool(
		"organization_validate_access",
		mcp.WithDescription("Validate access to a namespace or organization"),
		mcp.WithString("namespace", mcp.Description("Namespace to validate access to")),
		mcp.WithString("organization", mcp.Description("Organization to validate access to")),
	)

	s.AddTool(validateTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		namespace := getStringArg(args, "namespace")
		orgName := getStringArg(args, "organization")

		if namespace == "" && orgName == "" {
			return nil, fmt.Errorf("either namespace or organization must be specified")
		}

		var output strings.Builder

		if namespace != "" {
			err := organization.ValidateNamespaceAccess(toolCtx, ctx.K8sClient, namespace)
			if err != nil {
				output.WriteString(fmt.Sprintf("Access to namespace '%s': DENIED\n", namespace))
				output.WriteString(fmt.Sprintf("Reason: %v\n", err))
			} else {
				output.WriteString(fmt.Sprintf("Access to namespace '%s': GRANTED\n", namespace))
			}
		}

		if orgName != "" {
			// Check access to organization namespaces
			namespaces, err := organization.GetNamespacesByOrganization(toolCtx, ctx.K8sClient, orgName)
			if err != nil {
				output.WriteString(fmt.Sprintf("\nFailed to get namespaces for organization '%s': %v\n", orgName, err))
			} else {
				output.WriteString(fmt.Sprintf("\nAccess to organization '%s' namespaces:\n", orgName))
				accessible := 0
				for _, ns := range namespaces {
					err := organization.ValidateNamespaceAccess(toolCtx, ctx.K8sClient, ns)
					if err == nil {
						accessible++
						output.WriteString(fmt.Sprintf("  - %s: GRANTED\n", ns))
					} else {
						output.WriteString(fmt.Sprintf("  - %s: DENIED\n", ns))
					}
				}
				output.WriteString(fmt.Sprintf("\nAccessible: %d/%d namespaces\n", accessible, len(namespaces)))
			}
		}

		return mcp.NewToolResultText(output.String()), nil
	})

	return nil
}
