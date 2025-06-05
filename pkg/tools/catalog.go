package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/giantswarm/mcp-giantswarm-apps/internal/server"
	"github.com/giantswarm/mcp-giantswarm-apps/pkg/catalog"
)

// RegisterCatalogTools registers all catalog management tools
func RegisterCatalogTools(s *mcpserver.MCPServer, ctx *server.Context) error {
	catalogClient := catalog.NewClient(ctx.DynamicClient)

	// catalog.list tool
	listTool := mcp.NewTool(
		"catalog.list",
		mcp.WithDescription("List Giant Swarm catalogs"),
		mcp.WithString("namespace", mcp.Description("Namespace to list catalogs from (empty for all namespaces)")),
		mcp.WithString("type", mcp.Description("Filter by catalog type (stable, testing, community)")),
		mcp.WithString("visibility", mcp.Description("Filter by visibility (public, private)")),
	)

	s.AddTool(listTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})

		namespace := getStringArg(args, "namespace")
		catalogType := getStringArg(args, "type")
		visibility := getStringArg(args, "visibility")

		catalogs, err := catalogClient.List(toolCtx, namespace)
		if err != nil {
			return nil, err
		}

		// Apply filters
		catalogs = catalog.FilterByType(catalogs, catalogType)
		catalogs = catalog.FilterByVisibility(catalogs, visibility)

		// Format output
		if len(catalogs) == 0 {
			return mcp.NewToolResultText("No catalogs found"), nil
		}

		var output strings.Builder
		output.WriteString(fmt.Sprintf("Found %d catalogs:\n\n", len(catalogs)))

		for _, c := range catalogs {
			output.WriteString(fmt.Sprintf("Name: %s\n", c.Name))
			output.WriteString(fmt.Sprintf("Namespace: %s\n", c.Namespace))
			output.WriteString(fmt.Sprintf("Title: %s\n", c.Spec.Title))
			output.WriteString(fmt.Sprintf("Description: %s\n", c.Spec.Description))
			output.WriteString(fmt.Sprintf("Type: %s\n", c.CatalogType()))
			output.WriteString(fmt.Sprintf("Visibility: %s\n", c.CatalogVisibility()))
			output.WriteString(fmt.Sprintf("Storage URL: %s\n", c.Spec.Storage.URL))
			if len(c.Spec.Repositories) > 0 {
				output.WriteString("Repositories:\n")
				for _, repo := range c.Spec.Repositories {
					output.WriteString(fmt.Sprintf("  - Type: %s, URL: %s\n", repo.Type, repo.URL))
				}
			}
			output.WriteString("---\n")
		}

		return mcp.NewToolResultText(output.String()), nil
	})

	// catalog.get tool
	getTool := mcp.NewTool(
		"catalog.get",
		mcp.WithDescription("Get detailed information about a specific catalog"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the catalog")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the catalog")),
	)

	s.AddTool(getTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		name := args["name"].(string)
		namespace := args["namespace"].(string)

		catalog, err := catalogClient.Get(toolCtx, namespace, name)
		if err != nil {
			return nil, err
		}

		var output strings.Builder
		output.WriteString(fmt.Sprintf("Catalog: %s\n", catalog.Name))
		output.WriteString(fmt.Sprintf("Namespace: %s\n", catalog.Namespace))
		output.WriteString("\nMetadata:\n")
		output.WriteString(fmt.Sprintf("  Type: %s\n", catalog.CatalogType()))
		output.WriteString(fmt.Sprintf("  Visibility: %s\n", catalog.CatalogVisibility()))

		output.WriteString("\nSpec:\n")
		output.WriteString(fmt.Sprintf("  Title: %s\n", catalog.Spec.Title))
		output.WriteString(fmt.Sprintf("  Description: %s\n", catalog.Spec.Description))
		if catalog.Spec.LogoURL != "" {
			output.WriteString(fmt.Sprintf("  Logo URL: %s\n", catalog.Spec.LogoURL))
		}

		output.WriteString("\nStorage:\n")
		output.WriteString(fmt.Sprintf("  Type: %s\n", catalog.Spec.Storage.Type))
		output.WriteString(fmt.Sprintf("  URL: %s\n", catalog.Spec.Storage.URL))

		if len(catalog.Spec.Repositories) > 0 {
			output.WriteString("\nRepositories:\n")
			for i, repo := range catalog.Spec.Repositories {
				output.WriteString(fmt.Sprintf("  %d. Type: %s\n", i+1, repo.Type))
				output.WriteString(fmt.Sprintf("     URL: %s\n", repo.URL))
			}
		}

		if catalog.Spec.Config != nil {
			output.WriteString("\nConfiguration:\n")
			if catalog.Spec.Config.ConfigMap != nil {
				output.WriteString(fmt.Sprintf("  ConfigMap: %s/%s\n",
					catalog.Spec.Config.ConfigMap.Namespace, catalog.Spec.Config.ConfigMap.Name))
			}
			if catalog.Spec.Config.Secret != nil {
				output.WriteString(fmt.Sprintf("  Secret: %s/%s\n",
					catalog.Spec.Config.Secret.Namespace, catalog.Spec.Config.Secret.Name))
			}
		}

		return mcp.NewToolResultText(output.String()), nil
	})

	// catalog.create tool
	createTool := mcp.NewTool(
		"catalog.create",
		mcp.WithDescription("Create a new Giant Swarm catalog"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name for the catalog")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace to create the catalog in")),
		mcp.WithString("title", mcp.Required(), mcp.Description("Human-readable title")),
		mcp.WithString("description", mcp.Required(), mcp.Description("Catalog description")),
		mcp.WithString("storage-url", mcp.Required(), mcp.Description("URL for the Helm repository")),
		mcp.WithString("storage-type", mcp.Description("Storage type (helm or oci, default: helm)")),
		mcp.WithString("logo-url", mcp.Description("URL for the catalog logo")),
		mcp.WithString("type", mcp.Description("Catalog type (stable, testing, community)")),
		mcp.WithString("visibility", mcp.Description("Catalog visibility (public, private)")),
		mcp.WithString("oci-url", mcp.Description("Additional OCI registry URL")),
	)

	s.AddTool(createTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})

		name := args["name"].(string)
		namespace := args["namespace"].(string)
		title := args["title"].(string)
		description := args["description"].(string)
		storageURL := args["storage-url"].(string)

		storageType := getStringArg(args, "storage-type")
		if storageType == "" {
			storageType = "helm"
		}

		// Validate storage URL
		if err := catalog.ValidateRepositoryURL(storageURL); err != nil {
			return nil, fmt.Errorf("invalid storage URL: %w", err)
		}

		newCatalog := &catalog.Catalog{
			Name:      name,
			Namespace: namespace,
			Spec: catalog.CatalogSpec{
				Title:       title,
				Description: description,
				LogoURL:     getStringArg(args, "logo-url"),
				Storage: catalog.Storage{
					Type: storageType,
					URL:  storageURL,
				},
				Repositories: []catalog.Repository{
					{
						Type: storageType,
						URL:  storageURL,
					},
				},
			},
			Labels: make(map[string]string),
		}

		// Add OCI repository if provided
		if ociURL := getStringArg(args, "oci-url"); ociURL != "" {
			newCatalog.Spec.Repositories = append(newCatalog.Spec.Repositories, catalog.Repository{
				Type: "oci",
				URL:  ociURL,
			})
		}

		// Set labels
		if catalogType := getStringArg(args, "type"); catalogType != "" {
			newCatalog.Labels["application.giantswarm.io/catalog-type"] = catalogType
		}
		if visibility := getStringArg(args, "visibility"); visibility != "" {
			newCatalog.Labels["application.giantswarm.io/catalog-visibility"] = visibility
		}

		created, err := catalogClient.Create(toolCtx, newCatalog)
		if err != nil {
			return nil, err
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully created catalog %s/%s", created.Namespace, created.Name)), nil
	})

	// catalog.update tool
	updateTool := mcp.NewTool(
		"catalog.update",
		mcp.WithDescription("Update an existing Giant Swarm catalog"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the catalog")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the catalog")),
		mcp.WithString("title", mcp.Description("Update title")),
		mcp.WithString("description", mcp.Description("Update description")),
		mcp.WithString("storage-url", mcp.Description("Update storage URL")),
		mcp.WithString("logo-url", mcp.Description("Update logo URL")),
		mcp.WithString("type", mcp.Description("Update catalog type")),
		mcp.WithString("visibility", mcp.Description("Update visibility")),
	)

	s.AddTool(updateTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		name := args["name"].(string)
		namespace := args["namespace"].(string)

		// Get current catalog
		currentCatalog, err := catalogClient.Get(toolCtx, namespace, name)
		if err != nil {
			return nil, err
		}

		// Update fields if provided
		if title := getStringArg(args, "title"); title != "" {
			currentCatalog.Spec.Title = title
		}
		if description := getStringArg(args, "description"); description != "" {
			currentCatalog.Spec.Description = description
		}
		if storageURL := getStringArg(args, "storage-url"); storageURL != "" {
			if err := catalog.ValidateRepositoryURL(storageURL); err != nil {
				return nil, fmt.Errorf("invalid storage URL: %w", err)
			}
			currentCatalog.Spec.Storage.URL = storageURL
			// Update first repository URL as well
			if len(currentCatalog.Spec.Repositories) > 0 {
				currentCatalog.Spec.Repositories[0].URL = storageURL
			}
		}
		if logoURL := getStringArg(args, "logo-url"); logoURL != "" {
			currentCatalog.Spec.LogoURL = logoURL
		}

		// Update labels
		if catalogType := getStringArg(args, "type"); catalogType != "" {
			if currentCatalog.Labels == nil {
				currentCatalog.Labels = make(map[string]string)
			}
			currentCatalog.Labels["application.giantswarm.io/catalog-type"] = catalogType
		}
		if visibility := getStringArg(args, "visibility"); visibility != "" {
			if currentCatalog.Labels == nil {
				currentCatalog.Labels = make(map[string]string)
			}
			currentCatalog.Labels["application.giantswarm.io/catalog-visibility"] = visibility
		}

		updated, err := catalogClient.Update(toolCtx, currentCatalog)
		if err != nil {
			return nil, err
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully updated catalog %s/%s", updated.Namespace, updated.Name)), nil
	})

	// catalog.delete tool
	deleteTool := mcp.NewTool(
		"catalog.delete",
		mcp.WithDescription("Delete a Giant Swarm catalog"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the catalog")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the catalog")),
	)

	s.AddTool(deleteTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		name := args["name"].(string)
		namespace := args["namespace"].(string)

		err := catalogClient.Delete(toolCtx, namespace, name)
		if err != nil {
			return nil, err
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully deleted catalog %s/%s", namespace, name)), nil
	})

	return nil
}
