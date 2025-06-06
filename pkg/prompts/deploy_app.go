package prompts

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/giantswarm/mcp-giantswarm-apps/internal/server"
	"github.com/giantswarm/mcp-giantswarm-apps/pkg/organization"
)

func registerDeployAppPrompt(s *mcpserver.MCPServer, ctx *server.Context) error {
	prompt := mcp.NewPrompt(
		"deploy-app",
		mcp.WithPromptDescription("Step-by-step guide to deploy a Giant Swarm app"),
		mcp.WithArgument("organization", mcp.ArgumentDescription("Organization to deploy the app in (e.g., 'giantswarm')")),
		mcp.WithArgument("catalog", mcp.ArgumentDescription("Catalog name to browse apps from (e.g., 'giantswarm')")),
		mcp.WithArgument("app", mcp.ArgumentDescription("App name to deploy (e.g., 'nginx-ingress-controller')")),
		mcp.WithArgument("namespace", mcp.ArgumentDescription("Namespace to deploy the app in (defaults to organization namespace)")),
	)

	s.AddPrompt(prompt, func(promptCtx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		args := req.Params.Arguments

		orgName := args["organization"]
		catalogName := args["catalog"]
		appName := args["app"]
		namespace := args["namespace"]

		pb := newPromptBuilder()

		// Title and overview
		pb.addSection("Deploy Giant Swarm App",
			"This guide will help you deploy a Giant Swarm app to your Kubernetes cluster. "+
				"Follow the steps below to ensure a successful deployment.")

		// Step 1: Organization selection
		if orgName == "" {
			pb.addSection("Step 1: Select Organization",
				"First, you need to select which organization to deploy the app in. "+
					"Use the following command to list available organizations:")
			pb.addCodeBlock("List Organizations", "bash", "organization.list")
			pb.addSection("Action Required",
				"Please specify the organization using the 'organization' argument.")
			return &mcp.GetPromptResult{
				Description: "Deploy app guide - organization selection needed",
				Messages: []mcp.PromptMessage{
					{
						Role:    mcp.RoleUser,
						Content: mcp.TextContent{Text: pb.build()},
					},
				},
			}, nil
		}

		// Validate organization namespace
		orgNamespace := organization.GetOrganizationNamespace(orgName)
		if namespace == "" {
			namespace = orgNamespace
		}

		pb.addSection("Organization", fmt.Sprintf("Deploying to organization: **%s**", orgName))

		// Step 2: Catalog selection
		if catalogName == "" {
			pb.addSection("Step 2: Select Catalog",
				"Choose a catalog that contains the app you want to deploy. "+
					"Available catalogs can be listed with:")
			pb.addCodeBlock("List Catalogs", "bash",
				fmt.Sprintf("catalog.list --organization %s", orgName))
			pb.addList("Common Catalogs", []string{
				"giantswarm - Official Giant Swarm apps",
				"giantswarm-playground - Experimental apps",
				"giantswarm-incubator - Apps in development",
			})
			pb.addSection("Action Required",
				"Please specify the catalog using the 'catalog' argument.")
			return &mcp.GetPromptResult{
				Description: "Deploy app guide - catalog selection needed",
				Messages: []mcp.PromptMessage{
					{
						Role:    mcp.RoleUser,
						Content: mcp.TextContent{Text: pb.build()},
					},
				},
			}, nil
		}

		pb.addSection("Catalog", fmt.Sprintf("Using catalog: **%s**", catalogName))

		// Step 3: App selection
		if appName == "" {
			pb.addSection("Step 3: Select App",
				"Browse available apps in the catalog to find the one you want to deploy:")
			pb.addCodeBlock("Browse Apps", "bash",
				fmt.Sprintf("appcatalogentry.list --catalog %s", catalogName))
			pb.addSection("Popular Apps",
				"Some commonly deployed apps:\n"+
					"- nginx-ingress-controller - Ingress controller\n"+
					"- prometheus-operator - Monitoring stack\n"+
					"- external-dns - DNS management\n"+
					"- cert-manager - Certificate management")
			pb.addSection("Action Required",
				"Please specify the app using the 'app' argument.")
			return &mcp.GetPromptResult{
				Description: "Deploy app guide - app selection needed",
				Messages: []mcp.PromptMessage{
					{
						Role:    mcp.RoleUser,
						Content: mcp.TextContent{Text: pb.build()},
					},
				},
			}, nil
		}

		pb.addSection("App Selection", fmt.Sprintf("Deploying app: **%s**", appName))

		// Step 4: Version selection
		pb.addSection("Step 4: Select Version",
			"Check available versions for the app:")
		pb.addCodeBlock("Get App Details", "bash",
			fmt.Sprintf("appcatalogentry.get --catalog %s --name %s", catalogName, appName))

		// Step 5: Configuration
		pb.addSection("Step 5: Configuration (Optional)",
			"Many apps require or support configuration. You can:")
		pb.addList("Configuration Options", []string{
			"Use default configuration (no action needed)",
			"Create a ConfigMap with custom values",
			"Reference an existing ConfigMap",
		})
		pb.addSection("Check Configuration Schema",
			"To see available configuration options:")
		pb.addCodeBlock("View Schema", "bash",
			fmt.Sprintf("config.schema --catalog %s --app %s", catalogName, appName))

		// Step 6: Deploy command
		pb.addSection("Step 6: Deploy the App",
			"Use the following command to deploy the app:")

		deployCmd := fmt.Sprintf(`app.create \
  --name %s \
  --namespace %s \
  --catalog %s \
  --app %s \
  --version <VERSION> \
  --target-namespace %s`,
			appName,     // default name to app name
			namespace,   // namespace to create the App CR in
			catalogName, // catalog name
			appName,     // app to deploy
			appName)     // default target namespace to app name

		pb.addCodeBlock("Deploy Command", "bash", deployCmd)

		// Step 7: Verification
		pb.addSection("Step 7: Verify Deployment",
			"After deployment, verify the app status:")
		pb.addCodeBlock("Check Status", "bash",
			fmt.Sprintf("app.get --namespace %s --name %s", namespace, appName))

		// Best practices
		pb.addList("Best Practices", []string{
			"Always specify a version instead of using 'latest'",
			"Review the app's documentation before deployment",
			"Test in a non-production environment first",
			"Use configuration management for custom values",
			"Monitor the app after deployment",
		})

		// Troubleshooting
		pb.addSection("Need Help?",
			"If you encounter issues, use the troubleshooting guide:")
		pb.addCodeBlock("Troubleshooting", "text",
			"Run prompt: troubleshoot-app")

		return &mcp.GetPromptResult{
			Description: fmt.Sprintf("Complete guide to deploy %s from %s catalog", appName, catalogName),
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
