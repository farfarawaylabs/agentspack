package wizard

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
)

// ClaudeCodeMode represents how tech stack rules should be generated for Claude Code
type ClaudeCodeMode string

const (
	ClaudeCodeModeRules  ClaudeCodeMode = "rules"
	ClaudeCodeModeSkills ClaudeCodeMode = "skills"
)

// SyncMode represents how changes should be applied to target repos
type SyncMode string

const (
	SyncModePR    SyncMode = "pr"
	SyncModeMerge SyncMode = "merge"
)

// Config holds the user's selections from the wizard
type Config struct {
	Providers      []string
	TechStacks     []string
	GenerateBase   bool           // Whether to generate the base file (CLAUDE.md, AGENTS.md, etc.)
	OutputDir      string
	ClaudeCodeMode ClaudeCodeMode // Only used when claude-code is selected

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

	ClaudeCodeModeOptions = []huh.Option[string]{
		huh.NewOption("Rule files (always loaded, path-scoped)", string(ClaudeCodeModeRules)),
		huh.NewOption("Skills (loaded on-demand by Claude)", string(ClaudeCodeModeSkills)),
	}

	SyncModeOptions = []huh.Option[string]{
		huh.NewOption("Create Pull Request (for review)", string(SyncModePR)),
		huh.NewOption("Merge directly to branch", string(SyncModeMerge)),
	}

	DefaultOutputDir   = "./dist/agentspack"
	DefaultTargetBranch = "main"
	SyncReposFile      = "sync_repos.md"
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

	// Expand and clean the output path
	config.OutputDir = expandPath(config.OutputDir)

	// Step 4: GitHub sync options (only if sync_repos.md exists)
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
	fmt.Printf("Providers:   %v\n", formatList(config.Providers))
	fmt.Printf("Tech Stacks: %v\n", formatList(config.TechStacks))
	fmt.Printf("Base file:   %v\n", boolToYesNo(config.GenerateBase))
	fmt.Printf("Output:      %s\n", config.OutputDir)
	if containsProvider(config.Providers, "claude-code") {
		fmt.Printf("Claude Code: %s mode\n", config.ClaudeCodeMode)
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
