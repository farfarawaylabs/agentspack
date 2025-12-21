package wizard

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/huh"
)

// ClaudeCodeMode represents how tech stack rules should be generated for Claude Code
type ClaudeCodeMode string

const (
	ClaudeCodeModeRules  ClaudeCodeMode = "rules"
	ClaudeCodeModeSkills ClaudeCodeMode = "skills"
)

// Config holds the user's selections from the wizard
type Config struct {
	Providers       []string
	TechStacks      []string
	GenerateBase    bool           // Whether to generate the base file (CLAUDE.md, AGENTS.md, etc.)
	OutputDir       string
	ClaudeCodeMode  ClaudeCodeMode // Only used when claude-code is selected
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

	ClaudeCodeModeOptions = []huh.Option[string]{
		huh.NewOption("Rule files (always loaded, path-scoped)", string(ClaudeCodeModeRules)),
		huh.NewOption("Skills (loaded on-demand by Claude)", string(ClaudeCodeModeSkills)),
	}

	DefaultOutputDir = "./dist/agentspack"
)

// Run executes the interactive wizard and returns the user's configuration
func Run() (*Config, error) {
	config := &Config{
		OutputDir:    DefaultOutputDir,
		GenerateBase: true, // default to yes
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

	// Step 2: If Claude Code was selected, ask about rules vs skills
	if containsProvider(config.Providers, "claude-code") {
		var modeStr string = string(ClaudeCodeModeRules) // default

		claudeForm := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Claude Code: How should tech stack guidelines be generated?").
					Description("Rules are always loaded; Skills are loaded on-demand when relevant").
					Options(ClaudeCodeModeOptions...).
					Value(&modeStr),
			),
		)

		err = claudeForm.Run()
		if err != nil {
			return nil, fmt.Errorf("wizard error: %w", err)
		}

		config.ClaudeCodeMode = ClaudeCodeMode(modeStr)
	}

	// Step 3: Select tech stacks, base file, and output directory
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

	// Clean up the output path
	config.OutputDir = filepath.Clean(config.OutputDir)

	return config, nil
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

// PrintSummary displays the user's selections
func PrintSummary(config *Config) {
	fmt.Println()
	fmt.Println("=== Configuration Summary ===")
	fmt.Println()
	fmt.Printf("Providers:   %v\n", formatList(config.Providers))
	fmt.Printf("Tech Stacks: %v\n", formatList(config.TechStacks))
	fmt.Printf("Base file:   %v\n", boolToYesNo(config.GenerateBase))
	fmt.Printf("Output:      %s\n", config.OutputDir)
	if containsProvider(config.Providers, "claude-code") {
		fmt.Printf("Claude Code: %s mode\n", config.ClaudeCodeMode)
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
