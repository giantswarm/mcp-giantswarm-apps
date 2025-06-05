package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/giantswarm/mcp-giantswarm-apps/internal/server"
	"github.com/giantswarm/mcp-giantswarm-apps/pkg/appcatalogentry"
)

// RegisterAppCatalogEntryTools registers all AppCatalogEntry management tools
func RegisterAppCatalogEntryTools(s *mcpserver.MCPServer, ctx *server.Context) error {
	client := appcatalogentry.NewClient(ctx.DynamicClient)

	// appcatalogentry.list tool
	listTool := mcp.NewTool(
		"appcatalogentry.list",
		mcp.WithDescription("List app catalog entries"),
		mcp.WithString("namespace", mcp.Description("Namespace to list entries from (empty for all namespaces)")),
		mcp.WithString("catalog", mcp.Description("Filter by catalog name")),
		mcp.WithString("catalog-namespace", mcp.Description("Catalog namespace (used with catalog filter)")),
		mcp.WithBoolean("cluster-apps", mcp.Description("Show only cluster-wide apps")),
		mcp.WithBoolean("latest-only", mcp.Description("Show only latest version of each app")),
	)

	s.AddTool(listTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		
		namespace := getStringArg(args, "namespace")
		catalogName := getStringArg(args, "catalog")
		catalogNamespace := getStringArg(args, "catalog-namespace")
		clusterApps := getBoolArg(args, "cluster-apps")
		latestOnly := getBoolArg(args, "latest-only")

		var entries []*appcatalogentry.AppCatalogEntry
		var err error

		if catalogName != "" {
			entries, err = client.ListByCatalog(toolCtx, catalogName, catalogNamespace)
		} else {
			entries, err = client.List(toolCtx, namespace)
		}

		if err != nil {
			return nil, err
		}

		// Apply filters
		if clusterApps {
			entries = appcatalogentry.FilterByRestrictions(entries, true)
		}

		// Group by app and show only latest if requested
		if latestOnly {
			grouped := appcatalogentry.GroupByApp(entries)
			entries = make([]*appcatalogentry.AppCatalogEntry, 0)
			for _, versions := range grouped {
				if len(versions) > 0 {
					// Sort by date and take the latest
					sorted := appcatalogentry.SortByDate(versions)
					entries = append(entries, sorted[0])
				}
			}
		}

		// Format output
		if len(entries) == 0 {
			return mcp.NewToolResultText("No app catalog entries found"), nil
		}

		var output strings.Builder
		output.WriteString(fmt.Sprintf("Found %d app catalog entries:\n\n", len(entries)))
		
		for _, entry := range entries {
			output.WriteString(fmt.Sprintf("Name: %s\n", entry.Name))
			output.WriteString(fmt.Sprintf("App: %s\n", entry.Spec.AppName))
			output.WriteString(fmt.Sprintf("Version: %s (App: %s)\n", entry.GetLatestVersion(), entry.GetAppVersion()))
			output.WriteString(fmt.Sprintf("Catalog: %s/%s\n", entry.Spec.Catalog.Namespace, entry.Spec.Catalog.Name))
			if entry.Spec.Chart.Description != "" {
				output.WriteString(fmt.Sprintf("Description: %s\n", entry.Spec.Chart.Description))
			}
			if entry.IsClusterApp() {
				output.WriteString("Type: Cluster App\n")
			}
			output.WriteString("---\n")
		}

		return mcp.NewToolResultText(output.String()), nil
	})

	// appcatalogentry.get tool
	getTool := mcp.NewTool(
		"appcatalogentry.get",
		mcp.WithDescription("Get detailed information about a specific app catalog entry"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the app catalog entry")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the app catalog entry")),
	)

	s.AddTool(getTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		name := args["name"].(string)
		namespace := args["namespace"].(string)

		entry, err := client.Get(toolCtx, namespace, name)
		if err != nil {
			return nil, err
		}

		var output strings.Builder
		output.WriteString(fmt.Sprintf("App Catalog Entry: %s\n", entry.Name))
		output.WriteString(fmt.Sprintf("Namespace: %s\n", entry.Namespace))
		
		output.WriteString("\nApp Information:\n")
		output.WriteString(fmt.Sprintf("  App Name: %s\n", entry.Spec.AppName))
		output.WriteString(fmt.Sprintf("  App Version: %s\n", entry.Spec.AppVersion))
		
		output.WriteString("\nCatalog:\n")
		output.WriteString(fmt.Sprintf("  Name: %s\n", entry.Spec.Catalog.Name))
		output.WriteString(fmt.Sprintf("  Namespace: %s\n", entry.Spec.Catalog.Namespace))
		
		output.WriteString("\nChart Details:\n")
		output.WriteString(fmt.Sprintf("  Name: %s\n", entry.Spec.Chart.Name))
		output.WriteString(fmt.Sprintf("  Version: %s\n", entry.Spec.Chart.Version))
		output.WriteString(fmt.Sprintf("  App Version: %s\n", entry.Spec.Chart.AppVersion))
		if entry.Spec.Chart.Description != "" {
			output.WriteString(fmt.Sprintf("  Description: %s\n", entry.Spec.Chart.Description))
		}
		if entry.Spec.Chart.Home != "" {
			output.WriteString(fmt.Sprintf("  Home: %s\n", entry.Spec.Chart.Home))
		}
		if entry.Spec.Chart.Icon != "" {
			output.WriteString(fmt.Sprintf("  Icon: %s\n", entry.Spec.Chart.Icon))
		}
		
		if len(entry.Spec.Chart.Keywords) > 0 {
			output.WriteString(fmt.Sprintf("  Keywords: %s\n", strings.Join(entry.Spec.Chart.Keywords, ", ")))
		}
		
		if len(entry.Spec.Chart.Sources) > 0 {
			output.WriteString("  Sources:\n")
			for _, source := range entry.Spec.Chart.Sources {
				output.WriteString(fmt.Sprintf("    - %s\n", source))
			}
		}
		
		if len(entry.Spec.Chart.URLs) > 0 {
			output.WriteString("  URLs:\n")
			for _, url := range entry.Spec.Chart.URLs {
				output.WriteString(fmt.Sprintf("    - %s\n", url))
			}
		}

		if entry.Spec.Restrictions != nil {
			output.WriteString("\nRestrictions:\n")
			output.WriteString(fmt.Sprintf("  Cluster Singleton: %v\n", entry.Spec.Restrictions.ClusterSingleton))
			output.WriteString(fmt.Sprintf("  Namespace Singleton: %v\n", entry.Spec.Restrictions.NamespaceSingleton))
			if entry.Spec.Restrictions.FixedNamespace != "" {
				output.WriteString(fmt.Sprintf("  Fixed Namespace: %s\n", entry.Spec.Restrictions.FixedNamespace))
			}
			output.WriteString(fmt.Sprintf("  GPU Instances: %v\n", entry.Spec.Restrictions.GpuInstances))
		}

		if entry.Spec.DateCreated != nil {
			output.WriteString(fmt.Sprintf("\nCreated: %s\n", entry.Spec.DateCreated.Format("2006-01-02 15:04:05")))
		}
		if entry.Spec.DateUpdated != nil {
			output.WriteString(fmt.Sprintf("Updated: %s\n", entry.Spec.DateUpdated.Format("2006-01-02 15:04:05")))
		}

		return mcp.NewToolResultText(output.String()), nil
	})

	// appcatalogentry.search tool
	searchTool := mcp.NewTool(
		"appcatalogentry.search",
		mcp.WithDescription("Search for apps in the catalog"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query (searches in name, description, keywords)")),
		mcp.WithBoolean("cluster-apps", mcp.Description("Show only cluster-wide apps")),
	)

	s.AddTool(searchTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		query := args["query"].(string)
		clusterApps := getBoolArg(args, "cluster-apps")

		results, err := client.Search(toolCtx, query)
		if err != nil {
			return nil, err
		}

		// Apply filters
		if clusterApps {
			results = appcatalogentry.FilterByRestrictions(results, true)
		}

		if len(results) == 0 {
			return mcp.NewToolResultText(fmt.Sprintf("No apps found matching '%s'", query)), nil
		}

		var output strings.Builder
		output.WriteString(fmt.Sprintf("Found %d apps matching '%s':\n\n", len(results), query))
		
		// Group by app to show all versions together
		grouped := appcatalogentry.GroupByApp(results)
		
		for appName, versions := range grouped {
			output.WriteString(fmt.Sprintf("App: %s\n", appName))
			
			// Sort versions by date
			sorted := appcatalogentry.SortByDate(versions)
			
			for i, entry := range sorted {
				if i == 0 {
					output.WriteString(fmt.Sprintf("  Latest: %s (App: %s)\n", entry.GetLatestVersion(), entry.GetAppVersion()))
					if entry.Spec.Chart.Description != "" {
						output.WriteString(fmt.Sprintf("  Description: %s\n", entry.Spec.Chart.Description))
					}
					output.WriteString(fmt.Sprintf("  Catalog: %s/%s\n", entry.Spec.Catalog.Namespace, entry.Spec.Catalog.Name))
					if entry.IsClusterApp() {
						output.WriteString("  Type: Cluster App\n")
					}
				} else {
					output.WriteString(fmt.Sprintf("  Other versions: %s", entry.GetLatestVersion()))
					if i < len(sorted)-1 {
						output.WriteString(", ")
					} else {
						output.WriteString("\n")
					}
				}
			}
			output.WriteString("---\n")
		}

		return mcp.NewToolResultText(output.String()), nil
	})

	// appcatalogentry.versions tool
	versionsTool := mcp.NewTool(
		"appcatalogentry.versions",
		mcp.WithDescription("List all available versions of an app"),
		mcp.WithString("app", mcp.Required(), mcp.Description("App name to get versions for")),
	)

	s.AddTool(versionsTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		appName := args["app"].(string)

		versions, err := client.GetVersions(toolCtx, appName)
		if err != nil {
			return nil, err
		}

		if len(versions) == 0 {
			return mcp.NewToolResultText(fmt.Sprintf("No versions found for app '%s'", appName)), nil
		}

		// Sort by date
		sorted := appcatalogentry.SortByDate(versions)

		var output strings.Builder
		output.WriteString(fmt.Sprintf("Available versions for %s:\n\n", appName))
		
		for i, entry := range sorted {
			output.WriteString(fmt.Sprintf("%d. Version: %s (App: %s)\n", i+1, entry.GetLatestVersion(), entry.GetAppVersion()))
			output.WriteString(fmt.Sprintf("   Entry: %s/%s\n", entry.Namespace, entry.Name))
			output.WriteString(fmt.Sprintf("   Catalog: %s/%s\n", entry.Spec.Catalog.Namespace, entry.Spec.Catalog.Name))
			if entry.Spec.DateUpdated != nil {
				output.WriteString(fmt.Sprintf("   Updated: %s\n", entry.Spec.DateUpdated.Format("2006-01-02")))
			} else if entry.Spec.DateCreated != nil {
				output.WriteString(fmt.Sprintf("   Created: %s\n", entry.Spec.DateCreated.Format("2006-01-02")))
			}
			if i == 0 {
				output.WriteString("   (Latest)\n")
			}
			output.WriteString("\n")
		}

		return mcp.NewToolResultText(output.String()), nil
	})

	return nil
} 