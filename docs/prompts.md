# Prompts Guide

This document describes the interactive prompts available in the MCP Giant Swarm Apps server. These prompts provide step-by-step guidance for common Giant Swarm App Platform operations.

## Overview

Prompts are interactive guides that help users through complex operations. They provide:
- Step-by-step instructions
- Context-aware recommendations
- Best practices and tips
- Troubleshooting guidance
- Example configurations

## Available Prompts

### 1. deploy-app

Guides you through deploying a Giant Swarm app.

**Arguments:**
- `organization` - Organization to deploy the app in (e.g., 'giantswarm')
- `catalog` - Catalog name to browse apps from (e.g., 'giantswarm')
- `app` - App name to deploy (e.g., 'nginx-ingress-controller')
- `namespace` - Namespace to deploy the app in (defaults to organization namespace)

**Example Usage:**
```
prompt: deploy-app
prompt: deploy-app --organization giantswarm
prompt: deploy-app --organization giantswarm --catalog giantswarm --app nginx-ingress-controller
```

**What it covers:**
- Organization selection
- Catalog browsing
- App selection
- Version selection
- Configuration basics
- Deployment commands
- Verification steps
- Best practices

### 2. upgrade-app

Helps you safely upgrade an existing app to a new version.

**Arguments:**
- `name` - Name of the app to upgrade
- `namespace` - Namespace where the app is deployed
- `version` - Target version to upgrade to

**Example Usage:**
```
prompt: upgrade-app
prompt: upgrade-app --name nginx-ingress --namespace org-giantswarm
prompt: upgrade-app --name nginx-ingress --namespace org-giantswarm --version 2.16.0
```

**What it covers:**
- Current status check
- Version selection guidelines
- Pre-upgrade checklist
- Configuration compatibility
- Upgrade execution
- Progress monitoring
- Verification steps
- Rollback procedures

### 3. troubleshoot-app

Comprehensive troubleshooting guide for app issues.

**Arguments:**
- `name` - Name of the app having issues
- `namespace` - Namespace where the app is deployed
- `issue` - Type of issue: deployment, configuration, performance, or general

**Example Usage:**
```
prompt: troubleshoot-app
prompt: troubleshoot-app --name prometheus --namespace org-giantswarm
prompt: troubleshoot-app --name prometheus --namespace org-giantswarm --issue configuration
```

**What it covers:**
- Status diagnostics
- Issue-specific troubleshooting
- Configuration validation
- Resource checking
- Log analysis
- Recovery actions
- Support information gathering

### 4. create-catalog

Guide to create a custom app catalog.

**Arguments:**
- `name` - Name for the new catalog
- `organization` - Organization to create the catalog in
- `type` - Catalog type: helm or oci
- `visibility` - Catalog visibility: public or private

**Example Usage:**
```
prompt: create-catalog
prompt: create-catalog --name myteam-apps --organization giantswarm
prompt: create-catalog --name myteam-apps --organization giantswarm --type oci --visibility private
```

**What it covers:**
- Catalog planning
- Repository setup (Helm or OCI)
- Authentication configuration
- Catalog creation
- App addition process
- Best practices
- Troubleshooting

### 5. configure-app

Interactive configuration wizard for Giant Swarm apps.

**Arguments:**
- `app` - App name to configure (e.g., 'nginx-ingress-controller')
- `catalog` - Catalog containing the app
- `version` - App version to check configuration for
- `organization` - Organization context for the configuration

**Example Usage:**
```
prompt: configure-app
prompt: configure-app --app prometheus-operator --catalog giantswarm
prompt: configure-app --app nginx-ingress-controller --catalog giantswarm --version 2.16.0 --organization myorg
```

**What it covers:**
- Configuration schema exploration
- App-specific configuration examples
- ConfigMap/Secret creation
- Configuration precedence
- Advanced topics
- Best practices
- Validation techniques

## Using Prompts Effectively

### Progressive Disclosure

Prompts are designed with progressive disclosure - you can start with minimal arguments and the prompt will guide you through what's needed:

```
# Start with no arguments
prompt: deploy-app
# The prompt will ask for organization

# Provide organization
prompt: deploy-app --organization giantswarm
# The prompt will ask for catalog

# Continue adding arguments as guided
prompt: deploy-app --organization giantswarm --catalog giantswarm --app nginx-ingress-controller
```

### Context-Aware Guidance

Prompts provide different guidance based on:
- The specific app being configured
- Your organization context
- The current state of resources
- Common patterns and use cases

### Integration with Tools

Prompts show exact commands to run using the available MCP tools:
- They reference the correct tool syntax
- Include your specific parameters
- Provide copy-paste ready commands

## Best Practices

1. **Start Simple**: Begin with minimal arguments and let the prompt guide you
2. **Read Carefully**: Prompts contain important warnings and recommendations
3. **Follow Steps**: Complete each step before moving to the next
4. **Use Examples**: Pay attention to example configurations provided
5. **Test First**: Always test in non-production environments first

## Troubleshooting Prompts

If a prompt isn't providing expected guidance:

1. **Check Arguments**: Ensure argument names and values are correct
2. **Verify Resources**: Make sure referenced resources (apps, catalogs) exist
3. **Review Output**: The prompt description indicates what information is needed

## Examples

### Complete App Deployment

```
# Start the deployment wizard
prompt: deploy-app --organization mycompany

# Follow the guide to:
# 1. Select a catalog
# 2. Choose an app
# 3. Pick a version
# 4. Configure the app
# 5. Deploy it
# 6. Verify deployment
```

### Troubleshooting Failed Deployment

```
# Get comprehensive troubleshooting guide
prompt: troubleshoot-app --name failing-app --namespace org-mycompany --issue deployment

# The guide will help you:
# 1. Check app status
# 2. Identify the issue
# 3. Review logs and events
# 4. Take corrective action
```

### Creating Custom Catalog

```
# Start catalog creation wizard
prompt: create-catalog --name team-x-apps --organization mycompany --type helm

# The guide covers:
# 1. Repository setup
# 2. Catalog resource creation
# 3. Authentication configuration
# 4. Adding apps
# 5. Verification
```

## Summary

Prompts make complex Giant Swarm operations accessible by providing:
- Clear step-by-step guidance
- Context-aware recommendations
- Integration with existing tools
- Best practices and troubleshooting

Use prompts whenever you need guidance for app deployment, upgrades, configuration, or troubleshooting. 