package prompts

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/giantswarm/mcp-giantswarm-apps/internal/server"
)

func registerConfigureAppPrompt(s *mcpserver.MCPServer, ctx *server.Context) error {
	prompt := mcp.NewPrompt(
		"configure-app",
		mcp.WithPromptDescription("App configuration wizard for Giant Swarm apps"),
		mcp.WithArgument("app", mcp.ArgumentDescription("App name to configure (e.g., 'nginx-ingress-controller')")),
		mcp.WithArgument("catalog", mcp.ArgumentDescription("Catalog containing the app")),
		mcp.WithArgument("version", mcp.ArgumentDescription("App version to check configuration for")),
		mcp.WithArgument("organization", mcp.ArgumentDescription("Organization context for the configuration")),
	)

	s.AddPrompt(prompt, func(promptCtx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		args := req.Params.Arguments

		appName := args["app"]
		catalog := args["catalog"]
		version := args["version"]
		organization := args["organization"]

		pb := newPromptBuilder()

		// Title and overview
		pb.addSection("App Configuration Wizard",
			"This guide will help you properly configure a Giant Swarm app. "+
				"We'll explore available configuration options and create a valid configuration.")

		// Step 1: App selection
		if appName == "" || catalog == "" {
			pb.addSection("Step 1: Select App to Configure",
				"First, identify which app you want to configure:")

			if catalog == "" {
				pb.addCodeBlock("List Available Catalogs", "bash", "catalog.list")
				pb.addSection("Popular Catalogs",
					"- **giantswarm**: Official Giant Swarm apps\n"+
						"- **giantswarm-playground**: Experimental apps\n"+
						"- **giantswarm-incubator**: Apps in development")
			}

			if appName == "" && catalog != "" {
				pb.addCodeBlock("Browse Apps in Catalog", "bash",
					fmt.Sprintf("appcatalogentry.list --catalog %s", catalog))
			}

			pb.addSection("Action Required",
				"Please specify both 'app' and 'catalog' arguments.")
			return &mcp.GetPromptResult{
				Description: "Configuration wizard - app selection needed",
				Messages: []mcp.PromptMessage{
					{
						Role:    mcp.RoleUser,
						Content: mcp.TextContent{Text: pb.build()},
					},
				},
			}, nil
		}

		pb.addSection("App Information",
			fmt.Sprintf("Configuring: **%s** from catalog **%s**", appName, catalog))

		// Step 2: Version selection
		if version == "" {
			pb.addSection("Step 2: Select Version",
				"Check available versions and their configuration requirements:")
			pb.addCodeBlock("Get App Versions", "bash",
				fmt.Sprintf("appcatalogentry.get --catalog %s --name %s", catalog, appName))
			pb.addSection("Version Selection Tips",
				"- Use the latest stable version for new deployments\n"+
					"- Check version changelog for configuration changes\n"+
					"- Match versions with your cluster compatibility")
			pb.addSection("Action Required",
				"Please specify the 'version' argument.")
			return &mcp.GetPromptResult{
				Description: "Configuration wizard - version selection needed",
				Messages: []mcp.PromptMessage{
					{
						Role:    mcp.RoleUser,
						Content: mcp.TextContent{Text: pb.build()},
					},
				},
			}, nil
		}

		// Step 3: Get configuration schema
		pb.addSection("Step 3: Understanding Configuration Options",
			"Let's explore what can be configured for this app:")
		pb.addCodeBlock("View Configuration Schema", "bash",
			fmt.Sprintf("config.schema --catalog %s --app %s --version %s", catalog, appName, version))

		// Common configuration patterns by app type
		pb.addSection("Common Configuration Patterns", "")

		if appName == "nginx-ingress-controller" || appName == "ingress-nginx" {
			pb.addSection("Ingress Controller Configuration",
				"Key configuration areas for ingress controllers:")
			pb.addList("Common Settings", []string{
				"**Service Type**: LoadBalancer, NodePort, or ClusterIP",
				"**Replica Count**: Number of controller instances",
				"**Resources**: CPU and memory requests/limits",
				"**Default SSL Certificate**: For HTTPS termination",
				"**Custom Annotations**: Cloud provider specific settings",
			})
			pb.addCodeBlock("Example Configuration", "yaml",
				`controller:
  replicaCount: 2
  service:
    type: LoadBalancer
    annotations:
      service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
  resources:
    requests:
      cpu: 100m
      memory: 256Mi
    limits:
      cpu: 500m
      memory: 512Mi`)
		} else if appName == "prometheus-operator" || appName == "kube-prometheus-stack" {
			pb.addSection("Monitoring Stack Configuration",
				"Key configuration areas for monitoring:")
			pb.addList("Common Settings", []string{
				"**Storage**: Persistent volume configuration",
				"**Retention**: How long to keep metrics",
				"**Resources**: Sizing for Prometheus instances",
				"**Ingress**: External access configuration",
				"**Alerting**: Alert manager configuration",
			})
			pb.addCodeBlock("Example Configuration", "yaml",
				`prometheus:
  prometheusSpec:
    retention: 30d
    storageSpec:
      volumeClaimTemplate:
        spec:
          accessModes: ["ReadWriteOnce"]
          resources:
            requests:
              storage: 50Gi
alertmanager:
  enabled: true
  config:
    route:
      group_by: ['alertname', 'cluster']`)
		} else if appName == "cert-manager" {
			pb.addSection("Certificate Manager Configuration",
				"Key configuration areas for cert-manager:")
			pb.addList("Common Settings", []string{
				"**Issuers**: Let's Encrypt or internal CA",
				"**DNS Providers**: For DNS-01 challenges",
				"**Resources**: Controller sizing",
				"**CRDs**: Custom Resource Definitions management",
			})
			pb.addCodeBlock("Example Configuration", "yaml",
				`installCRDs: true
resources:
  requests:
    cpu: 10m
    memory: 64Mi
webhook:
  enabled: true`)
		}

		// Step 4: Create configuration
		pb.addSection("Step 4: Create Your Configuration",
			"Based on the schema and your requirements, create a configuration file:")

		namespace := "default"
		if organization != "" {
			namespace = fmt.Sprintf("org-%s", organization)
		}

		configMapName := fmt.Sprintf("%s-config", appName)

		pb.addSection("Option A: Create ConfigMap with Values",
			"Create a ConfigMap containing your configuration:")
		pb.addCodeBlock("Create ConfigMap", "bash",
			fmt.Sprintf(`# Create a values.yaml file with your configuration
cat > %s-values.yaml << EOF
# Your configuration here
replicaCount: 2
service:
  type: LoadBalancer
EOF

# Create ConfigMap from file
kubectl create configmap %s \
  --namespace %s \
  --from-file=values.yaml=%s-values.yaml`,
				appName, configMapName, namespace, appName))

		pb.addSection("Option B: Create ConfigMap Inline",
			"For simple configurations, create directly:")
		pb.addCodeBlock("Inline ConfigMap", "bash",
			fmt.Sprintf(`kubectl create configmap %s \
  --namespace %s \
  --from-literal=values.yaml='
replicaCount: 2
service:
  type: LoadBalancer
resources:
  requests:
    cpu: 100m
    memory: 256Mi
'`, configMapName, namespace))

		// Step 5: Reference in app
		pb.addSection("Step 5: Reference Configuration in App",
			"When creating or updating the app, reference your configuration:")
		pb.addCodeBlock("Deploy with Configuration", "bash",
			fmt.Sprintf(`app.create \
  --name %s \
  --namespace %s \
  --catalog %s \
  --app %s \
  --version %s \
  --config-name %s`,
				appName, namespace, catalog, appName, version, configMapName))

		// Step 6: Advanced configuration
		pb.addSection("Step 6: Advanced Configuration Topics", "")

		pb.addSection("User Configuration vs App Configuration",
			"Giant Swarm apps support two configuration levels:")
		pb.addList("Configuration Types", []string{
			"**App Configuration**: Platform-level defaults (managed by admins)",
			"**User Configuration**: User-specific overrides (higher precedence)",
		})

		pb.addSection("Using Secrets for Sensitive Data",
			"For sensitive configuration like passwords:")
		pb.addCodeBlock("Create Secret", "bash",
			fmt.Sprintf(`kubectl create secret generic %s-secret \
  --namespace %s \
  --from-literal=password=mysecretpassword \
  --from-literal=apiKey=myapikey`, appName, namespace))

		pb.addSection("Configuration Precedence",
			"Values are merged in this order (later overrides earlier):")
		pb.addList("Precedence Order", []string{
			"1. Default values from the Helm chart",
			"2. App configuration (ConfigMap/Secret)",
			"3. User configuration (ConfigMap/Secret)",
			"4. Extra configuration (if supported)",
		})

		// Best practices
		pb.addSection("Configuration Best Practices", "")
		pb.addList("Recommendations", []string{
			"Start with minimal configuration and add as needed",
			"Use comments to document your configuration choices",
			"Keep sensitive data in Secrets, not ConfigMaps",
			"Version your configuration files in Git",
			"Test configuration changes in non-production first",
			"Use meaningful names for ConfigMaps and Secrets",
			"Regular review and cleanup of unused configurations",
		})

		// Validation
		pb.addSection("Validate Your Configuration", "")
		pb.addSection("Dry Run",
			"Some apps support dry-run to validate configuration:")
		pb.addCodeBlock("Validation Commands", "bash",
			`# Check YAML syntax
kubectl create configmap test --from-file=values.yaml --dry-run=client -o yaml

# Validate against schema (if available)
helm lint . -f values.yaml`)

		// Troubleshooting
		pb.addSection("Configuration Troubleshooting", "")
		pb.addList("Common Issues", []string{
			"**YAML syntax errors**: Use a YAML validator",
			"**Type mismatches**: Check schema for correct types",
			"**Missing required values**: Review schema requirements",
			"**ConfigMap not found**: Ensure it exists in the right namespace",
			"**Values not applied**: Check configuration precedence",
		})

		return &mcp.GetPromptResult{
			Description: fmt.Sprintf("Configuration guide for %s v%s", appName, version),
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
