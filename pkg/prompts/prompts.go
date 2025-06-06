package prompts

import (
	"fmt"
	"strings"

	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/giantswarm/mcp-giantswarm-apps/internal/server"
)

// RegisterPrompts registers all available prompts with the MCP server
func RegisterPrompts(s *mcpserver.MCPServer, ctx *server.Context) error {
	// Register deploy-app prompt
	if err := registerDeployAppPrompt(s, ctx); err != nil {
		return fmt.Errorf("failed to register deploy-app prompt: %w", err)
	}

	// Register upgrade-app prompt
	if err := registerUpgradeAppPrompt(s, ctx); err != nil {
		return fmt.Errorf("failed to register upgrade-app prompt: %w", err)
	}

	// Register troubleshoot-app prompt
	if err := registerTroubleshootAppPrompt(s, ctx); err != nil {
		return fmt.Errorf("failed to register troubleshoot-app prompt: %w", err)
	}

	// Register create-catalog prompt
	if err := registerCreateCatalogPrompt(s, ctx); err != nil {
		return fmt.Errorf("failed to register create-catalog prompt: %w", err)
	}

	// Register configure-app prompt
	if err := registerConfigureAppPrompt(s, ctx); err != nil {
		return fmt.Errorf("failed to register configure-app prompt: %w", err)
	}

	return nil
}

// promptBuilder helps build formatted prompts with sections
type promptBuilder struct {
	sections []string
}

func newPromptBuilder() *promptBuilder {
	return &promptBuilder{
		sections: make([]string, 0),
	}
}

func (pb *promptBuilder) addSection(title, content string) {
	section := fmt.Sprintf("## %s\n\n%s", title, content)
	pb.sections = append(pb.sections, section)
}

func (pb *promptBuilder) addList(title string, items []string) {
	var content strings.Builder
	for _, item := range items {
		content.WriteString(fmt.Sprintf("- %s\n", item))
	}
	pb.addSection(title, content.String())
}

func (pb *promptBuilder) addCodeBlock(title, language, code string) {
	content := fmt.Sprintf("```%s\n%s\n```", language, code)
	pb.addSection(title, content)
}

func (pb *promptBuilder) build() string {
	return strings.Join(pb.sections, "\n\n")
}

// validateInput provides common input validation
func validateInput(value, fieldName string, required bool) error {
	if required && value == "" {
		return fmt.Errorf("%s is required", fieldName)
	}

	// Check for valid Kubernetes resource names
	if fieldName == "name" || fieldName == "namespace" {
		if !isValidKubernetesName(value) {
			return fmt.Errorf("%s must be a valid Kubernetes resource name (lowercase alphanumeric and hyphens)", fieldName)
		}
	}

	return nil
}

// isValidKubernetesName checks if a string is a valid Kubernetes resource name
func isValidKubernetesName(name string) bool {
	if name == "" {
		return false
	}

	// Must start and end with alphanumeric
	if !isAlphanumeric(name[0]) || !isAlphanumeric(name[len(name)-1]) {
		return false
	}

	// Check all characters
	for _, char := range name {
		if !isAlphanumeric(byte(char)) && char != '-' {
			return false
		}
	}

	return true
}

func isAlphanumeric(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}
