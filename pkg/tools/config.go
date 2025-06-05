package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/giantswarm/mcp-giantswarm-apps/internal/server"
	"github.com/giantswarm/mcp-giantswarm-apps/pkg/config"
)

// RegisterConfigTools registers all configuration management tools
func RegisterConfigTools(s *mcpserver.MCPServer, ctx *server.Context) error {
	client := config.NewClient(ctx.K8sClient)

	// config.get tool
	getTool := mcp.NewTool(
		"config.get",
		mcp.WithDescription("Get app configuration (ConfigMap or Secret)"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the ConfigMap or Secret")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace")),
		mcp.WithString("type", mcp.Description("Type: configmap or secret (default: configmap)")),
		mcp.WithString("format", mcp.Description("Output format: yaml, json, or text (default: text)")),
		mcp.WithBoolean("decode", mcp.Description("Decode base64 values for secrets (default: false)")),
	)

	s.AddTool(getTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		name := args["name"].(string)
		namespace := args["namespace"].(string)
		configType := getStringArg(args, "type")
		format := getStringArg(args, "format")
		decode := getBoolArg(args, "decode")

		if configType == "" {
			configType = "configmap"
		}
		if format == "" {
			format = "text"
		}

		// Determine config type
		var cfgType config.ConfigType
		switch configType {
		case "configmap":
			cfgType = config.ConfigTypeConfigMap
		case "secret":
			cfgType = config.ConfigTypeSecret
		default:
			return nil, fmt.Errorf("invalid type: %s (must be configmap or secret)", configType)
		}

		// Get the configuration
		cfg, err := client.Get(toolCtx, namespace, name, cfgType)
		if err != nil {
			return nil, err
		}

		// Format output
		var output string
		switch format {
		case "yaml":
			output, err = cfg.ToYAML()
			if err != nil {
				return nil, err
			}
		case "json":
			output, err = cfg.ToJSON()
			if err != nil {
				return nil, err
			}
		default: // text
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Name: %s\n", cfg.Name))
			sb.WriteString(fmt.Sprintf("Namespace: %s\n", cfg.Namespace))
			sb.WriteString(fmt.Sprintf("Type: %s\n", cfg.Type))

			if len(cfg.Labels) > 0 {
				sb.WriteString("\nLabels:\n")
				for k, v := range cfg.Labels {
					sb.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
				}
			}

			sb.WriteString("\nData:\n")
			if cfg.IsSecret() && !decode {
				sb.WriteString("  (base64 encoded - use --decode to view)\n")
				cfg.EncodeSecretData()
			}
			for k, v := range cfg.Data {
				if len(v) > 100 && !strings.Contains(v, "\n") {
					v = v[:97] + "..."
				}
				sb.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
			}
			output = sb.String()
		}

		return mcp.NewToolResultText(output), nil
	})

	// config.set tool
	setTool := mcp.NewTool(
		"config.set",
		mcp.WithDescription("Update app configuration values"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the ConfigMap or Secret")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace")),
		mcp.WithString("type", mcp.Description("Type: configmap or secret (default: configmap)")),
		mcp.WithString("key", mcp.Required(), mcp.Description("Configuration key to set")),
		mcp.WithString("value", mcp.Required(), mcp.Description("Configuration value")),
		mcp.WithBoolean("create", mcp.Description("Create if it doesn't exist (default: false)")),
	)

	s.AddTool(setTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		name := args["name"].(string)
		namespace := args["namespace"].(string)
		configType := getStringArg(args, "type")
		key := args["key"].(string)
		value := args["value"].(string)
		create := getBoolArg(args, "create")

		if configType == "" {
			configType = "configmap"
		}

		// Determine config type
		var cfgType config.ConfigType
		switch configType {
		case "configmap":
			cfgType = config.ConfigTypeConfigMap
		case "secret":
			cfgType = config.ConfigTypeSecret
		default:
			return nil, fmt.Errorf("invalid type: %s (must be configmap or secret)", configType)
		}

		// Get or create the configuration
		cfg, err := client.Get(toolCtx, namespace, name, cfgType)
		if err != nil {
			if !create {
				return nil, err
			}
			// Create new config
			cfg = &config.Config{
				Name:      name,
				Namespace: namespace,
				Type:      cfgType,
				Data:      make(map[string]string),
				Labels:    make(map[string]string),
			}
		}

		// Set the value
		cfg.SetValue(key, value)

		// Update or create
		if err == nil {
			err = client.Update(toolCtx, cfg)
		} else {
			err = client.Create(toolCtx, cfg)
		}

		if err != nil {
			return nil, err
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully set %s=%s in %s %s/%s", key, value, configType, namespace, name)), nil
	})

	// config.validate tool
	validateTool := mcp.NewTool(
		"config.validate",
		mcp.WithDescription("Validate configuration against a schema"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the ConfigMap or Secret")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace")),
		mcp.WithString("type", mcp.Description("Type: configmap or secret (default: configmap)")),
		mcp.WithString("required-keys", mcp.Description("Comma-separated list of required keys")),
		mcp.WithString("optional-keys", mcp.Description("Comma-separated list of optional keys")),
	)

	s.AddTool(validateTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		name := args["name"].(string)
		namespace := args["namespace"].(string)
		configType := getStringArg(args, "type")
		requiredKeys := getStringArg(args, "required-keys")
		optionalKeys := getStringArg(args, "optional-keys")

		if configType == "" {
			configType = "configmap"
		}

		// Determine config type
		var cfgType config.ConfigType
		switch configType {
		case "configmap":
			cfgType = config.ConfigTypeConfigMap
		case "secret":
			cfgType = config.ConfigTypeSecret
		default:
			return nil, fmt.Errorf("invalid type: %s (must be configmap or secret)", configType)
		}

		// Get the configuration
		cfg, err := client.Get(toolCtx, namespace, name, cfgType)
		if err != nil {
			return nil, err
		}

		// Build schema
		schema := &config.ConfigSchema{
			RequiredKeys: []string{},
			OptionalKeys: []string{},
			KeyPatterns:  make(map[string]string),
		}

		if requiredKeys != "" {
			schema.RequiredKeys = strings.Split(requiredKeys, ",")
		}
		if optionalKeys != "" {
			schema.OptionalKeys = strings.Split(optionalKeys, ",")
		}

		// Validate
		result := client.Validate(cfg, schema)

		var output strings.Builder
		if result.Valid {
			output.WriteString("✓ Configuration is valid\n")
		} else {
			output.WriteString("✗ Configuration is invalid\n\nErrors:\n")
			for _, err := range result.Errors {
				output.WriteString(fmt.Sprintf("  - %s\n", err))
			}
		}

		return mcp.NewToolResultText(output.String()), nil
	})

	// config.diff tool
	diffTool := mcp.NewTool(
		"config.diff",
		mcp.WithDescription("Show differences between two configurations"),
		mcp.WithString("name1", mcp.Required(), mcp.Description("Name of the first ConfigMap/Secret")),
		mcp.WithString("namespace1", mcp.Required(), mcp.Description("Namespace of the first config")),
		mcp.WithString("name2", mcp.Required(), mcp.Description("Name of the second ConfigMap/Secret")),
		mcp.WithString("namespace2", mcp.Required(), mcp.Description("Namespace of the second config")),
		mcp.WithString("type", mcp.Description("Type: configmap or secret (default: configmap)")),
	)

	s.AddTool(diffTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		name1 := args["name1"].(string)
		namespace1 := args["namespace1"].(string)
		name2 := args["name2"].(string)
		namespace2 := args["namespace2"].(string)
		configType := getStringArg(args, "type")

		if configType == "" {
			configType = "configmap"
		}

		// Determine config type
		var cfgType config.ConfigType
		switch configType {
		case "configmap":
			cfgType = config.ConfigTypeConfigMap
		case "secret":
			cfgType = config.ConfigTypeSecret
		default:
			return nil, fmt.Errorf("invalid type: %s (must be configmap or secret)", configType)
		}

		// Get both configurations
		cfg1, err := client.Get(toolCtx, namespace1, name1, cfgType)
		if err != nil {
			return nil, fmt.Errorf("failed to get first config: %w", err)
		}

		cfg2, err := client.Get(toolCtx, namespace2, name2, cfgType)
		if err != nil {
			return nil, fmt.Errorf("failed to get second config: %w", err)
		}

		// Calculate diff
		diff := cfg1.Diff(cfg2)

		var output strings.Builder
		output.WriteString(fmt.Sprintf("Diff between %s/%s and %s/%s:\n\n", namespace1, name1, namespace2, name2))

		if !diff.HasChanges() {
			output.WriteString("No differences found\n")
		} else {
			if len(diff.Added) > 0 {
				output.WriteString("Added keys:\n")
				for k, v := range diff.Added {
					output.WriteString(fmt.Sprintf("  + %s: %s\n", k, v))
				}
			}

			if len(diff.Modified) > 0 {
				output.WriteString("\nModified keys:\n")
				for k, entry := range diff.Modified {
					output.WriteString(fmt.Sprintf("  ~ %s:\n", k))
					output.WriteString(fmt.Sprintf("    - %s\n", entry.Old))
					output.WriteString(fmt.Sprintf("    + %s\n", entry.New))
				}
			}

			if len(diff.Removed) > 0 {
				output.WriteString("\nRemoved keys:\n")
				for k, v := range diff.Removed {
					output.WriteString(fmt.Sprintf("  - %s: %s\n", k, v))
				}
			}
		}

		return mcp.NewToolResultText(output.String()), nil
	})

	// secret.create tool
	createSecretTool := mcp.NewTool(
		"secret.create",
		mcp.WithDescription("Create a new secret for an app"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the secret")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace")),
		mcp.WithString("data", mcp.Required(), mcp.Description("Secret data in key=value format (comma-separated)")),
		mcp.WithString("app", mcp.Description("App name to associate with the secret")),
		mcp.WithString("labels", mcp.Description("Additional labels in key=value format (comma-separated)")),
	)

	s.AddTool(createSecretTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		name := args["name"].(string)
		namespace := args["namespace"].(string)
		dataStr := args["data"].(string)
		appName := getStringArg(args, "app")
		labelsStr := getStringArg(args, "labels")

		// Parse data
		data := make(map[string]string)
		for _, kv := range strings.Split(dataStr, ",") {
			parts := strings.SplitN(kv, "=", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid data format: %s (expected key=value)", kv)
			}
			data[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}

		// Parse labels
		labels := make(map[string]string)
		if appName != "" {
			labels["app.kubernetes.io/name"] = appName
		}
		if labelsStr != "" {
			for _, kv := range strings.Split(labelsStr, ",") {
				parts := strings.SplitN(kv, "=", 2)
				if len(parts) != 2 {
					return nil, fmt.Errorf("invalid label format: %s (expected key=value)", kv)
				}
				labels[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}

		// Create secret
		secret := &config.Config{
			Name:      name,
			Namespace: namespace,
			Type:      config.ConfigTypeSecret,
			Data:      data,
			Labels:    labels,
		}

		err := client.Create(toolCtx, secret)
		if err != nil {
			return nil, err
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully created secret %s/%s with %d keys", namespace, name, len(data))), nil
	})

	// secret.update tool
	updateSecretTool := mcp.NewTool(
		"secret.update",
		mcp.WithDescription("Update an existing secret"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the secret")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace")),
		mcp.WithString("key", mcp.Description("Key to update (if not specified, replaces all data)")),
		mcp.WithString("value", mcp.Description("Value for the key")),
		mcp.WithString("data", mcp.Description("Complete data in key=value format (comma-separated)")),
		mcp.WithBoolean("merge", mcp.Description("Merge with existing data instead of replacing (default: false)")),
	)

	s.AddTool(updateSecretTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		name := args["name"].(string)
		namespace := args["namespace"].(string)
		key := getStringArg(args, "key")
		value := getStringArg(args, "value")
		dataStr := getStringArg(args, "data")
		merge := getBoolArg(args, "merge")

		// Get current secret
		secret, err := client.Get(toolCtx, namespace, name, config.ConfigTypeSecret)
		if err != nil {
			return nil, err
		}

		// Update data
		if key != "" && value != "" {
			// Update single key
			secret.SetValue(key, value)
		} else if dataStr != "" {
			// Parse new data
			newData := make(map[string]string)
			for _, kv := range strings.Split(dataStr, ",") {
				parts := strings.SplitN(kv, "=", 2)
				if len(parts) != 2 {
					return nil, fmt.Errorf("invalid data format: %s (expected key=value)", kv)
				}
				newData[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}

			if merge {
				// Merge with existing data
				for k, v := range newData {
					secret.SetValue(k, v)
				}
			} else {
				// Replace all data
				secret.Data = newData
			}
		} else {
			return nil, fmt.Errorf("either key/value or data must be specified")
		}

		// Update secret
		err = client.Update(toolCtx, secret)
		if err != nil {
			return nil, err
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully updated secret %s/%s", namespace, name)), nil
	})

	// config.merge tool for merging configurations
	mergeTool := mcp.NewTool(
		"config.merge",
		mcp.WithDescription("Merge multiple configurations (later ones take precedence)"),
		mcp.WithString("configs", mcp.Required(), mcp.Description("Comma-separated list of namespace/name pairs")),
		mcp.WithString("type", mcp.Description("Type: configmap or secret (default: configmap)")),
		mcp.WithString("format", mcp.Description("Output format: yaml, json, or text (default: text)")),
	)

	s.AddTool(mergeTool, func(toolCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		configsStr := args["configs"].(string)
		configType := getStringArg(args, "type")
		format := getStringArg(args, "format")

		if configType == "" {
			configType = "configmap"
		}
		if format == "" {
			format = "text"
		}

		// Determine config type
		var cfgType config.ConfigType
		switch configType {
		case "configmap":
			cfgType = config.ConfigTypeConfigMap
		case "secret":
			cfgType = config.ConfigTypeSecret
		default:
			return nil, fmt.Errorf("invalid type: %s (must be configmap or secret)", configType)
		}

		// Parse config references
		configRefs := strings.Split(configsStr, ",")
		configs := make([]*config.Config, 0, len(configRefs))

		for _, ref := range configRefs {
			parts := strings.Split(strings.TrimSpace(ref), "/")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid config reference: %s (expected namespace/name)", ref)
			}

			cfg, err := client.Get(toolCtx, parts[0], parts[1], cfgType)
			if err != nil {
				return nil, fmt.Errorf("failed to get %s: %w", ref, err)
			}
			configs = append(configs, cfg)
		}

		// Merge configurations
		merged := config.MergeConfigs(configs...)

		// Format output
		var output string
		var err error
		switch format {
		case "yaml":
			output, err = merged.ToYAML()
			if err != nil {
				return nil, err
			}
		case "json":
			output, err = merged.ToJSON()
			if err != nil {
				return nil, err
			}
		default: // text
			var sb strings.Builder
			sb.WriteString("Merged configuration:\n\n")
			for k, v := range merged.Data {
				sb.WriteString(fmt.Sprintf("%s: %s\n", k, v))
			}
			output = sb.String()
		}

		return mcp.NewToolResultText(output), nil
	})

	return nil
}
