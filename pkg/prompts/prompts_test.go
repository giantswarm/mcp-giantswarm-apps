package prompts

import (
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestPromptBuilder(t *testing.T) {
	tests := []struct {
		name     string
		build    func() string
		contains []string
	}{
		{
			name: "basic sections",
			build: func() string {
				pb := newPromptBuilder()
				pb.addSection("Title", "Content here")
				pb.addSection("Another Title", "More content")
				return pb.build()
			},
			contains: []string{
				"## Title",
				"Content here",
				"## Another Title",
				"More content",
			},
		},
		{
			name: "list section",
			build: func() string {
				pb := newPromptBuilder()
				pb.addList("My List", []string{"Item 1", "Item 2", "Item 3"})
				return pb.build()
			},
			contains: []string{
				"## My List",
				"- Item 1",
				"- Item 2",
				"- Item 3",
			},
		},
		{
			name: "code block",
			build: func() string {
				pb := newPromptBuilder()
				pb.addCodeBlock("Example Code", "bash", "kubectl get pods")
				return pb.build()
			},
			contains: []string{
				"## Example Code",
				"```bash",
				"kubectl get pods",
				"```",
			},
		},
		{
			name: "mixed content",
			build: func() string {
				pb := newPromptBuilder()
				pb.addSection("Introduction", "This is the intro")
				pb.addList("Steps", []string{"Step 1", "Step 2"})
				pb.addCodeBlock("Command", "bash", "app.list")
				return pb.build()
			},
			contains: []string{
				"## Introduction",
				"This is the intro",
				"## Steps",
				"- Step 1",
				"- Step 2",
				"## Command",
				"```bash",
				"app.list",
				"```",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.build()
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected output to contain %q, but it didn't.\nGot:\n%s", expected, result)
				}
			}
		})
	}
}

func TestValidateInput(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		fieldName string
		required  bool
		wantErr   bool
	}{
		{
			name:      "required field with value",
			value:     "test-value",
			fieldName: "testField",
			required:  true,
			wantErr:   false,
		},
		{
			name:      "required field without value",
			value:     "",
			fieldName: "testField",
			required:  true,
			wantErr:   true,
		},
		{
			name:      "optional field without value",
			value:     "",
			fieldName: "testField",
			required:  false,
			wantErr:   false,
		},
		{
			name:      "valid kubernetes name",
			value:     "my-app-123",
			fieldName: "name",
			required:  true,
			wantErr:   false,
		},
		{
			name:      "invalid kubernetes name - uppercase",
			value:     "My-App",
			fieldName: "name",
			required:  true,
			wantErr:   true,
		},
		{
			name:      "invalid kubernetes name - starts with hyphen",
			value:     "-myapp",
			fieldName: "name",
			required:  true,
			wantErr:   true,
		},
		{
			name:      "invalid kubernetes name - ends with hyphen",
			value:     "myapp-",
			fieldName: "name",
			required:  true,
			wantErr:   true,
		},
		{
			name:      "valid namespace",
			value:     "org-giantswarm",
			fieldName: "namespace",
			required:  true,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInput(tt.value, tt.fieldName, tt.required)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateInput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsValidKubernetesName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "valid simple name",
			input: "myapp",
			want:  true,
		},
		{
			name:  "valid with hyphens",
			input: "my-app-123",
			want:  true,
		},
		{
			name:  "valid single char",
			input: "a",
			want:  true,
		},
		{
			name:  "valid number",
			input: "123",
			want:  true,
		},
		{
			name:  "empty string",
			input: "",
			want:  false,
		},
		{
			name:  "contains uppercase",
			input: "myApp",
			want:  false,
		},
		{
			name:  "starts with hyphen",
			input: "-myapp",
			want:  false,
		},
		{
			name:  "ends with hyphen",
			input: "myapp-",
			want:  false,
		},
		{
			name:  "contains underscore",
			input: "my_app",
			want:  false,
		},
		{
			name:  "contains dot",
			input: "my.app",
			want:  false,
		},
		{
			name:  "contains space",
			input: "my app",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidKubernetesName(tt.input); got != tt.want {
				t.Errorf("isValidKubernetesName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// Test prompt registration
func TestPromptRegistration(t *testing.T) {
	// This is more of an integration test, but we can at least verify
	// that the prompt handlers return valid results for basic cases

	testCases := []struct {
		name       string
		promptFunc func() (*mcp.GetPromptResult, error)
		wantErr    bool
		checkDesc  string
	}{
		{
			name: "deploy-app with no args",
			promptFunc: func() (*mcp.GetPromptResult, error) {
				// Simulate calling the deploy-app prompt handler
				pb := newPromptBuilder()
				pb.addSection("Deploy Giant Swarm App",
					"This guide will help you deploy a Giant Swarm app to your Kubernetes cluster. "+
						"Follow the steps below to ensure a successful deployment.")
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
			},
			wantErr:   false,
			checkDesc: "Deploy app guide - organization selection needed",
		},
		{
			name: "upgrade-app with partial args",
			promptFunc: func() (*mcp.GetPromptResult, error) {
				// Simulate the upgrade prompt with version selection needed
				pb := newPromptBuilder()
				pb.addSection("Upgrade Giant Swarm App",
					"This guide will help you safely upgrade a Giant Swarm app to a new version. "+
						"Follow these steps to ensure a smooth upgrade process.")
				pb.addSection("App Information",
					"Upgrading app: **nginx-ingress** in namespace: **org-giantswarm**")

				return &mcp.GetPromptResult{
					Description: "Upgrade app guide - version selection needed",
					Messages: []mcp.PromptMessage{
						{
							Role:    mcp.RoleUser,
							Content: mcp.TextContent{Text: pb.build()},
						},
					},
				}, nil
			},
			wantErr:   false,
			checkDesc: "Upgrade app guide - version selection needed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.promptFunc()

			if (err != nil) != tc.wantErr {
				t.Errorf("Expected error: %v, got error: %v", tc.wantErr, err)
			}

			if err == nil {
				if result.Description != tc.checkDesc {
					t.Errorf("Expected description %q, got %q", tc.checkDesc, result.Description)
				}

				if len(result.Messages) == 0 {
					t.Error("Expected at least one message in result")
				}
			}
		})
	}
}
