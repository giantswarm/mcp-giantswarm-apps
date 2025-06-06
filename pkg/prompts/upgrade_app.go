package prompts

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/giantswarm/mcp-giantswarm-apps/internal/server"
)

func registerUpgradeAppPrompt(s *mcpserver.MCPServer, ctx *server.Context) error {
	prompt := mcp.NewPrompt(
		"upgrade-app",
		mcp.WithPromptDescription("Guide for upgrading a Giant Swarm app to a new version"),
		mcp.WithArgument("name", mcp.ArgumentDescription("Name of the app to upgrade")),
		mcp.WithArgument("namespace", mcp.ArgumentDescription("Namespace where the app is deployed")),
		mcp.WithArgument("version", mcp.ArgumentDescription("Target version to upgrade to")),
	)

	s.AddPrompt(prompt, func(promptCtx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		args := req.Params.Arguments

		appName := args["name"]
		namespace := args["namespace"]
		targetVersion := args["version"]

		pb := newPromptBuilder()

		// Title and overview
		pb.addSection("Upgrade Giant Swarm App",
			"This guide will help you safely upgrade a Giant Swarm app to a new version. "+
				"Follow these steps to ensure a smooth upgrade process.")

		// Step 1: App identification
		if appName == "" || namespace == "" {
			pb.addSection("Step 1: Identify the App",
				"First, identify the app you want to upgrade. List all apps to find the correct one:")
			pb.addCodeBlock("List Apps", "bash", "app.list")
			pb.addSection("Action Required",
				"Please specify both 'name' and 'namespace' arguments for the app you want to upgrade.")
			return &mcp.GetPromptResult{
				Description: "Upgrade app guide - app identification needed",
				Messages: []mcp.PromptMessage{
					{
						Role:    mcp.RoleUser,
						Content: mcp.TextContent{Text: pb.build()},
					},
				},
			}, nil
		}

		pb.addSection("App Information",
			fmt.Sprintf("Upgrading app: **%s** in namespace: **%s**", appName, namespace))

		// Step 2: Current status check
		pb.addSection("Step 2: Check Current Status",
			"Before upgrading, check the current status and version of your app:")
		pb.addCodeBlock("Get App Details", "bash",
			fmt.Sprintf("app.get --name %s --namespace %s", appName, namespace))
		pb.addList("What to Check", []string{
			"Current version",
			"Release status (should be 'deployed')",
			"Any existing errors or warnings",
			"Last deployment time",
		})

		// Step 3: Version selection
		if targetVersion == "" {
			pb.addSection("Step 3: Select Target Version",
				"Check available versions for the app:")
			pb.addSection("Find Available Versions",
				"First, identify the catalog and app name from the current app details, then:")
			pb.addCodeBlock("List Versions", "bash",
				"appcatalogentry.get --catalog <CATALOG> --name <APP_NAME>")
			pb.addList("Version Selection Guidelines", []string{
				"Check the changelog for breaking changes",
				"Prefer incremental upgrades over major jumps",
				"Verify version compatibility with your cluster",
				"Consider testing in a non-production environment first",
			})
			pb.addSection("Action Required",
				"Please specify the target 'version' argument.")
			return &mcp.GetPromptResult{
				Description: "Upgrade app guide - version selection needed",
				Messages: []mcp.PromptMessage{
					{
						Role:    mcp.RoleUser,
						Content: mcp.TextContent{Text: pb.build()},
					},
				},
			}, nil
		}

		pb.addSection("Target Version", fmt.Sprintf("Upgrading to version: **%s**", targetVersion))

		// Step 4: Pre-upgrade checklist
		pb.addSection("Step 4: Pre-Upgrade Checklist",
			"Complete these checks before proceeding:")
		pb.addList("Checklist", []string{
			"✓ Review the changelog for version " + targetVersion,
			"✓ Check for breaking changes or migration requirements",
			"✓ Backup any important data or configurations",
			"✓ Verify cluster has sufficient resources",
			"✓ Plan a maintenance window if needed",
			"✓ Prepare rollback plan",
		})

		// Step 5: Configuration review
		pb.addSection("Step 5: Review Configuration",
			"Check if configuration changes are needed for the new version:")
		pb.addCodeBlock("View Current Config", "bash",
			fmt.Sprintf("config.get --namespace %s --app %s", namespace, appName))
		pb.addSection("Configuration Compatibility",
			"Compare your current configuration with the new version's schema:")
		pb.addCodeBlock("Check New Schema", "bash",
			fmt.Sprintf("config.schema --catalog <CATALOG> --app <APP_NAME> --version %s", targetVersion))

		// Step 6: Perform upgrade
		pb.addSection("Step 6: Perform the Upgrade",
			"Execute the upgrade command:")
		pb.addCodeBlock("Upgrade Command", "bash",
			fmt.Sprintf("app.update --name %s --namespace %s --version %s",
				appName, namespace, targetVersion))

		// Step 7: Monitor upgrade
		pb.addSection("Step 7: Monitor the Upgrade",
			"After initiating the upgrade, monitor its progress:")
		pb.addCodeBlock("Check Status", "bash",
			fmt.Sprintf("app.get --name %s --namespace %s", appName, namespace))
		pb.addList("Monitor These Aspects", []string{
			"Release status transitions",
			"Pod rollout status",
			"Any error messages",
			"Resource utilization",
		})

		// Step 8: Verification
		pb.addSection("Step 8: Verify the Upgrade",
			"Once the upgrade completes, verify everything is working:")
		pb.addList("Verification Steps", []string{
			"Check app status shows 'deployed'",
			"Verify new version is running",
			"Test app functionality",
			"Check logs for any errors",
			"Monitor metrics and performance",
		})

		// Rollback procedure
		pb.addSection("Rollback (If Needed)",
			"If issues occur, you can rollback to the previous version:")
		pb.addCodeBlock("Rollback Command", "bash",
			fmt.Sprintf("app.update --name %s --namespace %s --version <PREVIOUS_VERSION>",
				appName, namespace))

		// Best practices
		pb.addList("Upgrade Best Practices", []string{
			"Always test upgrades in non-production first",
			"Document the upgrade process",
			"Monitor the app for 24-48 hours post-upgrade",
			"Keep track of configuration changes",
			"Maintain a rollback plan",
		})

		return &mcp.GetPromptResult{
			Description: fmt.Sprintf("Complete guide to upgrade %s to version %s", appName, targetVersion),
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
