package wizard

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agentspack/agentspack/internal/catalog"
	"github.com/agentspack/agentspack/internal/content"
	"github.com/charmbracelet/huh"
)

type GenerationMode string

const (
	GenerationModeFull GenerationMode = "full"
	GenerationModeAdd  GenerationMode = "add"
)

// GuidelinesMode controls how tech stack guidelines are generated for providers
// that support both rules and skills formats.
type GuidelinesMode string

const (
	GuidelinesModeRules  GuidelinesMode = "rules"
	GuidelinesModeSkills GuidelinesMode = "skills"
)

// SkillInvocationProfile controls whether generated skills are auto-invokable,
// user-invokable, or both.
type SkillInvocationProfile string

const (
	SkillInvocationDual       SkillInvocationProfile = "dual"
	SkillInvocationManualOnly SkillInvocationProfile = "manual-only"
	SkillInvocationAutoOnly   SkillInvocationProfile = "auto-only"
)

type ConflictPolicy string

const (
	ConflictPolicyError        ConflictPolicy = "error"
	ConflictPolicySkipExisting ConflictPolicy = "skip-existing"
)

type SelectiveInstallCategory string

const (
	SelectiveCategoryBaseSkills     SelectiveInstallCategory = "base-skills"
	SelectiveCategoryWorkflows      SelectiveInstallCategory = "workflows"
	SelectiveCategorySystemCommands SelectiveInstallCategory = "system-commands"
)

// SyncMode represents how changes should be applied to target repos
type SyncMode string

const (
	SyncModePR    SyncMode = "pr"
	SyncModeMerge SyncMode = "merge"
)

// Config holds the user's selections from the wizard
type Config struct {
	Mode              GenerationMode
	Providers         []string
	TechStacks        []string
	GenerateBase      bool // Whether to generate the base file (CLAUDE.md, AGENTS.md, etc.)
	OutputDir         string
	GuidelinesMode    GuidelinesMode         // Used when cursor or claude-code is selected
	InvocationProfile SkillInvocationProfile // Used when generating skills
	ConflictPolicy    ConflictPolicy

	SelectedBaseSkills     []string
	SelectedWorkflows      []string
	SelectedSystemCommands []string

	// GitHub sync options
	SyncToGitHub bool     // Whether to sync generated files to GitHub repos
	SyncMode     SyncMode // "pr" or "merge"
	TargetBranch string   // Branch to create PR against or merge into (default: "main")
}

// Available options
var (
	AvailableProviders = []huh.Option[string]{
		huh.NewOption("Cursor", "cursor"),
		huh.NewOption("Claude Code", "claude-code"),
		huh.NewOption("Codex", "codex"),
	}

	AvailableTechStacks = []huh.Option[string]{
		huh.NewOption("Backend", "backend"),
		huh.NewOption("React", "react"),
	}

	GuidelinesModeOptions = []huh.Option[string]{
		huh.NewOption("Rule files (always loaded, path-scoped)", string(GuidelinesModeRules)),
		huh.NewOption("Skills (loaded on-demand when relevant)", string(GuidelinesModeSkills)),
	}

	InvocationProfileOptions = []huh.Option[string]{
		huh.NewOption("Dual: model auto-use + user command invocation", string(SkillInvocationDual)),
		huh.NewOption("Manual only: user command invocation only", string(SkillInvocationManualOnly)),
		huh.NewOption("Auto only: model auto-use only", string(SkillInvocationAutoOnly)),
	}

	SelectiveInstallCategoryOptions = []huh.Option[string]{
		huh.NewOption("Base skills", string(SelectiveCategoryBaseSkills)),
		huh.NewOption("Workflows", string(SelectiveCategoryWorkflows)),
		huh.NewOption("System commands", string(SelectiveCategorySystemCommands)),
	}

	SyncModeOptions = []huh.Option[string]{
		huh.NewOption("Create Pull Request (for review)", string(SyncModePR)),
		huh.NewOption("Merge directly to branch", string(SyncModeMerge)),
	}

	DefaultOutputDir    = "./dist/agentspack"
	DefaultTargetBranch = "main"
	SyncReposFile       = "sync_repos.md"
)

// Run executes the interactive wizard and returns the user's configuration
func Run() (*Config, error) {
	config := &Config{
		Mode:              GenerationModeFull,
		OutputDir:         DefaultOutputDir,
		GenerateBase:      true, // default to yes
		GuidelinesMode:    GuidelinesModeRules,
		InvocationProfile: SkillInvocationDual,
		ConflictPolicy:    ConflictPolicyError,
	}

	// Step 1: Select providers
	providersForm := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select providers to generate for").
				Description("Choose one or more AI coding providers").
				Options(AvailableProviders...).
				Value(&config.Providers).
				Validate(func(selected []string) error {
					if len(selected) == 0 {
						return errors.New("please select at least one provider")
					}
					return nil
				}),
		),
	)

	err := providersForm.Run()
	if err != nil {
		return nil, fmt.Errorf("wizard error: %w", err)
	}

	// Step 2: If Cursor or Claude Code was selected, ask about rules vs skills
	if containsProvider(config.Providers, "claude-code") || containsProvider(config.Providers, "cursor") {
		var modeStr string = string(GuidelinesModeRules) // default

		claudeForm := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("How should tech stack guidelines be generated?").
					Description("Applies to selected providers that support both formats (Cursor and Claude Code)").
					Options(GuidelinesModeOptions...).
					Value(&modeStr),
			),
		)

		err = claudeForm.Run()
		if err != nil {
			return nil, fmt.Errorf("wizard error: %w", err)
		}

		config.GuidelinesMode = GuidelinesMode(modeStr)
	}

	// Step 3: If skills are generated, ask invocation profile.
	if shouldAskInvocationProfile(config) {
		var profileStr string = string(SkillInvocationDual) // default

		invocationForm := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("How should generated skills be invocable?").
					Description("Controls auto-invocation by the model and command invocation by users").
					Options(InvocationProfileOptions...).
					Value(&profileStr),
			),
		)

		err = invocationForm.Run()
		if err != nil {
			return nil, fmt.Errorf("wizard error: %w", err)
		}
		config.InvocationProfile = SkillInvocationProfile(profileStr)
	}

	// Step 4: Select tech stacks, base file, and output directory
	remainingForm := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select tech stacks").
				Description("Choose the tech stacks to include templates for").
				Options(AvailableTechStacks...).
				Value(&config.TechStacks).
				Validate(func(selected []string) error {
					if len(selected) == 0 {
						return errors.New("please select at least one tech stack")
					}
					return nil
				}),
		),
		huh.NewGroup(
			huh.NewConfirm().
				Title("Generate base instructions file?").
				Description("Creates CLAUDE.md or AGENTS.md with workflow guidelines").
				Value(&config.GenerateBase),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Output directory").
				Description("Where to write the generated files").
				Value(&config.OutputDir).
				Placeholder(DefaultOutputDir).
				Validate(func(s string) error {
					if s == "" {
						return errors.New("output directory cannot be empty")
					}
					return nil
				}),
		),
	)

	err = remainingForm.Run()
	if err != nil {
		return nil, fmt.Errorf("wizard error: %w", err)
	}

	// Expand and clean the output path
	config.OutputDir = expandPath(config.OutputDir)

	// Step 5: GitHub sync options (only if sync_repos.md exists)
	if syncReposFileExists() {
		syncForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Sync generated files to GitHub repositories?").
					Description("Repositories listed in " + SyncReposFile).
					Value(&config.SyncToGitHub),
			),
		)

		err = syncForm.Run()
		if err != nil {
			return nil, fmt.Errorf("wizard error: %w", err)
		}

		// If user wants to sync, ask for mode and branch
		if config.SyncToGitHub {
			var syncModeStr string = string(SyncModePR) // default to PR
			config.TargetBranch = DefaultTargetBranch

			syncOptionsForm := huh.NewForm(
				huh.NewGroup(
					huh.NewSelect[string]().
						Title("How should changes be applied?").
						Options(SyncModeOptions...).
						Value(&syncModeStr),
				),
				huh.NewGroup(
					huh.NewInput().
						Title("Target branch for PR/merge").
						Value(&config.TargetBranch).
						Placeholder(DefaultTargetBranch),
				),
			)

			err = syncOptionsForm.Run()
			if err != nil {
				return nil, fmt.Errorf("wizard error: %w", err)
			}

			config.SyncMode = SyncMode(syncModeStr)
		}
	}

	return config, nil
}

func RunAdd(fs content.FileSystem) (*Config, error) {
	baseSkills, err := catalog.ListBaseSkills(fs)
	if err != nil {
		return nil, fmt.Errorf("failed to discover base skills: %w", err)
	}
	workflows, err := catalog.ListWorkflows(fs)
	if err != nil {
		return nil, fmt.Errorf("failed to discover workflows: %w", err)
	}
	systemCommands, err := catalog.ListSystemCommands(fs)
	if err != nil {
		return nil, fmt.Errorf("failed to discover system commands: %w", err)
	}

	availableCategories := buildSelectiveCategoryOptions(baseSkills, workflows, systemCommands)
	if len(availableCategories) == 0 {
		return nil, errors.New("no base skills, workflows, or system commands are available to install")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve current working directory: %w", err)
	}

	config := &Config{
		Mode:              GenerationModeAdd,
		OutputDir:         cwd,
		InvocationProfile: SkillInvocationDual,
		ConflictPolicy:    ConflictPolicySkipExisting,
	}

	providersForm := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select providers already used in this repo").
				Description("Only the selected providers will receive the new items").
				Options(AvailableProviders...).
				Value(&config.Providers).
				Validate(func(selected []string) error {
					if len(selected) == 0 {
						return errors.New("please select at least one provider")
					}
					return nil
				}),
		),
	)
	if err := providersForm.Run(); err != nil {
		return nil, fmt.Errorf("wizard error: %w", err)
	}

	selectedCategories := []string{}
	categoryForm := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("What would you like to add?").
				Description("Choose one or more categories to install into the current repo").
				Options(availableCategories...).
				Value(&selectedCategories).
				Validate(func(selected []string) error {
					if len(selected) == 0 {
						return errors.New("please select at least one category")
					}
					return nil
				}),
		),
	)
	if err := categoryForm.Run(); err != nil {
		return nil, fmt.Errorf("wizard error: %w", err)
	}

	if containsCategory(selectedCategories, SelectiveCategoryBaseSkills) {
		options := buildBaseSkillOptions(baseSkills)
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title("Select base skills to install").
					Options(options...).
					Value(&config.SelectedBaseSkills).
					Validate(requireSelection("please select at least one base skill")),
			),
		)
		if err := form.Run(); err != nil {
			return nil, fmt.Errorf("wizard error: %w", err)
		}
	}

	if containsCategory(selectedCategories, SelectiveCategoryWorkflows) {
		options := buildWorkflowOptions(workflows)
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title("Select workflows to install").
					Options(options...).
					Value(&config.SelectedWorkflows).
					Validate(requireSelection("please select at least one workflow")),
			),
		)
		if err := form.Run(); err != nil {
			return nil, fmt.Errorf("wizard error: %w", err)
		}
	}

	if containsCategory(selectedCategories, SelectiveCategorySystemCommands) {
		options := buildSystemCommandOptions(systemCommands)
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title("Select system commands to install").
					Options(options...).
					Value(&config.SelectedSystemCommands).
					Validate(requireSelection("please select at least one system command")),
			),
		)
		if err := form.Run(); err != nil {
			return nil, fmt.Errorf("wizard error: %w", err)
		}
	}

	if shouldAskSelectiveInvocationProfile(config) {
		var profileStr string = string(SkillInvocationDual)
		invocationForm := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("How should generated skills be invocable?").
					Description("Applies when the selected providers/categories generate skills").
					Options(InvocationProfileOptions...).
					Value(&profileStr),
			),
		)
		if err := invocationForm.Run(); err != nil {
			return nil, fmt.Errorf("wizard error: %w", err)
		}
		config.InvocationProfile = SkillInvocationProfile(profileStr)
	}

	return config, nil
}

// expandPath expands ~ to home directory and handles absolute paths
func expandPath(path string) string {
	// First clean the path
	path = filepath.Clean(path)

	// Expand ~ to home directory
	if strings.HasPrefix(path, "~") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				path = homeDir
			} else if strings.HasPrefix(path, "~/") {
				path = filepath.Join(homeDir, path[2:])
			}
		}
	}

	// If path is already absolute (starts with /), use it as-is
	// Otherwise, it's relative and will be resolved relative to cwd by the caller
	return path
}

// containsProvider checks if a provider is in the selected list
func containsProvider(providers []string, target string) bool {
	for _, p := range providers {
		if p == target {
			return true
		}
	}
	return false
}

func shouldAskInvocationProfile(config *Config) bool {
	if config.Mode == GenerationModeAdd {
		return shouldAskSelectiveInvocationProfile(config)
	}
	if containsProvider(config.Providers, "codex") {
		return true
	}
	hasCursorOrClaude := containsProvider(config.Providers, "cursor") || containsProvider(config.Providers, "claude-code")
	return hasCursorOrClaude && config.GuidelinesMode == GuidelinesModeSkills
}

func shouldAskSelectiveInvocationProfile(config *Config) bool {
	if containsProvider(config.Providers, "codex") && (len(config.SelectedBaseSkills) > 0 || len(config.SelectedWorkflows) > 0 || len(config.SelectedSystemCommands) > 0) {
		return true
	}
	hasCursorOrClaude := containsProvider(config.Providers, "cursor") || containsProvider(config.Providers, "claude-code")
	return hasCursorOrClaude && len(config.SelectedBaseSkills) > 0
}

// syncReposFileExists checks if sync_repos.md exists in the current directory
func syncReposFileExists() bool {
	info, err := os.Stat(SyncReposFile)
	return err == nil && !info.IsDir()
}

// PrintSummary displays the user's selections
func PrintSummary(config *Config) {
	fmt.Println()
	fmt.Println("=== Configuration Summary ===")
	fmt.Println()
	if config.Mode == GenerationModeAdd {
		fmt.Printf("Mode:        add selective content\n")
		fmt.Printf("Providers:   %v\n", formatList(config.Providers))
		fmt.Printf("Repo Root:   %s\n", config.OutputDir)
		fmt.Printf("Base Skills: %v\n", formatList(config.SelectedBaseSkills))
		fmt.Printf("Workflows:   %v\n", formatList(config.SelectedWorkflows))
		fmt.Printf("Commands:    %v\n", formatList(config.SelectedSystemCommands))
		if shouldAskSelectiveInvocationProfile(config) {
			fmt.Printf("Skills:      %s invocation\n", config.InvocationProfile)
		}
		fmt.Printf("Conflicts:   %s\n", config.ConflictPolicy)
		fmt.Println()
		return
	}
	fmt.Printf("Providers:   %v\n", formatList(config.Providers))
	fmt.Printf("Tech Stacks: %v\n", formatList(config.TechStacks))
	fmt.Printf("Base file:   %v\n", boolToYesNo(config.GenerateBase))
	fmt.Printf("Output:      %s\n", config.OutputDir)
	if containsProvider(config.Providers, "claude-code") || containsProvider(config.Providers, "cursor") {
		fmt.Printf("Guidelines: %s mode\n", config.GuidelinesMode)
	}
	if shouldAskInvocationProfile(config) {
		fmt.Printf("Skills:     %s invocation\n", config.InvocationProfile)
	}
	if config.SyncToGitHub {
		syncModeDesc := "PR"
		if config.SyncMode == SyncModeMerge {
			syncModeDesc = "merge"
		}
		fmt.Printf("GitHub Sync: Yes (%s to %s)\n", syncModeDesc, config.TargetBranch)
	}
	fmt.Println()
}

func boolToYesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func formatList(items []string) string {
	if len(items) == 0 {
		return "(none)"
	}
	result := ""
	for i, item := range items {
		if i > 0 {
			result += ", "
		}
		result += item
	}
	return result
}

func containsCategory(categories []string, target SelectiveInstallCategory) bool {
	for _, category := range categories {
		if category == string(target) {
			return true
		}
	}
	return false
}

func requireSelection(message string) func([]string) error {
	return func(selected []string) error {
		if len(selected) == 0 {
			return errors.New(message)
		}
		return nil
	}
}

func buildSelectiveCategoryOptions(baseSkills []catalog.BaseSkill, workflows []catalog.Workflow, systemCommands []catalog.SystemCommand) []huh.Option[string] {
	options := make([]huh.Option[string], 0, len(SelectiveInstallCategoryOptions))
	if len(baseSkills) > 0 {
		options = append(options, huh.NewOption(fmt.Sprintf("Base skills (%d available)", len(baseSkills)), string(SelectiveCategoryBaseSkills)))
	}
	if len(workflows) > 0 {
		options = append(options, huh.NewOption(fmt.Sprintf("Workflows (%d available)", len(workflows)), string(SelectiveCategoryWorkflows)))
	}
	if len(systemCommands) > 0 {
		options = append(options, huh.NewOption(fmt.Sprintf("System commands (%d available)", len(systemCommands)), string(SelectiveCategorySystemCommands)))
	}
	return options
}

func buildBaseSkillOptions(skills []catalog.BaseSkill) []huh.Option[string] {
	options := make([]huh.Option[string], 0, len(skills))
	for _, skill := range skills {
		label := skill.Name
		if skill.Description != "" {
			label = fmt.Sprintf("%s - %s", skill.Name, skill.Description)
		}
		options = append(options, huh.NewOption(label, skill.Name))
	}
	return options
}

func buildWorkflowOptions(workflows []catalog.Workflow) []huh.Option[string] {
	options := make([]huh.Option[string], 0, len(workflows))
	for _, workflow := range workflows {
		label := workflow.Name
		if workflow.Description != "" {
			label = fmt.Sprintf("%s - %s", workflow.Name, workflow.Description)
		}
		options = append(options, huh.NewOption(label, workflow.Name))
	}
	return options
}

func buildSystemCommandOptions(commands []catalog.SystemCommand) []huh.Option[string] {
	options := make([]huh.Option[string], 0, len(commands))
	for _, command := range commands {
		label := command.Name
		if command.Description != "" {
			label = fmt.Sprintf("%s - %s", command.Name, command.Description)
		}
		options = append(options, huh.NewOption(label, command.Name))
	}
	return options
}
