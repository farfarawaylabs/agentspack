package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/agentspack/agentspack/internal/generator"
	_ "github.com/agentspack/agentspack/internal/providers" // Register providers
	"github.com/agentspack/agentspack/internal/wizard"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "agentspack",
	Short: "Generate provider-specific AI agent context files",
	Long: `agentspack is a CLI tool that helps developers generate provider-specific
AI agent context files (e.g., Cursor, Claude Code, Codex) from a built-in
library of markdown templates.`,
	Run: func(cmd *cobra.Command, args []string) {
		runWizard()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runWizard() {
	fmt.Println("Welcome to agentspack!")
	fmt.Println()

	config, err := wizard.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	wizard.PrintSummary(config)

	// Determine system directory (relative to binary location for now)
	// In MVP, we assume the system folder is next to the binary
	execPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding executable path: %v\n", err)
		os.Exit(1)
	}

	// Helper to check if a directory is a valid agentspack system directory
	// (must contain base/base.md to be valid)
	isValidSystemDir := func(dir string) bool {
		baseFile := filepath.Join(dir, "base", "base.md")
		info, err := os.Stat(baseFile)
		return err == nil && !info.IsDir()
	}

	systemDir := filepath.Join(filepath.Dir(execPath), "system")

	// If system dir doesn't exist or isn't valid next to binary, try current working directory
	if !isValidSystemDir(systemDir) {
		cwd, _ := os.Getwd()
		systemDir = filepath.Join(cwd, "system")
	}

	// Also check parent directory (for development when running from agentspack subfolder)
	if !isValidSystemDir(systemDir) {
		cwd, _ := os.Getwd()
		systemDir = filepath.Join(filepath.Dir(cwd), "system")
	}

	// If still not valid, set to empty string to force embedded mode
	if !isValidSystemDir(systemDir) {
		systemDir = ""
	}

	// Run the generator
	gen := generator.New(config, systemDir)
	if err := gen.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
