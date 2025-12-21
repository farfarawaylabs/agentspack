package templates

import (
	"fmt"
	"strings"
)

// WorkflowStep represents a single step in a workflow
type WorkflowStep struct {
	Order       int    // Step number (1, 2, 3...)
	Name        string // Step name derived from filename
	RuleName    string // The rule name for referencing (e.g., "workflow-planning-01-create-prd")
	Description string // Brief description extracted from content
}

// WorkflowOrchestratorData holds data for generating a workflow orchestrator
type WorkflowOrchestratorData struct {
	WorkflowName string         // e.g., "planning"
	DisplayName  string         // e.g., "Planning"
	Description  string         // Brief description of the workflow
	Steps        []WorkflowStep // Ordered list of steps
}

// GenerateWorkflowOrchestrator creates the orchestrator content for a workflow
// This template can be adapted for different providers
func GenerateWorkflowOrchestrator(data WorkflowOrchestratorData, ruleRefPrefix string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s Workflow\n\n", data.DisplayName))
	sb.WriteString(fmt.Sprintf("%s\n\n", data.Description))

	sb.WriteString("## How to Use This Workflow\n\n")
	sb.WriteString("This workflow guides you through a structured process. Execute each step in order.\n\n")
	sb.WriteString("**To run the complete workflow**, follow the steps below. Each step has detailed instructions in its own rule.\n\n")

	sb.WriteString("## Workflow Steps\n\n")

	for _, step := range data.Steps {
		sb.WriteString(fmt.Sprintf("### Step %d: %s\n\n", step.Order, step.Name))
		if step.Description != "" {
			sb.WriteString(fmt.Sprintf("%s\n\n", step.Description))
		}
		// Reference the step rule using the provider-specific prefix
		sb.WriteString(fmt.Sprintf("**Invoke**: %s%s\n\n", ruleRefPrefix, step.RuleName))
		sb.WriteString("---\n\n")
	}

	sb.WriteString("## Execution Instructions\n\n")
	sb.WriteString("1. Start with Step 1 and complete it fully before moving to the next step\n")
	sb.WriteString("2. Each step may require user input or produce artifacts\n")
	sb.WriteString("3. Steps build on previous outputs, so order matters\n")
	sb.WriteString("4. If a step references a file that doesn't exist, complete the prerequisite step first\n")

	return sb.String()
}

// NormalizeWorkflowName converts a folder name to a display name
func NormalizeWorkflowName(name string) string {
	// Replace underscores and hyphens with spaces
	display := strings.ReplaceAll(name, "_", " ")
	display = strings.ReplaceAll(display, "-", " ")

	// Title case
	words := strings.Fields(display)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, " ")
}

// ExtractStepDescription tries to get a brief description from step content
func ExtractStepDescription(content string) string {
	lines := strings.Split(content, "\n")

	// Look for first paragraph after any heading
	foundHeading := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines
		if trimmed == "" {
			continue
		}

		// Track if we've passed a heading
		if strings.HasPrefix(trimmed, "#") {
			foundHeading = true
			continue
		}

		// Return first non-heading, non-empty line after a heading (or at start)
		if foundHeading || !strings.HasPrefix(trimmed, "#") {
			// Clean up and truncate
			desc := trimmed
			// Remove markdown formatting
			desc = strings.TrimPrefix(desc, "- ")
			desc = strings.TrimPrefix(desc, "* ")

			// Truncate if too long
			if len(desc) > 150 {
				if idx := strings.Index(desc, ". "); idx > 0 && idx < 150 {
					desc = desc[:idx+1]
				} else {
					desc = desc[:147] + "..."
				}
			}

			return desc
		}
	}

	return ""
}
