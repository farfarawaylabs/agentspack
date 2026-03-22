package generator

import (
	"os"
	"path/filepath"
	"strings"
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
		GuidelinesMode: wizard.GuidelinesModeRules,
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
		".agents/skills/react-guidelines/SKILL.md",
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
		GuidelinesMode: wizard.GuidelinesModeSkills,
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

func TestGeneratorSelectiveInstallSkipsExistingAndFiltersSelections(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agentspack-selective-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	existingSkillPath := filepath.Join(tmpDir, ".cursor", "skills", "frontend-design", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(existingSkillPath), 0755); err != nil {
		t.Fatalf("Failed to create existing skill directory: %v", err)
	}
	existingContent := "existing skill content"
	if err := os.WriteFile(existingSkillPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to write existing skill: %v", err)
	}

	config := &wizard.Config{
		Mode:                   wizard.GenerationModeAdd,
		Providers:              []string{"cursor", "claude-code", "codex"},
		OutputDir:              tmpDir,
		InvocationProfile:      wizard.SkillInvocationDual,
		ConflictPolicy:         wizard.ConflictPolicySkipExisting,
		SelectedBaseSkills:     []string{"frontend-design"},
		SelectedWorkflows:      []string{"planning"},
		SelectedSystemCommands: []string{"remember"},
	}

	gen := New(config, "/nonexistent/system")
	if err := gen.Run(); err != nil {
		t.Fatalf("Selective install failed: %v", err)
	}

	content, err := os.ReadFile(existingSkillPath)
	if err != nil {
		t.Fatalf("Failed to read existing skill after selective install: %v", err)
	}
	if string(content) != existingContent {
		t.Fatalf("Expected existing skill to be preserved, got %q", string(content))
	}

	expectedPaths := []string{
		filepath.Join(tmpDir, ".claude", "skills", "frontend-design", "SKILL.md"),
		filepath.Join(tmpDir, ".claude", "commands", "planning.md"),
		filepath.Join(tmpDir, ".claude", "commands", "remember.md"),
		filepath.Join(tmpDir, ".cursor", "commands", "planning.md"),
		filepath.Join(tmpDir, ".cursor", "commands", "remember.md"),
		filepath.Join(tmpDir, ".agents", "skills", "frontend-design", "SKILL.md"),
		filepath.Join(tmpDir, ".agents", "skills", "workflow-planning", "SKILL.md"),
		filepath.Join(tmpDir, ".agents", "skills", "remember", "SKILL.md"),
	}
	for _, path := range expectedPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s does not exist", path)
		}
	}

	unexpectedPaths := []string{
		filepath.Join(tmpDir, ".cursor", "skills", "cloudflare-platform-development", "SKILL.md"),
		filepath.Join(tmpDir, ".cursor", "commands", "development.md"),
		filepath.Join(tmpDir, ".claude", "commands", "development.md"),
		filepath.Join(tmpDir, ".agents", "skills", "workflow-development", "SKILL.md"),
		filepath.Join(tmpDir, ".agents", "skills", "create-new-skill", "SKILL.md"),
	}
	for _, path := range unexpectedPaths {
		if _, err := os.Stat(path); err == nil {
			t.Errorf("Did not expect file %s to exist", path)
		}
	}

	cursorPlanningContent, err := os.ReadFile(filepath.Join(tmpDir, ".cursor", "commands", "planning.md"))
	if err != nil {
		t.Fatalf("Failed to read cursor planning command: %v", err)
	}
	if !strings.Contains(string(cursorPlanningContent), "/planning-01-create-prd-interactive") {
		t.Fatalf("Expected cursor planning orchestrator to reference planning steps")
	}
}
