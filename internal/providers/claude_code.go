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
	Register(&ClaudeCodeProvider{})
}

// ClaudeCodeProvider generates Claude Code-specific files (rules, skills, agents, commands)
type ClaudeCodeProvider struct{}

func (p *ClaudeCodeProvider) Name() string {
	return "claude-code"
}

// claudeCodeStackConfigs maps wizard tech stack choices to their configurations
var claudeCodeStackConfigs = map[string]struct {
	SourcePath       string
	Name             string
	Globs            []string // For rules mode (path scoping)
	Description      string   // For skills mode
	ShortDescription string
}{
	"react": {
		SourcePath:       "frontend/react",
		Name:             "react",
		Globs:            []string{"**/*.tsx", "**/*.jsx", "src/components/**", "src/pages/**", "src/app/**"},
		Description:      "Best practices for React development. Use when building React components, managing state with hooks, creating reusable UI elements, or working with JSX/TSX files.",
		ShortDescription: "React component and UI development guidelines",
	},
	"backend": {
		SourcePath:       "backend",
		Name:             "backend",
		Globs:            []string{"**/*.go", "**/*.py", "src/api/**", "src/server/**", "api/**", "server/**"},
		Description:      "Best practices for backend development. Use when building APIs, working with databases, designing data models, implementing authentication, or writing server-side logic.",
		ShortDescription: "Backend API and database development guidelines",
	},
}

func (p *ClaudeCodeProvider) Generate(config *wizard.Config, fs content.FileSystem, outputDir string) error {
	// Create the .claude directory structure
	claudeDir := filepath.Join(outputDir, ".claude")
	rulesDir := filepath.Join(claudeDir, "rules")
	agentsDir := filepath.Join(claudeDir, "agents")
	commandsDir := filepath.Join(claudeDir, "commands")
	skillsDir := filepath.Join(claudeDir, "skills")

	// Create necessary directories
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		return fmt.Errorf("failed to create claude rules directory: %w", err)
	}

	// 0. Generate CLAUDE.md base file if requested
	if config.GenerateBase {
		if err := p.generateBaseFile(fs, outputDir); err != nil {
			return fmt.Errorf("failed to generate CLAUDE.md: %w", err)
		}
	}

	// 1. Generate global rules (always as a rule file)
	if err := p.generateGlobalRules(fs, rulesDir); err != nil {
		return fmt.Errorf("failed to generate global rules: %w", err)
	}

	// 2. Generate tech stack content based on user's mode choice
	if config.ClaudeCodeMode == wizard.ClaudeCodeModeSkills {
		// Skills mode: generate skills for tech stacks
		if err := os.MkdirAll(skillsDir, 0755); err != nil {
			return fmt.Errorf("failed to create skills directory: %w", err)
		}
		for _, stack := range config.TechStacks {
			stackConfig, ok := claudeCodeStackConfigs[stack]
			if !ok {
				fmt.Printf("Warning: no configuration for tech stack '%s', skipping\n", stack)
				continue
			}
			if err := p.generateStackSkill(fs, skillsDir, stack, stackConfig); err != nil {
				return fmt.Errorf("failed to generate %s skill: %w", stack, err)
			}
		}
	} else {
		// Rules mode: generate rule files with path scoping
		for _, stack := range config.TechStacks {
			stackConfig, ok := claudeCodeStackConfigs[stack]
			if !ok {
				fmt.Printf("Warning: no configuration for tech stack '%s', skipping\n", stack)
				continue
			}
			if err := p.generateStackRules(fs, rulesDir, stack, stackConfig); err != nil {
				return fmt.Errorf("failed to generate %s rules: %w", stack, err)
			}
		}
	}

	// 3. Generate agents as sub-agents
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create agents directory: %w", err)
	}
	if err := p.generateSubAgents(fs, agentsDir); err != nil {
		return fmt.Errorf("failed to generate sub-agents: %w", err)
	}

	// 4. Generate workflows as slash commands
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		return fmt.Errorf("failed to create commands directory: %w", err)
	}
	if err := p.generateWorkflowCommands(fs, commandsDir); err != nil {
		return fmt.Errorf("failed to generate workflow commands: %w", err)
	}

	return nil
}

// generateBaseFile creates the CLAUDE.md file from base.md + Claude.md
func (p *ClaudeCodeProvider) generateBaseFile(fs content.FileSystem, outputDir string) error {
	// Read base.md
	baseContent, err := fs.ReadFile("system/base/base.md")
	if err != nil {
		return fmt.Errorf("failed to read base.md: %w", err)
	}

	// Read Claude.md
	providerContent, err := fs.ReadFile("system/base/Claude.md")
	if err != nil {
		return fmt.Errorf("failed to read Claude.md: %w", err)
	}

	// Concatenate the content
	var contentBuilder strings.Builder
	contentBuilder.Write(baseContent)
	contentBuilder.WriteString("\n\n")
	contentBuilder.Write(providerContent)

	// Write CLAUDE.md to output directory root
	outputPath := filepath.Join(outputDir, "CLAUDE.md")
	if err := os.WriteFile(outputPath, []byte(contentBuilder.String()), 0644); err != nil {
		return fmt.Errorf("failed to write CLAUDE.md: %w", err)
	}

	fmt.Printf("  Created: %s\n", outputPath)
	return nil
}

// generateGlobalRules creates a single global.md rule file with all global rules
func (p *ClaudeCodeProvider) generateGlobalRules(fs content.FileSystem, rulesDir string) error {
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

	// Write frontmatter for Claude Code rules
	contentBuilder.WriteString("---\n")
	contentBuilder.WriteString("description: Global coding standards and best practices that apply to all files\n")
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

		contentBuilder.Write(fileContent)
		contentBuilder.WriteString("\n")
	}

	// Write global.md
	outputPath := filepath.Join(rulesDir, "global.md")
	if err := os.WriteFile(outputPath, []byte(contentBuilder.String()), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outputPath, err)
	}

	fmt.Printf("  Created: %s\n", outputPath)
	return nil
}

// generateStackRules creates rule files with path scoping for tech stacks
func (p *ClaudeCodeProvider) generateStackRules(fs content.FileSystem, rulesDir, stackName string, config struct {
	SourcePath       string
	Name             string
	Globs            []string
	Description      string
	ShortDescription string
}) error {
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

	if len(files) == 0 {
		return nil
	}

	// Collect all content
	var allContent strings.Builder
	for i, file := range files {
		fileContent, err := fs.ReadFile(file)
		if err != nil {
			return err
		}

		if i > 0 {
			allContent.WriteString("\n---\n\n")
		}

		allContent.Write(fileContent)
		allContent.WriteString("\n")
	}

	// Build rule content with frontmatter
	var ruleContent strings.Builder

	ruleContent.WriteString("---\n")
	ruleContent.WriteString(fmt.Sprintf("description: %s development guidelines\n", templates.NormalizeWorkflowName(stackName)))
	ruleContent.WriteString("alwaysApply: false\n")
	ruleContent.WriteString("globs:\n")
	for _, glob := range config.Globs {
		ruleContent.WriteString(fmt.Sprintf("  - %s\n", glob))
	}
	ruleContent.WriteString("---\n\n")

	ruleContent.WriteString(fmt.Sprintf("# %s Guidelines\n\n", templates.NormalizeWorkflowName(stackName)))
	ruleContent.WriteString(allContent.String())

	// Write the rule file
	outputPath := filepath.Join(rulesDir, fmt.Sprintf("%s.md", config.Name))
	if err := os.WriteFile(outputPath, []byte(ruleContent.String()), 0644); err != nil {
		return err
	}

	fmt.Printf("  Created: %s\n", outputPath)
	return nil
}

// generateStackSkill creates a skill for a tech stack with good metadata
func (p *ClaudeCodeProvider) generateStackSkill(fs content.FileSystem, skillsDir, stackName string, config struct {
	SourcePath       string
	Name             string
	Globs            []string
	Description      string
	ShortDescription string
}) error {
	// Create skill directory
	skillDir := filepath.Join(skillsDir, fmt.Sprintf("%s-guidelines", config.Name))
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return err
	}

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

	if len(files) == 0 {
		return nil
	}

	// Collect all content
	var allContent strings.Builder
	for i, file := range files {
		fileContent, err := fs.ReadFile(file)
		if err != nil {
			return err
		}

		if i > 0 {
			allContent.WriteString("\n---\n\n")
		}

		allContent.Write(fileContent)
		allContent.WriteString("\n")
	}

	// Build SKILL.md content with frontmatter
	var skillContent strings.Builder

	skillContent.WriteString("---\n")
	skillContent.WriteString(fmt.Sprintf("name: %s-guidelines\n", config.Name))
	skillContent.WriteString(fmt.Sprintf("description: %s\n", config.Description))
	skillContent.WriteString("---\n\n")

	skillContent.WriteString(fmt.Sprintf("# %s Guidelines\n\n", templates.NormalizeWorkflowName(stackName)))
	skillContent.WriteString(allContent.String())

	// Write SKILL.md
	outputPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(outputPath, []byte(skillContent.String()), 0644); err != nil {
		return err
	}

	fmt.Printf("  Created: %s\n", outputPath)
	return nil
}

// generateSubAgents creates sub-agent files from agent templates
func (p *ClaudeCodeProvider) generateSubAgents(fs content.FileSystem, agentsDir string) error {
	// Check if agents directory exists
	if _, err := fs.Stat("system/agents"); err != nil {
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
		if err := p.createSubAgent(fs, file, agentsDir); err != nil {
			return err
		}
	}

	return nil
}

// createSubAgent creates a Claude Code sub-agent from an agent markdown file
func (p *ClaudeCodeProvider) createSubAgent(fs content.FileSystem, sourcePath, agentsDir string) error {
	fileContent, err := fs.ReadFile(sourcePath)
	if err != nil {
		return err
	}

	contentStr := string(fileContent)

	// Parse frontmatter to extract agent metadata
	agentName, description, bodyContent := parseAgentFrontmatter(contentStr)

	// Generate agent name from filename if not in frontmatter
	if agentName == "" {
		baseName := filepath.Base(sourcePath)
		agentName = strings.TrimSuffix(baseName, ".md")
	}

	// Normalize the name (replace underscores with hyphens)
	agentName = strings.ReplaceAll(agentName, "_", "-")

	// Build sub-agent content with frontmatter
	var agentContent strings.Builder

	// Use extracted description or generate one
	if description == "" {
		description = fmt.Sprintf("Specialized agent for %s tasks", templates.NormalizeWorkflowName(agentName))
	}

	agentContent.WriteString("---\n")
	agentContent.WriteString(fmt.Sprintf("name: %s\n", agentName))
	agentContent.WriteString(fmt.Sprintf("description: %s\n", escapeYAMLString(description)))
	agentContent.WriteString("---\n\n")
	agentContent.WriteString(bodyContent)

	// Write the agent file
	outputPath := filepath.Join(agentsDir, fmt.Sprintf("%s.md", agentName))
	if err := os.WriteFile(outputPath, []byte(agentContent.String()), 0644); err != nil {
		return err
	}

	fmt.Printf("  Created: %s\n", outputPath)
	return nil
}

// generateWorkflowCommands creates slash commands for workflows
func (p *ClaudeCodeProvider) generateWorkflowCommands(fs content.FileSystem, commandsDir string) error {
	// Check if workflows directory exists
	if _, err := fs.Stat("system/workflows"); err != nil {
		return nil
	}

	// Find all workflow folders
	entries, err := fs.ReadDir("system/workflows")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		workflowName := entry.Name()

		if err := p.generateSingleWorkflowCommand(fs, commandsDir, workflowName); err != nil {
			return fmt.Errorf("failed to generate workflow command '%s': %w", workflowName, err)
		}
	}

	return nil
}

// generateSingleWorkflowCommand creates step commands and an orchestrator command for one workflow
func (p *ClaudeCodeProvider) generateSingleWorkflowCommand(fs content.FileSystem, commandsDir, workflowName string) error {
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
			order = 99
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
			RuleName:    commandName, // reusing RuleName for command name
			Description: description,
		})

		// Create the step command
		if err := p.createWorkflowStepCommand(fs, file, commandsDir, commandName, workflowName); err != nil {
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

// createWorkflowStepCommand creates a slash command for a single workflow step
func (p *ClaudeCodeProvider) createWorkflowStepCommand(fs content.FileSystem, sourcePath, commandsDir, commandName, workflowName string) error {
	fileContent, err := fs.ReadFile(sourcePath)
	if err != nil {
		return err
	}

	// Build command content (slash commands don't need frontmatter in Claude Code)
	var commandContent strings.Builder

	commandContent.WriteString(fmt.Sprintf("# %s Workflow Step\n\n", templates.NormalizeWorkflowName(workflowName)))
	commandContent.Write(fileContent)

	// Write the command file
	outputPath := filepath.Join(commandsDir, fmt.Sprintf("%s.md", commandName))
	if err := os.WriteFile(outputPath, []byte(commandContent.String()), 0644); err != nil {
		return err
	}

	fmt.Printf("  Created: %s\n", outputPath)
	return nil
}

// createWorkflowOrchestratorCommand creates the main workflow command that guides through all steps
func (p *ClaudeCodeProvider) createWorkflowOrchestratorCommand(commandsDir, workflowName string, steps []templates.WorkflowStep) error {
	// Build the orchestrator data
	data := templates.WorkflowOrchestratorData{
		WorkflowName: workflowName,
		DisplayName:  templates.NormalizeWorkflowName(workflowName),
		Description:  fmt.Sprintf("Complete %s workflow with %d steps. Follow each step in order.", templates.NormalizeWorkflowName(workflowName), len(steps)),
		Steps:        steps,
	}

	// Generate orchestrator content using the template
	// For Claude Code slash commands, use / to reference other commands
	orchestratorContent := templates.GenerateWorkflowOrchestrator(data, "/")

	// Write the command file
	outputPath := filepath.Join(commandsDir, fmt.Sprintf("%s.md", workflowName))
	if err := os.WriteFile(outputPath, []byte(orchestratorContent), 0644); err != nil {
		return err
	}

	fmt.Printf("  Created: %s\n", outputPath)
	return nil
}
