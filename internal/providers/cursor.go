package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/agentspack/agentspack/internal/content"
	"github.com/agentspack/agentspack/internal/templates"
	"github.com/agentspack/agentspack/internal/wizard"
)

func init() {
	Register(&CursorProvider{})
}

// CursorProvider generates Cursor-specific rule files
type CursorProvider struct{}

func (p *CursorProvider) Name() string {
	return "cursor"
}

// TechStackConfig defines how to handle a specific tech stack
type TechStackConfig struct {
	// SourcePath is the relative path under system/rules (e.g., "frontend/react")
	SourcePath string
	// Globs are the file patterns this stack applies to
	Globs []string
	// Description prefix for the rules
	Description string
}

// techStackConfigs maps wizard tech stack choices to their configurations
var techStackConfigs = map[string]TechStackConfig{
	"react": {
		SourcePath:  "frontend/react",
		Globs:       []string{"*.tsx", "*.jsx", "src/components/**", "src/pages/**", "src/app/**"},
		Description: "React component",
	},
	"backend": {
		SourcePath:  "backend",
		Globs:       []string{"*.go", "*.py", "*.ts", "src/api/**", "src/server/**", "api/**", "server/**"},
		Description: "Backend API",
	},
}

func (p *CursorProvider) Generate(config *wizard.Config, fs content.FileSystem, outputDir string) error {
	// Create the output directory structure directly in the user's chosen directory
	rulesDir := filepath.Join(outputDir, ".cursor", "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		return fmt.Errorf("failed to create cursor rules directory: %w", err)
	}

	// Create commands directory for workflows
	commandsDir := filepath.Join(outputDir, ".cursor", "commands")
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		return fmt.Errorf("failed to create cursor commands directory: %w", err)
	}

	// 0. Generate AGENTS.md base file if requested
	if config.GenerateBase {
		if err := p.generateBaseFile(fs, outputDir); err != nil {
			return fmt.Errorf("failed to generate AGENTS.md: %w", err)
		}
	}

	// 1. Generate global rules (concatenated)
	if err := p.generateGlobalRules(fs, rulesDir); err != nil {
		return fmt.Errorf("failed to generate global rules: %w", err)
	}

	// 2. Generate tech stack specific rules
	for _, stack := range config.TechStacks {
		stackConfig, ok := techStackConfigs[stack]
		if !ok {
			fmt.Printf("Warning: no configuration for tech stack '%s', skipping\n", stack)
			continue
		}
		if err := p.generateStackRules(fs, rulesDir, stack, stackConfig); err != nil {
			return fmt.Errorf("failed to generate %s rules: %w", stack, err)
		}
	}

	// 3. Generate agent rules
	if err := p.generateAgentRules(fs, rulesDir); err != nil {
		return fmt.Errorf("failed to generate agent rules: %w", err)
	}

	// 4. Generate workflow commands (Cursor supports /commands like Claude Code)
	if err := p.generateWorkflowCommands(fs, commandsDir); err != nil {
		return fmt.Errorf("failed to generate workflow commands: %w", err)
	}

	return nil
}

// generateBaseFile creates the AGENTS.md file from base.md + Cursor.md
func (p *CursorProvider) generateBaseFile(fs content.FileSystem, outputDir string) error {
	// Read base.md
	baseContent, err := fs.ReadFile("system/base/base.md")
	if err != nil {
		return fmt.Errorf("failed to read base.md: %w", err)
	}

	// Read Cursor.md
	providerContent, err := fs.ReadFile("system/base/Cursor.md")
	if err != nil {
		return fmt.Errorf("failed to read Cursor.md: %w", err)
	}

	// Concatenate the content
	var contentBuilder strings.Builder
	contentBuilder.Write(baseContent)
	contentBuilder.WriteString("\n\n")
	contentBuilder.Write(providerContent)

	// Write AGENTS.md to output directory root
	outputPath := filepath.Join(outputDir, "AGENTS.md")
	if err := os.WriteFile(outputPath, []byte(contentBuilder.String()), 0644); err != nil {
		return fmt.Errorf("failed to write AGENTS.md: %w", err)
	}

	fmt.Printf("  Created: %s\n", outputPath)
	return nil
}

// generateGlobalRules concatenates all global rules into a single RULE.md
func (p *CursorProvider) generateGlobalRules(fs content.FileSystem, cursorDir string) error {
	globalRuleDir := filepath.Join(cursorDir, "global")

	if err := os.MkdirAll(globalRuleDir, 0755); err != nil {
		return err
	}

	// Find all markdown files in global directory
	files, err := fs.Glob("system/rules/global/*.md")
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf("no global rule files found")
	}

	// Concatenate all files
	var contentBuilder strings.Builder

	// Write frontmatter
	contentBuilder.WriteString("---\n")
	contentBuilder.WriteString("description: \"Global coding standards and best practices\"\n")
	contentBuilder.WriteString("alwaysApply: true\n")
	contentBuilder.WriteString("---\n\n")

	contentBuilder.WriteString("# Global Coding Standards\n\n")
	contentBuilder.WriteString("These rules apply to all files in the project.\n\n")

	for i, file := range files {
		fileContent, err := fs.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file, err)
		}

		if i > 0 {
			contentBuilder.WriteString("\n---\n\n")
		}

		// Add the file content
		contentBuilder.Write(fileContent)
		contentBuilder.WriteString("\n")
	}

	// Write the concatenated file
	outputPath := filepath.Join(globalRuleDir, "RULE.md")
	if err := os.WriteFile(outputPath, []byte(contentBuilder.String()), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outputPath, err)
	}

	fmt.Printf("  Created: %s\n", outputPath)
	return nil
}

// generateStackRules creates individual rule files for each stack template
func (p *CursorProvider) generateStackRules(fs content.FileSystem, cursorDir, stackName string, config TechStackConfig) error {
	// Find all markdown files in the stack directory
	pattern := fmt.Sprintf("system/rules/%s/*.md", config.SourcePath)
	files, err := fs.Glob(pattern)
	if err != nil {
		return err
	}

	// Also check subdirectories
	subPattern := fmt.Sprintf("system/rules/%s/**/*.md", config.SourcePath)
	subFiles, err := fs.Glob(subPattern)
	if err == nil {
		files = append(files, subFiles...)
	}

	for _, file := range files {
		if err := p.createRuleFromFile(fs, file, cursorDir, stackName, config); err != nil {
			return err
		}
	}

	return nil
}

// createRuleFromFile creates a Cursor rule from a single source markdown file
func (p *CursorProvider) createRuleFromFile(fs content.FileSystem, sourcePath, cursorDir, stackName string, config TechStackConfig) error {
	// Read the source file
	fileContent, err := fs.ReadFile(sourcePath)
	if err != nil {
		return err
	}

	// Generate rule name from filename (replace underscores with hyphens)
	baseName := filepath.Base(sourcePath)
	ruleName := strings.TrimSuffix(baseName, ".md")
	ruleName = strings.ReplaceAll(ruleName, "_", "-")
	ruleName = fmt.Sprintf("%s-%s", stackName, ruleName)

	// Create the rule directory
	ruleDir := filepath.Join(cursorDir, ruleName)
	if err := os.MkdirAll(ruleDir, 0755); err != nil {
		return err
	}

	// Build the RULE.md content with frontmatter
	var ruleContent strings.Builder

	// Extract a description from the first heading or use filename
	description := extractDescription(string(fileContent), config.Description, ruleName)

	ruleContent.WriteString("---\n")
	ruleContent.WriteString(fmt.Sprintf("description: \"%s\"\n", description))
	ruleContent.WriteString("alwaysApply: false\n")
	ruleContent.WriteString(fmt.Sprintf("globs: %s\n", formatGlobs(config.Globs)))
	ruleContent.WriteString("---\n\n")
	ruleContent.Write(fileContent)

	// Write the rule file
	outputPath := filepath.Join(ruleDir, "RULE.md")
	if err := os.WriteFile(outputPath, []byte(ruleContent.String()), 0644); err != nil {
		return err
	}

	fmt.Printf("  Created: %s\n", outputPath)
	return nil
}

// extractDescription tries to get a meaningful description from the content
func extractDescription(content, prefix, fallback string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "## ") {
			// Remove ## and clean up
			desc := strings.TrimPrefix(line, "## ")
			return fmt.Sprintf("%s: %s", prefix, desc)
		}
		if strings.HasPrefix(line, "# ") {
			desc := strings.TrimPrefix(line, "# ")
			return fmt.Sprintf("%s: %s", prefix, desc)
		}
	}
	return fmt.Sprintf("%s guidelines: %s", prefix, fallback)
}

// formatGlobs converts a slice of globs to YAML array format
func formatGlobs(globs []string) string {
	if len(globs) == 0 {
		return "[]"
	}

	quoted := make([]string, len(globs))
	for i, g := range globs {
		quoted[i] = fmt.Sprintf("\"%s\"", g)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

// generateAgentRules creates individual rule files for each agent
func (p *CursorProvider) generateAgentRules(fs content.FileSystem, cursorDir string) error {
	// Check if agents directory exists
	if _, err := fs.Stat("system/agents"); err != nil {
		// No agents directory, skip silently
		return nil
	}

	// Find all markdown files in the agents directory
	files, err := fs.Glob("system/agents/*.md")
	if err != nil {
		return err
	}

	// Also check subdirectories (e.g., system/agents/backend/*.md)
	subFiles, err := fs.Glob("system/agents/**/*.md")
	if err == nil {
		files = append(files, subFiles...)
	}

	for _, file := range files {
		if err := p.createAgentRule(fs, file, cursorDir); err != nil {
			return err
		}
	}

	return nil
}

// createAgentRule creates a Cursor rule from an agent markdown file
func (p *CursorProvider) createAgentRule(fs content.FileSystem, sourcePath, cursorDir string) error {
	// Read the source file
	fileContent, err := fs.ReadFile(sourcePath)
	if err != nil {
		return err
	}

	contentStr := string(fileContent)

	// Parse frontmatter to extract agent metadata
	agentName, description, bodyContent := parseAgentFrontmatter(contentStr)

	// Generate rule name from filename if not in frontmatter
	if agentName == "" {
		baseName := filepath.Base(sourcePath)
		agentName = strings.TrimSuffix(baseName, ".md")
	}

	// Normalize the name (replace underscores with hyphens)
	ruleName := "agent-" + strings.ReplaceAll(agentName, "_", "-")

	// Create the rule directory
	ruleDir := filepath.Join(cursorDir, ruleName)
	if err := os.MkdirAll(ruleDir, 0755); err != nil {
		return err
	}

	// Build the RULE.md content with frontmatter
	var ruleContent strings.Builder

	// Use extracted description or generate one
	if description == "" {
		description = fmt.Sprintf("Agent: %s", agentName)
	}

	ruleContent.WriteString("---\n")
	ruleContent.WriteString(fmt.Sprintf("description: \"%s\"\n", escapeYAMLString(description)))
	ruleContent.WriteString("alwaysApply: false\n")
	ruleContent.WriteString("---\n\n")

	// Write the body content (without frontmatter)
	ruleContent.WriteString(bodyContent)

	// Write the rule file
	outputPath := filepath.Join(ruleDir, "RULE.md")
	if err := os.WriteFile(outputPath, []byte(ruleContent.String()), 0644); err != nil {
		return err
	}

	fmt.Printf("  Created: %s\n", outputPath)
	return nil
}

// parseAgentFrontmatter extracts name, description, and body from agent markdown
func parseAgentFrontmatter(content string) (name, description, body string) {
	// Check if content starts with frontmatter
	if !strings.HasPrefix(content, "---") {
		return "", "", content
	}

	// Find the end of frontmatter
	endIndex := strings.Index(content[3:], "---")
	if endIndex == -1 {
		return "", "", content
	}

	frontmatter := content[3 : endIndex+3]
	body = strings.TrimSpace(content[endIndex+6:])

	// Parse frontmatter fields
	lines := strings.Split(frontmatter, "\n")
	var descLines []string
	inDescription := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for name field
		if strings.HasPrefix(trimmed, "name:") {
			name = strings.TrimSpace(strings.TrimPrefix(trimmed, "name:"))
			inDescription = false
			continue
		}

		// Check for description field (can be multi-line)
		if strings.HasPrefix(trimmed, "description:") {
			descValue := strings.TrimSpace(strings.TrimPrefix(trimmed, "description:"))
			if descValue != "" && descValue != "|" {
				descLines = append(descLines, descValue)
			}
			inDescription = true
			continue
		}

		// Check for other fields that end description
		if strings.Contains(trimmed, ":") && !strings.HasPrefix(trimmed, "-") && !strings.HasPrefix(trimmed, " ") {
			inDescription = false
			continue
		}

		// Continue collecting description lines
		if inDescription && trimmed != "" {
			descLines = append(descLines, trimmed)
		}
	}

	// Join description lines, taking first meaningful line for Cursor
	if len(descLines) > 0 {
		// For Cursor, we want a concise description - take the first sentence or line
		fullDesc := strings.Join(descLines, " ")
		// Clean up and truncate if needed
		description = strings.TrimSpace(fullDesc)
		// Take first sentence if description is too long
		if len(description) > 200 {
			if idx := strings.Index(description, ". "); idx > 0 && idx < 200 {
				description = description[:idx+1]
			} else if len(description) > 200 {
				description = description[:197] + "..."
			}
		}
	}

	return name, description, body
}

// escapeYAMLString escapes special characters in a YAML string value
func escapeYAMLString(s string) string {
	// Replace double quotes with escaped quotes
	s = strings.ReplaceAll(s, "\"", "\\\"")
	// Replace newlines
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

// generateWorkflowCommands creates commands for workflow steps and orchestrators
// Cursor supports /commands similar to Claude Code, so workflows map naturally to commands
func (p *CursorProvider) generateWorkflowCommands(fs content.FileSystem, commandsDir string) error {
	// Check if workflows directory exists
	if _, err := fs.Stat("system/workflows"); err != nil {
		// No workflows directory, skip silently
		return nil
	}

	// Find all workflow folders (direct subdirectories)
	entries, err := fs.ReadDir("system/workflows")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		workflowName := entry.Name()

		// Generate commands for this workflow
		if err := p.generateSingleWorkflowCommands(fs, commandsDir, workflowName); err != nil {
			return fmt.Errorf("failed to generate workflow '%s': %w", workflowName, err)
		}
	}

	return nil
}

// generateSingleWorkflowCommands creates step commands and an orchestrator for one workflow
func (p *CursorProvider) generateSingleWorkflowCommands(fs content.FileSystem, commandsDir, workflowName string) error {
	// Find all markdown files in the workflow directory
	pattern := fmt.Sprintf("system/workflows/%s/*.md", workflowName)
	files, err := fs.Glob(pattern)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return nil
	}

	// Parse and sort workflow steps
	steps := make([]templates.WorkflowStep, 0, len(files))
	stepPattern := regexp.MustCompile(`^(\d+)[_-](.+)\.md$`)

	for _, file := range files {
		baseName := filepath.Base(file)
		matches := stepPattern.FindStringSubmatch(baseName)

		var order int
		var stepName string

		if matches != nil {
			order, _ = strconv.Atoi(matches[1])
			stepName = matches[2]
		} else {
			// No number prefix, use filename as-is
			order = 99 // Put unnumbered files at end
			stepName = strings.TrimSuffix(baseName, ".md")
		}

		// Normalize step name
		stepName = strings.ReplaceAll(stepName, "_", "-")

		// Generate the command name for this step
		commandName := fmt.Sprintf("%s-%02d-%s", workflowName, order, stepName)

		// Read file to extract description
		fileContent, err := fs.ReadFile(file)
		if err != nil {
			return err
		}

		description := templates.ExtractStepDescription(string(fileContent))

		steps = append(steps, templates.WorkflowStep{
			Order:       order,
			Name:        templates.NormalizeWorkflowName(stepName),
			RuleName:    commandName, // Reusing RuleName field for command name
			Description: description,
		})

		// Create the step command
		if err := p.createWorkflowStepCommand(fs, file, commandsDir, commandName); err != nil {
			return err
		}
	}

	// Sort steps by order
	sort.Slice(steps, func(i, j int) bool {
		return steps[i].Order < steps[j].Order
	})

	// Create the workflow orchestrator command
	return p.createWorkflowOrchestratorCommand(commandsDir, workflowName, steps)
}

// createWorkflowStepCommand creates a Cursor command for a single workflow step
// Commands are simple markdown files without YAML frontmatter
func (p *CursorProvider) createWorkflowStepCommand(fs content.FileSystem, sourcePath, commandsDir, commandName string) error {
	fileContent, err := fs.ReadFile(sourcePath)
	if err != nil {
		return err
	}

	// Commands are flat markdown files - no YAML frontmatter needed
	outputPath := filepath.Join(commandsDir, commandName+".md")
	if err := os.WriteFile(outputPath, fileContent, 0644); err != nil {
		return err
	}

	fmt.Printf("  Created: %s\n", outputPath)
	return nil
}

// createWorkflowOrchestratorCommand creates the main workflow command that references all steps
func (p *CursorProvider) createWorkflowOrchestratorCommand(commandsDir, workflowName string, steps []templates.WorkflowStep) error {
	// Build the orchestrator data
	data := templates.WorkflowOrchestratorData{
		WorkflowName: workflowName,
		DisplayName:  templates.NormalizeWorkflowName(workflowName),
		Description:  fmt.Sprintf("Complete %s workflow with %d steps. Follow each step in order.", templates.NormalizeWorkflowName(workflowName), len(steps)),
		Steps:        steps,
	}

	// Generate orchestrator content using the template
	// For Cursor commands, use / to reference other commands
	orchestratorContent := templates.GenerateWorkflowOrchestrator(data, "/")

	// Write the command file (no YAML frontmatter for commands)
	outputPath := filepath.Join(commandsDir, workflowName+".md")
	if err := os.WriteFile(outputPath, []byte(orchestratorContent), 0644); err != nil {
		return err
	}

	fmt.Printf("  Created: %s\n", outputPath)
	return nil
}
