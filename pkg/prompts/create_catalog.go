package prompts

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/giantswarm/mcp-giantswarm-apps/internal/server"
)

func registerCreateCatalogPrompt(s *mcpserver.MCPServer, ctx *server.Context) error {
	prompt := mcp.NewPrompt(
		"create-catalog",
		mcp.WithPromptDescription("Guide to create a custom Giant Swarm app catalog"),
		mcp.WithArgument("name", mcp.ArgumentDescription("Name for the new catalog")),
		mcp.WithArgument("organization", mcp.ArgumentDescription("Organization to create the catalog in")),
		mcp.WithArgument("type", mcp.ArgumentDescription("Catalog type: helm or oci")),
		mcp.WithArgument("visibility", mcp.ArgumentDescription("Catalog visibility: public or private")),
	)

	s.AddPrompt(prompt, func(promptCtx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		args := req.Params.Arguments

		catalogName := args["name"]
		organization := args["organization"]
		catalogType := args["type"]
		visibility := args["visibility"]

		pb := newPromptBuilder()

		// Title and overview
		pb.addSection("Create Custom App Catalog",
			"This guide will help you create a custom catalog for your Giant Swarm apps. "+
				"Custom catalogs allow you to manage and deploy your own applications.")

		// Step 1: Planning
		pb.addSection("Step 1: Plan Your Catalog",
			"Before creating a catalog, consider these aspects:")
		pb.addList("Planning Considerations", []string{
			"Purpose: Internal apps, testing, or production use",
			"Repository type: Helm chart repository or OCI registry",
			"Access control: Public or private visibility",
			"Organization: Which organization will own this catalog",
			"Naming convention: Use descriptive, lowercase names",
		})

		// Check required inputs
		if catalogName == "" {
			pb.addSection("Catalog Name",
				"Choose a descriptive name for your catalog:")
			pb.addList("Naming Guidelines", []string{
				"Use lowercase letters, numbers, and hyphens",
				"Be descriptive (e.g., 'mycompany-apps', 'team-x-testing')",
				"Avoid generic names like 'apps' or 'catalog'",
				"Consider including organization or team name",
			})
			pb.addSection("Action Required",
				"Please specify the catalog 'name' argument.")
			return &mcp.GetPromptResult{
				Description: "Create catalog guide - name required",
				Messages: []mcp.PromptMessage{
					{
						Role:    mcp.RoleUser,
						Content: mcp.TextContent{Text: pb.build()},
					},
				},
			}, nil
		}

		if organization == "" {
			pb.addSection("Select Organization",
				"Catalogs must be created within an organization namespace:")
			pb.addCodeBlock("List Organizations", "bash", "organization.list")
			pb.addSection("Action Required",
				"Please specify the 'organization' argument.")
			return &mcp.GetPromptResult{
				Description: "Create catalog guide - organization required",
				Messages: []mcp.PromptMessage{
					{
						Role:    mcp.RoleUser,
						Content: mcp.TextContent{Text: pb.build()},
					},
				},
			}, nil
		}

		// Set defaults
		if catalogType == "" {
			catalogType = "helm"
		}
		if visibility == "" {
			visibility = "private"
		}

		pb.addSection("Catalog Configuration",
			fmt.Sprintf("Creating catalog: **%s**\n"+
				"Organization: **%s**\n"+
				"Type: **%s**\n"+
				"Visibility: **%s**",
				catalogName, organization, catalogType, visibility))

		// Step 2: Repository setup
		pb.addSection("Step 2: Set Up Your Repository",
			"You need a repository to host your app charts:")

		if catalogType == "helm" {
			pb.addSection("Helm Repository Setup",
				"For a Helm chart repository, you can use:")
			pb.addList("Repository Options", []string{
				"**GitHub Pages** - Free, easy to set up",
				"**GitLab Pages** - Integrated with GitLab CI",
				"**ChartMuseum** - Dedicated Helm chart server",
				"**Harbor** - Enterprise container registry with Helm support",
				"**AWS S3** - Cloud storage with static website hosting",
			})

			pb.addSection("Example: GitHub Pages Setup", "")
			pb.addCodeBlock("Repository Structure", "text",
				`my-charts/
├── charts/
│   ├── app1-0.1.0.tgz
│   ├── app2-1.0.0.tgz
│   └── ...
└── index.yaml`)

			pb.addSection("Generate Repository Index",
				"After adding charts, generate the index:")
			pb.addCodeBlock("Index Generation", "bash",
				`# In your charts directory
helm repo index . --url https://mycompany.github.io/charts/`)
		} else {
			pb.addSection("OCI Registry Setup",
				"For OCI-based catalogs, you can use:")
			pb.addList("Registry Options", []string{
				"**Harbor** - Full-featured registry with OCI support",
				"**Docker Hub** - Popular, widely accessible",
				"**GitHub Container Registry** - Integrated with GitHub",
				"**AWS ECR** - Managed container registry",
				"**Google Artifact Registry** - GCP's registry solution",
			})

			pb.addSection("Push Charts to OCI Registry", "")
			pb.addCodeBlock("OCI Commands", "bash",
				`# Package your chart
helm package ./my-app

# Push to OCI registry
helm push my-app-1.0.0.tgz oci://myregistry.io/helm-charts/`)
		}

		// Step 3: Create catalog command
		pb.addSection("Step 3: Create the Catalog Resource",
			"Once your repository is ready, create the catalog:")

		namespace := fmt.Sprintf("org-%s", organization)
		repoURL := "<YOUR_REPOSITORY_URL>"

		createCmd := fmt.Sprintf(`catalog.create \
  --name %s \
  --namespace %s \
  --title "%s Apps" \
  --description "Custom app catalog for %s" \
  --storage-url %s \
  --storage-type %s \
  --type stable \
  --visibility %s`,
			catalogName,
			namespace,
			organization,
			organization,
			repoURL,
			catalogType,
			visibility)

		pb.addCodeBlock("Create Catalog", "bash", createCmd)

		pb.addSection("Repository URL Examples", "")
		pb.addList("URL Formats", []string{
			"Helm HTTP: https://mycompany.github.io/charts/",
			"Helm HTTPS with auth: https://username:password@charts.mycompany.com",
			"OCI: oci://myregistry.io/helm-charts",
		})

		// Step 4: Add authentication (if private)
		if visibility == "private" {
			pb.addSection("Step 4: Configure Authentication",
				"For private catalogs, you need to provide credentials:")

			pb.addSection("Create Secret for Credentials", "")
			pb.addCodeBlock("Create Secret", "bash",
				fmt.Sprintf(`kubectl create secret generic %s-catalog-auth \
  --namespace %s \
  --from-literal=username=<USERNAME> \
  --from-literal=password=<PASSWORD>`,
					catalogName, namespace))

			pb.addSection("Reference Secret in Catalog",
				"Update the catalog to use the credentials:")
			pb.addCodeBlock("Update with Auth", "bash",
				fmt.Sprintf(`# After creating, update the catalog to reference the secret
# This would be done by editing the Catalog resource directly`))
		}

		// Step 5: Verification
		pb.addSection("Step 5: Verify the Catalog",
			"After creation, verify your catalog is working:")

		pb.addCodeBlock("Check Catalog", "bash",
			fmt.Sprintf("catalog.get --name %s --namespace %s", catalogName, namespace))

		pb.addCodeBlock("List Apps", "bash",
			fmt.Sprintf("appcatalogentry.list --catalog %s", catalogName))

		// Step 6: Adding apps
		pb.addSection("Step 6: Adding Apps to Your Catalog",
			"To add apps to your catalog:")

		pb.addList("Adding Apps", []string{
			"Package your Helm charts using 'helm package'",
			"Upload charts to your repository",
			"Update the repository index",
			"The catalog will automatically sync (may take a few minutes)",
		})

		pb.addSection("Example: Package and Upload", "")
		pb.addCodeBlock("Package Chart", "bash",
			`# Package your chart
helm package ./my-app-chart

# For Helm repos: copy to repository and update index
cp my-app-chart-1.0.0.tgz /path/to/repo/
helm repo index /path/to/repo/ --url https://myrepo.com

# For OCI: push directly
helm push my-app-chart-1.0.0.tgz oci://myregistry.io/charts/`)

		// Best practices
		pb.addSection("Best Practices", "")
		pb.addList("Catalog Management", []string{
			"Use semantic versioning for your apps",
			"Include comprehensive README files in charts",
			"Test charts before adding to production catalogs",
			"Use separate catalogs for dev/staging/prod",
			"Implement CI/CD for automatic chart updates",
			"Regular backup of your chart repository",
			"Document required values and dependencies",
		})

		// Troubleshooting
		pb.addSection("Troubleshooting", "")
		pb.addList("Common Issues", []string{
			"**No apps showing**: Check repository URL and accessibility",
			"**Authentication errors**: Verify credentials and secret configuration",
			"**Sync delays**: Allow 5-10 minutes for catalog to sync",
			"**Invalid charts**: Validate charts with 'helm lint'",
		})

		return &mcp.GetPromptResult{
			Description: fmt.Sprintf("Complete guide to create %s catalog for %s", catalogName, organization),
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
