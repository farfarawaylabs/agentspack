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
	Register(&CodexProvider{})
}

// CodexProvider generates Codex-specific files (AGENTS.md and skills)
type CodexProvider struct{}

func (p *CodexProvider) Name() string {
	return "codex"
}

// codexStackConfigs maps wizard tech stack choices to their skill configurations
var codexStackConfigs = map[string]struct {
	SourcePath       string
	SkillName        string
	Description      string
	ShortDescription string
}{
	"react": {
		SourcePath:       "frontend/react",
		SkillName:        "react-guidelines",
		Description:      "Best practices for React development. Use when building React components, managing state, creating UI layouts, or working with JSX/TSX files.",
		ShortDescription: "React component and UI development guidelines",
	},
	"backend": {
		SourcePath:       "backend",
		SkillName:        "backend-guidelines",
		Description:      "Best practices for backend development. Use when building APIs, working with databases, designing data models, or implementing server-side logic.",
		ShortDescription: "Backend API and database development guidelines",
	},
}

func (p *CodexProvider) Generate(config *wizard.Config, fs content.FileSystem, outputDir string) error {
	if config.Mode == wizard.GenerationModeAdd {
		return p.generateSelectedContent(config, fs, outputDir)
	}

	// Create the .agents/skills directory structure (Codex scans .agents/skills)
	skillsDir := filepath.Join(outputDir, ".agents", "skills")

	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return fmt.Errorf("failed to create codex skills directory: %w", err)
	}

	// 1. Generate AGENTS.md with global rules (always applied)
	if err := p.generateAgentsMD(fs, outputDir, config.GenerateBase); err != nil {
		return fmt.Errorf("failed to generate AGENTS.md: %w", err)
	}

	invocationSettings, invocationWarning := resolveInvocationSettings(config.InvocationProfile, p.Name())
	if invocationWarning != "" {
		fmt.Println(invocationWarning)
	}

	// 2. Generate tech stack skills
	for _, stack := range config.TechStacks {
		if IsDocPack(stack) {
			if err := p.installDocPack(fs, outputDir, stack); err != nil {
				return fmt.Errorf("failed to install doc pack '%s': %w", stack, err)
			}
			continue
		}
		stackConfig, ok := codexStackConfigs[stack]
		if !ok {
			fmt.Printf("Warning: no configuration for tech stack '%s', skipping\n", stack)
			continue
		}
		if err := p.generateStackSkill(fs, skillsDir, stack, stackConfig, invocationSettings); err != nil {
			return fmt.Errorf("failed to generate %s skill: %w", stack, err)
		}
	}

	// 3. Generate reusable base skills for all providers.
	if err := p.generateBaseSkills(fs, skillsDir, invocationSettings, nil, wizard.ConflictPolicyError); err != nil {
		return fmt.Errorf("failed to generate base skills: %w", err)
	}

	// 4. Generate agent skills
	if err := p.generateAgentSkills(fs, skillsDir, invocationSettings); err != nil {
		return fmt.Errorf("failed to generate agent skills: %w", err)
	}

	// 5. Generate workflow skills
	if err := p.generateWorkflowSkills(fs, skillsDir, invocationSettings, nil, wizard.ConflictPolicyError); err != nil {
		return fmt.Errorf("failed to generate workflow skills: %w", err)
	}

	// 6. Generate direct command skills from system/commands.
	if err := p.generateSystemCommandSkills(fs, skillsDir, invocationSettings, nil, wizard.ConflictPolicyError); err != nil {
		return fmt.Errorf("failed to generate system command skills: %w", err)
	}

	return nil
}

func (p *CodexProvider) generateSelectedContent(config *wizard.Config, fs content.FileSystem, outputDir string) error {
	skillsDir := filepath.Join(outputDir, ".agents", "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return fmt.Errorf("failed to create codex skills directory: %w", err)
	}

	invocationSettings, invocationWarning := resolveInvocationSettings(config.InvocationProfile, p.Name())
	if invocationWarning != "" {
		fmt.Println(invocationWarning)
	}

	if len(config.SelectedBaseSkills) > 0 {
		if err := p.generateBaseSkills(fs, skillsDir, invocationSettings, selectedSet(config.SelectedBaseSkills), config.ConflictPolicy); err != nil {
			return fmt.Errorf("failed to generate selected base skills: %w", err)
		}
	}

	if len(config.SelectedWorkflows) > 0 {
		if err := p.generateWorkflowSkills(fs, skillsDir, invocationSettings, selectedSet(config.SelectedWorkflows), config.ConflictPolicy); err != nil {
			return fmt.Errorf("failed to generate selected workflow skills: %w", err)
		}
	}

	if len(config.SelectedSystemCommands) > 0 {
		if err := p.generateSystemCommandSkills(fs, skillsDir, invocationSettings, selectedSet(config.SelectedSystemCommands), config.ConflictPolicy); err != nil {
			return fmt.Errorf("failed to generate selected system command skills: %w", err)
		}
	}

	if len(config.SelectedDocPacks) > 0 {
		for _, packName := range config.SelectedDocPacks {
			if err := p.installDocPack(fs, outputDir, packName); err != nil {
				return fmt.Errorf("failed to install doc pack '%s': %w", packName, err)
			}
		}
	}

	return nil
}

// generateAgentsMD creates the AGENTS.md file with base content + global rules
func (p *CodexProvider) generateAgentsMD(fs content.FileSystem, outputDir string, includeBase bool) error {
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

	// If includeBase, prepend base.md + Codex.md content
	if includeBase {
		// Read base.md
		baseContent, err := fs.ReadFile("system/base/base.md")
		if err != nil {
			return fmt.Errorf("failed to read base.md: %w", err)
		}

		// Read Codex.md
		providerContent, err := fs.ReadFile("system/base/Codex.md")
		if err != nil {
			return fmt.Errorf("failed to read Codex.md: %w", err)
		}

		contentBuilder.Write(baseContent)
		contentBuilder.WriteString("\n\n")
		contentBuilder.Write(providerContent)
		contentBuilder.WriteString("\n\n---\n\n")
	}

	contentBuilder.WriteString("# Project Guidelines\n\n")
	contentBuilder.WriteString("These guidelines apply to all work in this project.\n\n")

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

	// Write AGENTS.md at the output root
	outputPath := filepath.Join(outputDir, "AGENTS.md")
	if err := os.WriteFile(outputPath, []byte(contentBuilder.String()), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outputPath, err)
	}

	fmt.Printf("  Created: %s\n", outputPath)
	return nil
}

// generateStackSkill creates a skill for a tech stack by concatenating its rules
func (p *CodexProvider) generateStackSkill(fs content.FileSystem, skillsDir, stackName string, config struct {
	SourcePath       string
	SkillName        string
	Description      string
	ShortDescription string
}, invocationSettings SkillInvocationSettings) error {
	// Create skill directory
	skillDir := filepath.Join(skillsDir, config.SkillName)
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
		files = mergeUniquePaths(files, subFiles)
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

	skillMarkdown, err := buildSkillMarkdown(SkillDocOptions{
		Name:                   config.SkillName,
		Description:            config.Description,
		Heading:                fmt.Sprintf("%s Guidelines", templates.NormalizeWorkflowName(stackName)),
		Body:                   allContent.String(),
		DisableModelInvocation: invocationSettings.DisableModelInvocation,
		Metadata: map[string]any{
			"short-description": config.ShortDescription,
		},
	})
	if err != nil {
		return err
	}

	// Write SKILL.md
	outputPath := filepath.Join(skillDir, "SKILL.md")
	if _, err := writeSkillFile(
		outputPath,
		[]byte(skillMarkdown),
		config.SkillName,
		fmt.Sprintf("skill '%s' conflicts with an existing generated skill at %s", config.SkillName, outputPath),
		wizard.ConflictPolicyError,
	); err != nil {
		return err
	}
	return nil
}

// installDocPack copies a doc pack's reference files into
// .agentspack/docs/<pack>/ and appends its directive to AGENTS.md so the
// guidance is always in-context for Codex.
func (p *CodexProvider) installDocPack(fs content.FileSystem, outputDir, packName string) error {
	pack, err := LoadDocPack(fs, packName)
	if err != nil {
		return err
	}
	if err := CopyDocPackDocs(fs, pack, outputDir); err != nil {
		return err
	}
	baseFilePath := filepath.Join(outputDir, "AGENTS.md")
	if err := AppendDirectiveToBaseFile(baseFilePath, pack); err != nil {
		return err
	}
	return nil
}

func (p *CodexProvider) generateBaseSkills(fs content.FileSystem, skillsDir string, invocationSettings SkillInvocationSettings, selectedNames map[string]struct{}, conflictPolicy wizard.ConflictPolicy) error {
	files, err := listBaseSkillTemplates(fs)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}

	for _, file := range files {
		skillName, skillMarkdown, err := buildBaseSkillFromTemplate(fs, file, invocationSettings, false)
		if err != nil {
			return err
		}
		if !isSelected(selectedNames, skillName) {
			continue
		}

		skillDir := filepath.Join(skillsDir, skillName)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			return err
		}

		outputPath := filepath.Join(skillDir, "SKILL.md")
		if _, err := writeSkillFile(
			outputPath,
			[]byte(skillMarkdown),
			skillName,
			fmt.Sprintf("skill '%s' conflicts with an existing generated skill at %s", skillName, outputPath),
			conflictPolicy,
		); err != nil {
			return err
		}
	}

	return nil
}

// generateAgentSkills creates skills for each agent
func (p *CodexProvider) generateAgentSkills(fs content.FileSystem, skillsDir string, invocationSettings SkillInvocationSettings) error {
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
		files = mergeUniquePaths(files, subFiles)
	}

	for _, file := range files {
		if err := p.createAgentSkill(fs, file, skillsDir, invocationSettings); err != nil {
			return err
		}
	}

	return nil
}

// createAgentSkill creates a Codex skill from an agent markdown file
func (p *CodexProvider) createAgentSkill(fs content.FileSystem, sourcePath, skillsDir string, invocationSettings SkillInvocationSettings) error {
	fileContent, err := fs.ReadFile(sourcePath)
	if err != nil {
		return err
	}

	contentStr := string(fileContent)

	// Parse frontmatter to extract agent metadata
	agentName, description, bodyContent := parseAgentFrontmatter(contentStr)
	if strings.TrimSpace(bodyContent) == "" {
		bodyContent = contentStr
	}

	// Generate skill name from filename if not in frontmatter
	if agentName == "" {
		agentName = agentNameFromSourcePath(sourcePath)
	}

	// Normalize the name
	skillName := strings.ReplaceAll(agentName, "_", "-")

	// Create skill directory
	skillDir := filepath.Join(skillsDir, skillName)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return err
	}

	// Use extracted description or generate one
	if description == "" {
		description = fmt.Sprintf("Agent: %s", agentName)
	}

	skillMarkdown, err := buildSkillMarkdown(SkillDocOptions{
		Name:                   skillName,
		Description:            description,
		Body:                   bodyContent,
		DisableModelInvocation: invocationSettings.DisableModelInvocation,
		Metadata: map[string]any{
			"short-description": fmt.Sprintf("%s agent", templates.NormalizeWorkflowName(agentName)),
		},
	})
	if err != nil {
		return err
	}

	// Write SKILL.md
	outputPath := filepath.Join(skillDir, "SKILL.md")
	if _, err := writeSkillFile(
		outputPath,
		[]byte(skillMarkdown),
		skillName,
		fmt.Sprintf("skill '%s' conflicts with an existing generated skill at %s", skillName, outputPath),
		wizard.ConflictPolicyError,
	); err != nil {
		return err
	}
	return nil
}

// generateWorkflowSkills creates skills for workflows
func (p *CodexProvider) generateWorkflowSkills(fs content.FileSystem, skillsDir string, invocationSettings SkillInvocationSettings, selectedWorkflows map[string]struct{}, conflictPolicy wizard.ConflictPolicy) error {
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
		if !isSelected(selectedWorkflows, workflowName) {
			continue
		}

		if err := p.generateSingleWorkflowSkill(fs, skillsDir, workflowName, invocationSettings, conflictPolicy); err != nil {
			return fmt.Errorf("failed to generate workflow skill '%s': %w", workflowName, err)
		}
	}

	return nil
}

// generateSingleWorkflowSkill creates step skills and an orchestrator skill for one workflow
func (p *CodexProvider) generateSingleWorkflowSkill(fs content.FileSystem, skillsDir, workflowName string, invocationSettings SkillInvocationSettings, conflictPolicy wizard.ConflictPolicy) error {
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

		// Generate the skill name for this step
		var skillName string
		if matches != nil {
			skillName = fmt.Sprintf("workflow-%s-%02d-%s", workflowName, order, stepName)
		} else {
			skillName = fmt.Sprintf("workflow-%s-%s", workflowName, stepName)
		}

		// Read file to extract description
		fileContent, err := fs.ReadFile(file)
		if err != nil {
			return err
		}

		description := templates.ExtractStepDescription(string(fileContent))

		steps = append(steps, templates.WorkflowStep{
			Order:       order,
			Name:        templates.NormalizeWorkflowName(stepName),
			RuleName:    skillName, // reusing RuleName for skill name
			Description: description,
		})

		// Create the step skill
		if err := p.createWorkflowStepSkill(fs, file, skillsDir, skillName, workflowName, invocationSettings, conflictPolicy); err != nil {
			return err
		}
	}

	// Sort steps by order
	sort.Slice(steps, func(i, j int) bool {
		return steps[i].Order < steps[j].Order
	})

	// Create the workflow orchestrator skill
	return p.createWorkflowOrchestratorSkill(skillsDir, workflowName, steps, invocationSettings, conflictPolicy)
}

// createWorkflowStepSkill creates a Codex skill for a single workflow step
func (p *CodexProvider) createWorkflowStepSkill(fs content.FileSystem, sourcePath, skillsDir, skillName, workflowName string, invocationSettings SkillInvocationSettings, conflictPolicy wizard.ConflictPolicy) error {
	fileContent, err := fs.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	_, _, stepBody := parseAgentFrontmatter(string(fileContent))
	if strings.TrimSpace(stepBody) == "" {
		stepBody = string(fileContent)
	}

	// Create skill directory
	skillDir := filepath.Join(skillsDir, skillName)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return err
	}

	// Extract description from first heading
	description := extractDescription(stepBody, "Workflow step", skillName)

	skillMarkdown, err := buildSkillMarkdown(SkillDocOptions{
		Name:                   skillName,
		Description:            description,
		Body:                   stepBody,
		DisableModelInvocation: invocationSettings.DisableModelInvocation,
		Metadata: map[string]any{
			"short-description": fmt.Sprintf("%s workflow step", templates.NormalizeWorkflowName(workflowName)),
		},
	})
	if err != nil {
		return err
	}

	// Write SKILL.md
	outputPath := filepath.Join(skillDir, "SKILL.md")
	if _, err := writeSkillFile(
		outputPath,
		[]byte(skillMarkdown),
		skillName,
		fmt.Sprintf("skill '%s' conflicts with an existing generated skill at %s", skillName, outputPath),
		conflictPolicy,
	); err != nil {
		return err
	}
	return nil
}

// createWorkflowOrchestratorSkill creates the main workflow skill that references all steps
func (p *CodexProvider) createWorkflowOrchestratorSkill(skillsDir, workflowName string, steps []templates.WorkflowStep, invocationSettings SkillInvocationSettings, conflictPolicy wizard.ConflictPolicy) error {
	skillName := fmt.Sprintf("workflow-%s", workflowName)

	// Create skill directory
	skillDir := filepath.Join(skillsDir, skillName)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return err
	}

	// Build the orchestrator data
	data := templates.WorkflowOrchestratorData{
		WorkflowName: workflowName,
		DisplayName:  templates.NormalizeWorkflowName(workflowName),
		Description:  fmt.Sprintf("Complete %s workflow with %d steps. Follow each step in order.", templates.NormalizeWorkflowName(workflowName), len(steps)),
		Steps:        steps,
	}

	// Generate orchestrator content using the template
	// For Codex, use $ to reference other skills
	orchestratorContent := templates.GenerateWorkflowOrchestrator(data, "$")

	// Generate workflow-specific description
	workflowDescription := generateWorkflowDescription(workflowName, len(steps))
	skillMarkdown, err := buildSkillMarkdown(SkillDocOptions{
		Name:                   skillName,
		Description:            workflowDescription,
		Body:                   orchestratorContent,
		DisableModelInvocation: invocationSettings.DisableModelInvocation,
		Metadata: map[string]any{
			"short-description": fmt.Sprintf("Complete %s workflow (%d steps)", templates.NormalizeWorkflowName(workflowName), len(steps)),
		},
	})
	if err != nil {
		return err
	}

	// Write SKILL.md
	outputPath := filepath.Join(skillDir, "SKILL.md")
	if _, err := writeSkillFile(
		outputPath,
		[]byte(skillMarkdown),
		skillName,
		fmt.Sprintf("skill '%s' conflicts with an existing generated skill at %s", skillName, outputPath),
		conflictPolicy,
	); err != nil {
		return err
	}
	return nil
}

// generateWorkflowDescription creates a helpful description for workflow orchestrators
func generateWorkflowDescription(workflowName string, stepCount int) string {
	// Map known workflow names to helpful descriptions
	descriptions := map[string]string{
		"planning": "Run the complete product planning workflow. Use when starting a new project or feature to create PRD, conduct market research, design UX/UI, and generate development tasks.",
	}

	if desc, ok := descriptions[workflowName]; ok {
		return desc
	}

	// Default description for unknown workflows
	displayName := templates.NormalizeWorkflowName(workflowName)
	return fmt.Sprintf("Run the complete %s workflow with %d sequential steps. Use when you need to execute this structured process from start to finish.", displayName, stepCount)
}

func (p *CodexProvider) generateSystemCommandSkills(fs content.FileSystem, skillsDir string, invocationSettings SkillInvocationSettings, selectedCommands map[string]struct{}, conflictPolicy wizard.ConflictPolicy) error {
	commands, err := listSystemCommandTemplates(fs)
	if err != nil {
		return err
	}
	if len(commands) == 0 {
		return nil
	}

	for _, command := range commands {
		if !isSelected(selectedCommands, command.Name) {
			continue
		}
		skillDir := filepath.Join(skillsDir, command.Name)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			return err
		}

		skillMarkdown, err := buildSkillMarkdown(SkillDocOptions{
			Name:                   command.Name,
			Description:            command.Description,
			Body:                   command.Body,
			DisableModelInvocation: invocationSettings.DisableModelInvocation,
			Metadata: map[string]any{
				"short-description": fmt.Sprintf("Command: %s", command.Name),
			},
		})
		if err != nil {
			return err
		}

		outputPath := filepath.Join(skillDir, "SKILL.md")
		if _, err := writeSkillFile(
			outputPath,
			[]byte(skillMarkdown),
			command.Name,
			fmt.Sprintf("skill '%s' conflicts with an existing generated skill at %s", command.Name, outputPath),
			conflictPolicy,
		); err != nil {
			return err
		}
	}

	return nil
}

func writeSkillFile(outputPath string, content []byte, skillName, conflictMessage string, conflictPolicy wizard.ConflictPolicy) (bool, error) {
	_ = skillName
	return writeFileWithConflictPolicy(outputPath, content, conflictMessage, conflictPolicy)
}
