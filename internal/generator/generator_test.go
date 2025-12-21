package generator

import (
	"os"
	"path/filepath"
	"testing"

	_ "github.com/agentspack/agentspack/internal/providers"
	"github.com/agentspack/agentspack/internal/wizard"
)

func TestGeneratorWithEmbeddedFS(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agentspack-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &wizard.Config{
		Providers:      []string{"claude-code", "cursor", "codex"},
		TechStacks:     []string{"react", "backend"},
		GenerateBase:   true,
		OutputDir:      tmpDir,
		ClaudeCodeMode: wizard.ClaudeCodeModeRules,
	}

	// Test with non-existent local system dir (should use embedded)
	gen := New(config, "/nonexistent/system")
	if err := gen.Run(); err != nil {
		t.Fatalf("Error with embedded filesystem: %v", err)
	}

	// Verify some expected files exist
	expectedFiles := []string{
		"CLAUDE.md",
		"AGENTS.md",
		".claude/rules/global.md",
		".cursor/rules/global/RULE.md",
		".codex/skills/react-guidelines/SKILL.md",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s does not exist", file)
		}
	}
}

func TestGeneratorWithLocalFS(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agentspack-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &wizard.Config{
		Providers:      []string{"claude-code"},
		TechStacks:     []string{"react"},
		GenerateBase:   true,
		OutputDir:      tmpDir,
		ClaudeCodeMode: wizard.ClaudeCodeModeSkills,
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get cwd: %v", err)
	}

	// The system folder should be at agentspack/system from the project root
	systemDir := filepath.Join(cwd, "..", "..", "system")

	// Check if local system dir exists
	if _, err := os.Stat(systemDir); os.IsNotExist(err) {
		t.Skip("Local system directory not found, skipping local FS test")
	}

	gen := New(config, systemDir)
	if err := gen.Run(); err != nil {
		t.Fatalf("Error with local filesystem: %v", err)
	}

	// Verify CLAUDE.md exists
	claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")
	if _, err := os.Stat(claudeMDPath); os.IsNotExist(err) {
		t.Error("Expected CLAUDE.md does not exist")
	}
}
