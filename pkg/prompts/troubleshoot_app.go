package prompts

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/giantswarm/mcp-giantswarm-apps/internal/server"
)

func registerTroubleshootAppPrompt(s *mcpserver.MCPServer, ctx *server.Context) error {
	prompt := mcp.NewPrompt(
		"troubleshoot-app",
		mcp.WithPromptDescription("Troubleshooting guide for Giant Swarm app issues"),
		mcp.WithArgument("name", mcp.ArgumentDescription("Name of the app having issues")),
		mcp.WithArgument("namespace", mcp.ArgumentDescription("Namespace where the app is deployed")),
		mcp.WithArgument("issue", mcp.ArgumentDescription("Type of issue: deployment, configuration, performance, or general")),
	)

	s.AddPrompt(prompt, func(promptCtx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		args := req.Params.Arguments

		appName := args["name"]
		namespace := args["namespace"]
		issueType := args["issue"]

		pb := newPromptBuilder()

		// Title and overview
		pb.addSection("Troubleshoot Giant Swarm App",
			"This guide will help you diagnose and resolve common issues with Giant Swarm apps. "+
				"Follow the diagnostic steps to identify and fix problems.")

		// App identification
		if appName == "" || namespace == "" {
			pb.addSection("Identify the App",
				"To troubleshoot effectively, we need to identify the specific app:")
			pb.addCodeBlock("List Apps with Status", "bash",
				"app.list --all-orgs")
			pb.addSection("Look For",
				"Apps with status other than 'deployed', such as:\n"+
					"- failed\n"+
					"- pending\n"+
					"- unknown")
			pb.addSection("Action Required",
				"Please specify 'name' and 'namespace' arguments for the app to troubleshoot.")
			return &mcp.GetPromptResult{
				Description: "Troubleshooting guide - app identification needed",
				Messages: []mcp.PromptMessage{
					{
						Role:    mcp.RoleUser,
						Content: mcp.TextContent{Text: pb.build()},
					},
				},
			}, nil
		}

		pb.addSection("App Details",
			fmt.Sprintf("Troubleshooting: **%s** in namespace: **%s**", appName, namespace))

		// Step 1: Current status
		pb.addSection("Step 1: Check App Status",
			"First, get detailed information about the app:")
		pb.addCodeBlock("Get App Details", "bash",
			fmt.Sprintf("app.get --name %s --namespace %s", appName, namespace))
		pb.addList("Key Information to Note", []string{
			"Release status",
			"Current version",
			"Last deployment time",
			"Any error messages",
			"Configuration references",
		})

		// Issue-specific troubleshooting
		if issueType == "deployment" || issueType == "" {
			pb.addSection("Deployment Issues",
				"If the app is not deploying successfully:")

			pb.addSection("Check 1: Release Status",
				"Common deployment statuses and their meanings:")
			pb.addList("Status Guide", []string{
				"**pending** - Deployment in progress, wait a few minutes",
				"**failed** - Deployment failed, check error messages",
				"**unknown** - Status cannot be determined",
				"**deployed** - Successfully deployed (no issue)",
			})

			pb.addSection("Check 2: Configuration Issues",
				"Verify configuration is correct:")
			pb.addCodeBlock("Check Config", "bash",
				fmt.Sprintf("config.get --namespace %s --app %s", namespace, appName))
			pb.addList("Common Config Issues", []string{
				"Missing required configuration values",
				"Invalid YAML syntax",
				"Incorrect value types",
				"Referenced ConfigMap/Secret doesn't exist",
			})

			pb.addSection("Check 3: Catalog and Version",
				"Ensure the app version exists in the catalog:")
			pb.addCodeBlock("Verify App in Catalog", "bash",
				"appcatalogentry.get --catalog <CATALOG> --name <APP_NAME>")

			pb.addSection("Check 4: Namespace Permissions",
				"Verify you have permissions in the namespace:")
			pb.addCodeBlock("Check Access", "bash",
				fmt.Sprintf("organization.validate-access --namespace %s", namespace))
		}

		if issueType == "configuration" || issueType == "" {
			pb.addSection("Configuration Issues",
				"For configuration-related problems:")

			pb.addSection("Validate Configuration",
				"Check if your configuration matches the schema:")
			pb.addCodeBlock("View Schema", "bash",
				"config.schema --catalog <CATALOG> --app <APP_NAME> --version <VERSION>")

			pb.addSection("Common Configuration Fixes", "")
			pb.addList("Steps to Fix", []string{
				"Compare your values with the schema",
				"Check for typos in configuration keys",
				"Ensure all required values are provided",
				"Validate value types (string vs number vs boolean)",
				"Check for deprecated configuration options",
			})
		}

		if issueType == "performance" || issueType == "" {
			pb.addSection("Performance Issues",
				"For apps experiencing performance problems:")

			pb.addList("Performance Checks", []string{
				"Check resource requests and limits in configuration",
				"Verify cluster has sufficient resources",
				"Look for pod restarts or evictions",
				"Check if horizontal pod autoscaling is configured",
				"Review app-specific metrics and logs",
			})

			pb.addSection("Resource Configuration",
				"Adjust resources if needed:")
			pb.addCodeBlock("Example Resource Config", "yaml",
				`resources:
  requests:
    memory: "256Mi"
    cpu: "100m"
  limits:
    memory: "512Mi"
    cpu: "200m"`)
		}

		// General troubleshooting steps
		pb.addSection("General Troubleshooting Steps", "")

		pb.addSection("1. Check Events",
			"Look for Kubernetes events related to the app:")
		pb.addCodeBlock("View Events", "bash",
			fmt.Sprintf("kubectl get events -n %s --field-selector involvedObject.name=%s",
				namespace, appName))

		pb.addSection("2. Check Logs",
			"If the app has pods running, check their logs:")
		pb.addCodeBlock("Get Pods", "bash",
			fmt.Sprintf("kubectl get pods -n %s -l app.kubernetes.io/name=%s",
				appName, appName))
		pb.addCodeBlock("View Logs", "bash",
			fmt.Sprintf("kubectl logs -n %s <POD_NAME>", appName))

		pb.addSection("3. Inspect Resources",
			"Check the actual Kubernetes resources:")
		pb.addCodeBlock("Describe App", "bash",
			fmt.Sprintf("kubectl describe app %s -n %s", appName, namespace))

		// Recovery actions
		pb.addSection("Recovery Actions", "")

		pb.addSection("Option 1: Reapply Configuration",
			"If configuration was the issue:")
		pb.addCodeBlock("Update Config", "bash",
			fmt.Sprintf("app.update --name %s --namespace %s --config-name <NEW_CONFIG>",
				appName, namespace))

		pb.addSection("Option 2: Force Redeploy",
			"Trigger a fresh deployment:")
		pb.addCodeBlock("Delete and Recreate", "bash",
			fmt.Sprintf(`# Delete the app
app.delete --name %s --namespace %s

# Recreate with same or updated configuration
app.create --name %s --namespace %s --catalog <CATALOG> --app <APP> --version <VERSION>`,
				appName, namespace, appName, namespace))

		pb.addSection("Option 3: Rollback Version",
			"If a recent upgrade caused issues:")
		pb.addCodeBlock("Rollback", "bash",
			fmt.Sprintf("app.update --name %s --namespace %s --version <PREVIOUS_VERSION>",
				appName, namespace))

		// Getting help
		pb.addSection("Need More Help?", "")
		pb.addList("Additional Resources", []string{
			"Check app-specific documentation",
			"Review Giant Swarm platform documentation",
			"Contact Giant Swarm support with gathered information",
			"Check if known issues exist for this app version",
		})

		pb.addSection("Information to Provide to Support", "")
		pb.addList("Gather This Information", []string{
			"App name, namespace, and version",
			"Error messages from 'app.get' command",
			"Kubernetes events",
			"Pod logs if available",
			"Configuration being used",
			"Time when issue started",
		})

		return &mcp.GetPromptResult{
			Description: fmt.Sprintf("Comprehensive troubleshooting guide for %s", appName),
			Messages: []mcp.PromptMessage{
				{
					Role:    mcp.RoleUser,
					Content: mcp.TextContent{Text: pb.build()},
				},
			},
		}, nil
	})

	return nil
}
