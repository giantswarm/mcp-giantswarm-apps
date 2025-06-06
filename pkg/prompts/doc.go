// Package prompts provides interactive guided workflows for common Giant Swarm App Platform operations.
//
// # Overview
//
// Prompts are interactive guides that help users through complex operations by providing:
//   - Step-by-step instructions
//   - Context-aware recommendations
//   - Best practices and tips
//   - Troubleshooting guidance
//   - Example configurations
//
// The prompts use progressive disclosure, allowing users to start with minimal information
// and guiding them through what's needed at each step.
//
// # Available Prompts
//
// The package includes five main prompts:
//
//   - deploy-app: Guides through deploying a Giant Swarm app
//   - upgrade-app: Helps safely upgrade an app to a new version
//   - troubleshoot-app: Comprehensive troubleshooting guide
//   - create-catalog: Guide to create custom app catalogs
//   - configure-app: Interactive configuration wizard
//
// # Usage
//
// Prompts are registered with the MCP server and can be invoked by clients:
//
//	// Start with minimal arguments
//	prompt: deploy-app
//
//	// Or provide more context
//	prompt: deploy-app --organization giantswarm --catalog giantswarm --app nginx-ingress-controller
//
// # Architecture
//
// The prompts package provides:
//   - A prompt builder for constructing formatted responses
//   - Input validation utilities
//   - Individual prompt implementations
//   - Registration function for MCP server integration
//
// Each prompt follows a consistent pattern:
//  1. Parse and validate arguments
//  2. Check what information is still needed
//  3. Build appropriate guidance
//  4. Return formatted prompt result
//
// # Best Practices
//
// When implementing new prompts:
//   - Use progressive disclosure for complex workflows
//   - Provide concrete examples and commands
//   - Include validation and error prevention
//   - Reference appropriate tools and commands
//   - Follow Giant Swarm platform conventions
package prompts
