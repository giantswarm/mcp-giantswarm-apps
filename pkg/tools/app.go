package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/giantswarm/mcp-giantswarm-apps/internal/server"
	"github.com/giantswarm/mcp-giantswarm-apps/pkg/app"
	"github.com/giantswarm/mcp-giantswarm-apps/pkg/organization"
)

// RegisterAppTools registers all app management tools
func RegisterAppTools(s *mcpserver.MCPServer, ctx *server.Context) error {
	appClient := app.NewClient(ctx.DynamicClient)

	// app_list tool
	listTool := mcp.NewTool(
		"app_list",
		mcp.WithDescription("List Giant Swarm apps with optional filtering"),
		mcp.WithString("namespace", mcp.Description("Namespace to list apps from (empty for all namespaces)")),
		mcp.WithString("organization", mcp.Description("Organization to list apps from (e.g., 'giantswarm')")),
		mcp.WithString("labels", mcp.Description("Label selector (e.g., 'app=nginx,env=prod')")),
		mcp.WithString("status", mcp.Description("Filter by release status (deployed, failed, pending, etc.)")),
		mcp.WithString("catalog", mcp.Description("Filter by catalog name")),
		mcp.WithBoolean("all-orgs", mcp.Description("List apps from all organization namespaces")),
		mcp.WithBoolean("include-workload-clusters", mcp.Description("Include apps from workload cluster namespaces")),
	)

	s.AddTool(listTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})

		namespace := getStringArg(args, "namespace")
		org := getStringArg(args, "organization")
		labelSelector := getStringArg(args, "labels")
		status := getStringArg(args, "status")
		catalog := getStringArg(args, "catalog")
		allOrgs := getBoolArg(args, "all-orgs")
		includeWorkloadClusters := getBoolArg(args, "include-workload-clusters")

		var apps []*app.App
		var err error

		// Determine which namespaces to query
		if org != "" {
			// List apps from specific organization
			if includeWorkloadClusters {
				apps, err = appClient.ListByOrganization(toolCtx, ctx.K8sClient, org, labelSelector)
			} else {
				// Just the organization namespace
				orgNs := organization.GetOrganizationNamespace(org)
				apps, err = appClient.List(toolCtx, orgNs, labelSelector)
			}
			if err != nil {
				return nil, fmt.Errorf("failed to list apps for organization %s: %w", org, err)
			}
		} else if allOrgs && namespace == "" {
			// List from all organization namespaces
			orgNamespaces, err := appClient.GetOrganizationNamespaces(toolCtx, ctx.K8sClient)
			if err != nil {
				return nil, fmt.Errorf("failed to get organization namespaces: %w", err)
			}

			apps = make([]*app.App, 0)
			for _, ns := range orgNamespaces {
				nsApps, err := appClient.List(toolCtx, ns, labelSelector)
				if err != nil {
					continue // Skip namespaces with errors
				}
				apps = append(apps, nsApps...)
			}
		} else {
			// List from specific namespace or all namespaces
			apps, err = appClient.List(toolCtx, namespace, labelSelector)
			if err != nil {
				return nil, err
			}
		}

		// Apply filters
		apps = app.FilterByStatus(apps, status)
		apps = app.FilterByCatalog(apps, catalog)

		// Format output
		if len(apps) == 0 {
			return mcp.NewToolResultText("No apps found"), nil
		}

		var output strings.Builder
		output.WriteString(fmt.Sprintf("Found %d apps:\n\n", len(apps)))

		for _, a := range apps {
			output.WriteString(fmt.Sprintf("Name: %s\n", a.Name))
			output.WriteString(fmt.Sprintf("Namespace: %s\n", a.Namespace))
			output.WriteString(fmt.Sprintf("App: %s (v%s)\n", a.Spec.Name, a.Spec.Version))
			output.WriteString(fmt.Sprintf("Catalog: %s\n", a.Spec.Catalog))
			output.WriteString(fmt.Sprintf("Target Namespace: %s\n", a.Spec.Namespace))
			output.WriteString(fmt.Sprintf("Status: %s\n", a.Status.Release.Status))
			if a.Status.Release.LastDeployed != "" {
				output.WriteString(fmt.Sprintf("Last Deployed: %s\n", a.Status.Release.LastDeployed))
			}
			output.WriteString("---\n")
		}

		return mcp.NewToolResultText(output.String()), nil
	})

	// app_get tool
	getTool := mcp.NewTool(
		"app_get",
		mcp.WithDescription("Get detailed information about a specific app"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the app")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the app")),
	)

	s.AddTool(getTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		name := args["name"].(string)
		namespace := args["namespace"].(string)

		app, err := appClient.Get(toolCtx, namespace, name)
		if err != nil {
			return nil, err
		}

		var output strings.Builder
		output.WriteString(fmt.Sprintf("App: %s\n", app.Name))
		output.WriteString(fmt.Sprintf("Namespace: %s\n", app.Namespace))
		output.WriteString("\nSpec:\n")
		output.WriteString(fmt.Sprintf("  Catalog: %s\n", app.Spec.Catalog))
		output.WriteString(fmt.Sprintf("  App Name: %s\n", app.Spec.Name))
		output.WriteString(fmt.Sprintf("  Version: %s\n", app.Spec.Version))
		output.WriteString(fmt.Sprintf("  Target Namespace: %s\n", app.Spec.Namespace))
		output.WriteString(fmt.Sprintf("  In-Cluster: %v\n", app.Spec.KubeConfig.InCluster))

		if app.Spec.Config != nil {
			output.WriteString("\nConfiguration:\n")
			if app.Spec.Config.ConfigMap != nil {
				output.WriteString(fmt.Sprintf("  ConfigMap: %s/%s\n",
					app.Spec.Config.ConfigMap.Namespace, app.Spec.Config.ConfigMap.Name))
			}
			if app.Spec.Config.Secret != nil {
				output.WriteString(fmt.Sprintf("  Secret: %s/%s\n",
					app.Spec.Config.Secret.Namespace, app.Spec.Config.Secret.Name))
			}
		}

		if app.Spec.UserConfig != nil {
			output.WriteString("\nUser Configuration:\n")
			if app.Spec.UserConfig.ConfigMap != nil {
				output.WriteString(fmt.Sprintf("  ConfigMap: %s/%s\n",
					app.Spec.UserConfig.ConfigMap.Namespace, app.Spec.UserConfig.ConfigMap.Name))
			}
			if app.Spec.UserConfig.Secret != nil {
				output.WriteString(fmt.Sprintf("  Secret: %s/%s\n",
					app.Spec.UserConfig.Secret.Namespace, app.Spec.UserConfig.Secret.Name))
			}
		}

		output.WriteString("\nStatus:\n")
		output.WriteString(fmt.Sprintf("  App Version: %s\n", app.Status.AppVersion))
		output.WriteString(fmt.Sprintf("  Chart Version: %s\n", app.Status.Version))
		output.WriteString(fmt.Sprintf("  Release Status: %s\n", app.Status.Release.Status))
		if app.Status.Release.LastDeployed != "" {
			output.WriteString(fmt.Sprintf("  Last Deployed: %s\n", app.Status.Release.LastDeployed))
		}

		return mcp.NewToolResultText(output.String()), nil
	})

	// app_create tool
	createTool := mcp.NewTool(
		"app_create",
		mcp.WithDescription("Create a new Giant Swarm app"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name for the app resource")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace to create the app in")),
		mcp.WithString("catalog", mcp.Required(), mcp.Description("Catalog name (e.g., giantswarm)")),
		mcp.WithString("app", mcp.Required(), mcp.Description("App name from catalog (e.g., nginx-ingress-controller)")),
		mcp.WithString("version", mcp.Required(), mcp.Description("App version")),
		mcp.WithString("target-namespace", mcp.Description("Target namespace for the app (defaults to app name)")),
		mcp.WithBoolean("in-cluster", mcp.Description("Deploy to management cluster (default: true)")),
		mcp.WithString("config-name", mcp.Description("Name of the ConfigMap for configuration")),
		mcp.WithString("user-config-name", mcp.Description("Name of the ConfigMap for user configuration")),
	)

	s.AddTool(createTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})

		name := args["name"].(string)
		namespace := args["namespace"].(string)
		catalog := args["catalog"].(string)
		appName := args["app"].(string)
		version := args["version"].(string)
		targetNamespace := getStringArg(args, "target-namespace")
		if targetNamespace == "" {
			targetNamespace = appName
		}

		inCluster := true
		if val, ok := args["in-cluster"].(bool); ok {
			inCluster = val
		}

		newApp := &app.App{
			Name:      name,
			Namespace: namespace,
			Spec: app.AppSpec{
				Catalog:   catalog,
				Name:      appName,
				Namespace: targetNamespace,
				Version:   version,
				KubeConfig: app.KubeConfig{
					InCluster: inCluster,
				},
			},
		}

		// Add config references if provided
		configName := getStringArg(args, "config-name")
		if configName != "" {
			newApp.Spec.Config = &app.AppConfig{
				ConfigMap: &app.ConfigMapReference{
					Name:      configName,
					Namespace: namespace,
				},
			}
		}

		userConfigName := getStringArg(args, "user-config-name")
		if userConfigName != "" {
			newApp.Spec.UserConfig = &app.AppConfig{
				ConfigMap: &app.ConfigMapReference{
					Name:      userConfigName,
					Namespace: namespace,
				},
			}
		}

		created, err := appClient.Create(toolCtx, newApp)
		if err != nil {
			return nil, err
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully created app %s/%s", created.Namespace, created.Name)), nil
	})

	// app_update tool
	updateTool := mcp.NewTool(
		"app_update",
		mcp.WithDescription("Update an existing Giant Swarm app"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the app")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the app")),
		mcp.WithString("version", mcp.Description("New version to update to")),
		mcp.WithString("config-name", mcp.Description("Update ConfigMap name")),
		mcp.WithString("user-config-name", mcp.Description("Update user ConfigMap name")),
	)

	s.AddTool(updateTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		name := args["name"].(string)
		namespace := args["namespace"].(string)

		// Get current app
		currentApp, err := appClient.Get(toolCtx, namespace, name)
		if err != nil {
			return nil, err
		}

		// Update version if provided
		if version := getStringArg(args, "version"); version != "" {
			currentApp.Spec.Version = version
		}

		// Update config if provided
		if configName := getStringArg(args, "config-name"); configName != "" {
			if currentApp.Spec.Config == nil {
				currentApp.Spec.Config = &app.AppConfig{}
			}
			if currentApp.Spec.Config.ConfigMap == nil {
				currentApp.Spec.Config.ConfigMap = &app.ConfigMapReference{}
			}
			currentApp.Spec.Config.ConfigMap.Name = configName
			currentApp.Spec.Config.ConfigMap.Namespace = namespace
		}

		// Update user config if provided
		if userConfigName := getStringArg(args, "user-config-name"); userConfigName != "" {
			if currentApp.Spec.UserConfig == nil {
				currentApp.Spec.UserConfig = &app.AppConfig{}
			}
			if currentApp.Spec.UserConfig.ConfigMap == nil {
				currentApp.Spec.UserConfig.ConfigMap = &app.ConfigMapReference{}
			}
			currentApp.Spec.UserConfig.ConfigMap.Name = userConfigName
			currentApp.Spec.UserConfig.ConfigMap.Namespace = namespace
		}

		updated, err := appClient.Update(toolCtx, currentApp)
		if err != nil {
			return nil, err
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully updated app %s/%s", updated.Namespace, updated.Name)), nil
	})

	// app_delete tool
	deleteTool := mcp.NewTool(
		"app_delete",
		mcp.WithDescription("Delete a Giant Swarm app"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the app")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the app")),
	)

	s.AddTool(deleteTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		name := args["name"].(string)
		namespace := args["namespace"].(string)

		err := appClient.Delete(toolCtx, namespace, name)
		if err != nil {
			return nil, err
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully deleted app %s/%s", namespace, name)), nil
	})

	return nil
}

// Helper functions
func getStringArg(args map[string]interface{}, key string) string {
	if val, ok := args[key].(string); ok {
		return val
	}
	return ""
}

func getBoolArg(args map[string]interface{}, key string) bool {
	if val, ok := args[key].(bool); ok {
		return val
	}
	return false
}
