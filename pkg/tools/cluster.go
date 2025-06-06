package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/giantswarm/mcp-giantswarm-apps/internal/server"
	"github.com/giantswarm/mcp-giantswarm-apps/pkg/app"
	"github.com/giantswarm/mcp-giantswarm-apps/pkg/cluster"
)

// RegisterClusterTools registers all cluster management tools
func RegisterClusterTools(s *mcpserver.MCPServer, ctx *server.Context) error {
	appClient := app.NewClient(ctx.DynamicClient)
	clusterClient := cluster.NewClient(ctx.DynamicClient, ctx.K8sClient, appClient)

	// cluster_list tool
	listTool := mcp.NewTool(
		"cluster_list",
		mcp.WithDescription("List available workload clusters"),
		mcp.WithString("namespace", mcp.Description("Namespace to list clusters from (empty for all namespaces)")),
		mcp.WithString("organization", mcp.Description("Organization to list clusters from")),
		mcp.WithString("labels", mcp.Description("Label selector (e.g., 'provider=aws,env=prod')")),
		mcp.WithString("provider", mcp.Description("Filter by infrastructure provider (aws, azure, etc.)")),
		mcp.WithBoolean("ready-only", mcp.Description("Show only ready clusters")),
	)

	s.AddTool(listTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})

		namespace := getStringArg(args, "namespace")
		org := getStringArg(args, "organization")
		labelSelector := getStringArg(args, "labels")
		provider := getStringArg(args, "provider")
		readyOnly := getBoolArg(args, "ready-only")

		var clusters []*cluster.Cluster
		var err error

		if org != "" {
			// List clusters for specific organization
			clusters, err = clusterClient.ListByOrganization(toolCtx, org)
			if err != nil {
				return nil, fmt.Errorf("failed to list clusters for organization %s: %w", org, err)
			}
		} else {
			// List clusters from namespace or all namespaces
			clusters, err = clusterClient.List(toolCtx, namespace, labelSelector)
			if err != nil {
				return nil, err
			}
		}

		// Apply filters
		clusters = cluster.FilterByProvider(clusters, provider)
		if readyOnly {
			clusters = cluster.FilterByStatus(clusters, true)
		}

		// Format output
		if len(clusters) == 0 {
			return mcp.NewToolResultText("No clusters found"), nil
		}

		var output strings.Builder
		output.WriteString(fmt.Sprintf("Found %d clusters:\n\n", len(clusters)))

		for _, c := range clusters {
			output.WriteString(fmt.Sprintf("Name: %s\n", c.Name))
			output.WriteString(fmt.Sprintf("Namespace: %s\n", c.Namespace))
			output.WriteString(fmt.Sprintf("Organization: %s\n", c.GetOrganization()))
			output.WriteString(fmt.Sprintf("Provider: %s\n", c.GetProvider()))
			output.WriteString(fmt.Sprintf("Status: %s\n", c.Status.Phase))
			output.WriteString(fmt.Sprintf("Ready: %v\n", c.IsReady()))

			if c.Status.InfrastructureReady {
				output.WriteString("Infrastructure: Ready\n")
			} else {
				output.WriteString("Infrastructure: Not Ready\n")
			}

			if c.Status.ControlPlaneReady {
				output.WriteString("Control Plane: Ready\n")
			} else {
				output.WriteString("Control Plane: Not Ready\n")
			}

			// Show conditions if any
			if len(c.Status.Conditions) > 0 {
				output.WriteString("Conditions:\n")
				for _, cond := range c.Status.Conditions {
					output.WriteString(fmt.Sprintf("  - %s: %s", cond.Type, cond.Status))
					if cond.Reason != "" {
						output.WriteString(fmt.Sprintf(" (%s)", cond.Reason))
					}
					output.WriteString("\n")
				}
			}

			output.WriteString("---\n")
		}

		return mcp.NewToolResultText(output.String()), nil
	})

	// cluster_apps tool
	appsTool := mcp.NewTool(
		"cluster_apps",
		mcp.WithDescription("List apps in a specific cluster"),
		mcp.WithString("cluster", mcp.Required(), mcp.Description("Cluster name")),
		mcp.WithString("namespace", mcp.Description("Namespace where the cluster is located")),
		mcp.WithString("organization", mcp.Description("Organization that owns the cluster")),
	)

	s.AddTool(appsTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		clusterName := args["cluster"].(string)
		namespace := getStringArg(args, "namespace")
		org := getStringArg(args, "organization")

		// Find the cluster
		var targetCluster *cluster.Cluster
		var err error

		if namespace != "" {
			targetCluster, err = clusterClient.Get(toolCtx, namespace, clusterName)
		} else if org != "" {
			// Search in organization namespaces
			clusters, err := clusterClient.ListByOrganization(toolCtx, org)
			if err == nil {
				for _, c := range clusters {
					if c.Name == clusterName {
						targetCluster = c
						break
					}
				}
			}
		} else {
			// Search across all namespaces
			clusters, err := clusterClient.List(toolCtx, "", "")
			if err == nil {
				for _, c := range clusters {
					if c.Name == clusterName {
						targetCluster = c
						break
					}
				}
			}
		}

		if targetCluster == nil {
			return nil, fmt.Errorf("cluster %s not found", clusterName)
		}

		// List apps in the cluster
		apps, err := clusterClient.ListApps(toolCtx, targetCluster)
		if err != nil {
			return nil, fmt.Errorf("failed to list apps in cluster %s: %w", clusterName, err)
		}

		// Format output
		if len(apps) == 0 {
			return mcp.NewToolResultText(fmt.Sprintf("No apps found in cluster %s", clusterName)), nil
		}

		var output strings.Builder
		output.WriteString(fmt.Sprintf("Apps in cluster %s:\n\n", clusterName))

		for _, a := range apps {
			output.WriteString(fmt.Sprintf("Name: %s\n", a.Name))
			output.WriteString(fmt.Sprintf("Namespace: %s\n", a.Namespace))
			output.WriteString(fmt.Sprintf("App: %s (v%s)\n", a.Spec.Name, a.Spec.Version))
			output.WriteString(fmt.Sprintf("Catalog: %s\n", a.Spec.Catalog))
			output.WriteString(fmt.Sprintf("Status: %s\n", a.Status.Release.Status))
			output.WriteString("---\n")
		}

		return mcp.NewToolResultText(output.String()), nil
	})

	// cluster_get tool
	getTool := mcp.NewTool(
		"cluster_get",
		mcp.WithDescription("Get detailed information about a specific cluster"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Cluster name")),
		mcp.WithString("namespace", mcp.Description("Namespace where the cluster is located")),
		mcp.WithString("organization", mcp.Description("Organization that owns the cluster")),
	)

	s.AddTool(getTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		clusterName := args["name"].(string)
		namespace := getStringArg(args, "namespace")
		org := getStringArg(args, "organization")

		var targetCluster *cluster.Cluster
		var err error

		if namespace != "" {
			targetCluster, err = clusterClient.Get(toolCtx, namespace, clusterName)
			if err != nil {
				return nil, err
			}
		} else if org != "" {
			// Search in organization namespaces
			clusters, err := clusterClient.ListByOrganization(toolCtx, org)
			if err != nil {
				return nil, err
			}
			for _, c := range clusters {
				if c.Name == clusterName {
					targetCluster = c
					break
				}
			}
		} else {
			// Search across all namespaces
			clusters, err := clusterClient.List(toolCtx, "", "")
			if err != nil {
				return nil, err
			}
			for _, c := range clusters {
				if c.Name == clusterName {
					targetCluster = c
					break
				}
			}
		}

		if targetCluster == nil {
			return nil, fmt.Errorf("cluster %s not found", clusterName)
		}

		// Format detailed output
		var output strings.Builder
		output.WriteString(fmt.Sprintf("Cluster: %s\n", targetCluster.Name))
		output.WriteString(fmt.Sprintf("Namespace: %s\n", targetCluster.Namespace))
		output.WriteString(fmt.Sprintf("Organization: %s\n", targetCluster.GetOrganization()))
		output.WriteString(fmt.Sprintf("Provider: %s\n", targetCluster.GetProvider()))
		output.WriteString(fmt.Sprintf("Type: %s\n", func() string {
			if clusterClient.IsWorkloadCluster(targetCluster) {
				return "Workload"
			}
			return "Management"
		}()))

		output.WriteString("\nSpec:\n")
		if targetCluster.Spec.InfrastructureRef != nil {
			output.WriteString(fmt.Sprintf("  Infrastructure: %s/%s\n",
				targetCluster.Spec.InfrastructureRef.Kind,
				targetCluster.Spec.InfrastructureRef.Name))
		}
		if targetCluster.Spec.ControlPlaneRef != nil {
			output.WriteString(fmt.Sprintf("  Control Plane: %s/%s\n",
				targetCluster.Spec.ControlPlaneRef.Kind,
				targetCluster.Spec.ControlPlaneRef.Name))
		}
		if targetCluster.Spec.ClusterNetwork != nil {
			output.WriteString("  Network:\n")
			if targetCluster.Spec.ClusterNetwork.Services != nil && len(targetCluster.Spec.ClusterNetwork.Services.CIDRBlocks) > 0 {
				output.WriteString(fmt.Sprintf("    Services: %s\n", strings.Join(targetCluster.Spec.ClusterNetwork.Services.CIDRBlocks, ", ")))
			}
			if targetCluster.Spec.ClusterNetwork.Pods != nil && len(targetCluster.Spec.ClusterNetwork.Pods.CIDRBlocks) > 0 {
				output.WriteString(fmt.Sprintf("    Pods: %s\n", strings.Join(targetCluster.Spec.ClusterNetwork.Pods.CIDRBlocks, ", ")))
			}
		}

		output.WriteString("\nStatus:\n")
		output.WriteString(fmt.Sprintf("  Phase: %s\n", targetCluster.Status.Phase))
		output.WriteString(fmt.Sprintf("  Infrastructure Ready: %v\n", targetCluster.Status.InfrastructureReady))
		output.WriteString(fmt.Sprintf("  Control Plane Ready: %v\n", targetCluster.Status.ControlPlaneReady))

		if len(targetCluster.Status.Conditions) > 0 {
			output.WriteString("\nConditions:\n")
			for _, cond := range targetCluster.Status.Conditions {
				output.WriteString(fmt.Sprintf("  %s: %s\n", cond.Type, cond.Status))
				if cond.Reason != "" {
					output.WriteString(fmt.Sprintf("    Reason: %s\n", cond.Reason))
				}
				if cond.Message != "" {
					output.WriteString(fmt.Sprintf("    Message: %s\n", cond.Message))
				}
				if cond.LastTransitionTime != "" {
					output.WriteString(fmt.Sprintf("    Last Transition: %s\n", cond.LastTransitionTime))
				}
			}
		}

		// Check for kubeconfig availability
		_, kubeconfigErr := clusterClient.GetKubeconfig(toolCtx, targetCluster)
		if kubeconfigErr == nil {
			output.WriteString("\nKubeconfig: Available\n")
			output.WriteString(fmt.Sprintf("  Secret: %s-kubeconfig\n", targetCluster.Name))
		} else {
			output.WriteString("\nKubeconfig: Not Available\n")
		}

		// Show workload namespace
		workloadNs := cluster.GetClusterNamespace(targetCluster.Name)
		output.WriteString(fmt.Sprintf("\nWorkload Namespace: %s\n", workloadNs))

		// Show labels if any
		if len(targetCluster.Labels) > 0 {
			output.WriteString("\nLabels:\n")
			for k, v := range targetCluster.Labels {
				output.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
			}
		}

		return mcp.NewToolResultText(output.String()), nil
	})

	return nil
}
