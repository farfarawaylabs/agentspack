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
		// Cloudflare base skills live under system/base-skills/cloudflare/
		// and get name-joined with "-" by the base-skill loader. Verify the
		// moved platform skill and the first-class agents-sdk-core skill
		// are emitted for every provider.
		".claude/skills/cloudflare-platform-development/SKILL.md",
		".cursor/skills/cloudflare-platform-development/SKILL.md",
		".agents/skills/cloudflare-platform-development/SKILL.md",
		".claude/skills/cloudflare-agents-sdk-core/SKILL.md",
		".cursor/skills/cloudflare-agents-sdk-core/SKILL.md",
		".agents/skills/cloudflare-agents-sdk-core/SKILL.md",
		".claude/skills/cloudflare-agent-state-and-sql/SKILL.md",
		".cursor/skills/cloudflare-agent-state-and-sql/SKILL.md",
		".agents/skills/cloudflare-agent-state-and-sql/SKILL.md",
		".claude/skills/cloudflare-callable-methods/SKILL.md",
		".cursor/skills/cloudflare-callable-methods/SKILL.md",
		".agents/skills/cloudflare-callable-methods/SKILL.md",
		".claude/skills/cloudflare-scheduled-tasks/SKILL.md",
		".cursor/skills/cloudflare-scheduled-tasks/SKILL.md",
		".agents/skills/cloudflare-scheduled-tasks/SKILL.md",
		".claude/skills/cloudflare-queued-tasks/SKILL.md",
		".cursor/skills/cloudflare-queued-tasks/SKILL.md",
		".agents/skills/cloudflare-queued-tasks/SKILL.md",
		".claude/skills/cloudflare-retries-with-idempotency/SKILL.md",
		".cursor/skills/cloudflare-retries-with-idempotency/SKILL.md",
		".agents/skills/cloudflare-retries-with-idempotency/SKILL.md",
		".claude/skills/cloudflare-durable-execution-fibers/SKILL.md",
		".cursor/skills/cloudflare-durable-execution-fibers/SKILL.md",
		".agents/skills/cloudflare-durable-execution-fibers/SKILL.md",
		".claude/skills/cloudflare-workflows-with-agents/SKILL.md",
		".cursor/skills/cloudflare-workflows-with-agents/SKILL.md",
		".agents/skills/cloudflare-workflows-with-agents/SKILL.md",
		".claude/skills/cloudflare-ai-search-rag/SKILL.md",
		".cursor/skills/cloudflare-ai-search-rag/SKILL.md",
		".agents/skills/cloudflare-ai-search-rag/SKILL.md",
		".claude/skills/cloudflare-browser-run/SKILL.md",
		".cursor/skills/cloudflare-browser-run/SKILL.md",
		".agents/skills/cloudflare-browser-run/SKILL.md",
		".claude/skills/cloudflare-sandboxes/SKILL.md",
		".cursor/skills/cloudflare-sandboxes/SKILL.md",
		".agents/skills/cloudflare-sandboxes/SKILL.md",
		".claude/skills/cloudflare-email-agents/SKILL.md",
		".cursor/skills/cloudflare-email-agents/SKILL.md",
		".agents/skills/cloudflare-email-agents/SKILL.md",
		".claude/skills/cloudflare-mcp-tools/SKILL.md",
		".cursor/skills/cloudflare-mcp-tools/SKILL.md",
		".agents/skills/cloudflare-mcp-tools/SKILL.md",
		".claude/skills/cloudflare-agent-security/SKILL.md",
		".cursor/skills/cloudflare-agent-security/SKILL.md",
		".agents/skills/cloudflare-agent-security/SKILL.md",
		".claude/skills/cloudflare-durable-objects-facets-dynamic-workers/SKILL.md",
		".cursor/skills/cloudflare-durable-objects-facets-dynamic-workers/SKILL.md",
		".agents/skills/cloudflare-durable-objects-facets-dynamic-workers/SKILL.md",
		// Tier-4 recipe skills under system/base-skills/cloudflare/recipes/.
		// Frontmatter names (e.g. `cloudflare-long-running-assistant-pattern`)
		// override the path-derived names, so nested path is invisible here.
		".claude/skills/cloudflare-long-running-assistant-pattern/SKILL.md",
		".cursor/skills/cloudflare-long-running-assistant-pattern/SKILL.md",
		".agents/skills/cloudflare-long-running-assistant-pattern/SKILL.md",
		".claude/skills/cloudflare-ui-controlled-agent-pattern/SKILL.md",
		".cursor/skills/cloudflare-ui-controlled-agent-pattern/SKILL.md",
		".agents/skills/cloudflare-ui-controlled-agent-pattern/SKILL.md",
		".claude/skills/cloudflare-durable-research-loop-pattern/SKILL.md",
		".cursor/skills/cloudflare-durable-research-loop-pattern/SKILL.md",
		".agents/skills/cloudflare-durable-research-loop-pattern/SKILL.md",
		".claude/skills/cloudflare-scheduled-publishing-pattern/SKILL.md",
		".cursor/skills/cloudflare-scheduled-publishing-pattern/SKILL.md",
		".agents/skills/cloudflare-scheduled-publishing-pattern/SKILL.md",
		".claude/skills/cloudflare-agent-generated-miniapp-pattern/SKILL.md",
		".cursor/skills/cloudflare-agent-generated-miniapp-pattern/SKILL.md",
		".agents/skills/cloudflare-agent-generated-miniapp-pattern/SKILL.md",
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

func TestGeneratorInstallsCloudflareDocPack(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agentspack-docpack-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &wizard.Config{
		Providers:      []string{"claude-code", "cursor", "codex"},
		TechStacks:     []string{"react", "cloudflare-agent-stack"},
		GenerateBase:   true,
		OutputDir:      tmpDir,
		GuidelinesMode: wizard.GuidelinesModeRules,
	}

	gen := New(config, "/nonexistent/system")
	if err := gen.Run(); err != nil {
		t.Fatalf("Doc pack generation failed: %v", err)
	}

	// Reference docs should be copied for every provider into the shared
	// .agentspack/docs path.
	expectedDocs := []string{
		".agentspack/docs/cloudflare-agent-stack/00_START_HERE.md",
		".agentspack/docs/cloudflare-agent-stack/01_DECISION_GUIDE.md",
		".agentspack/docs/cloudflare-agent-stack/08_DURABLE_EXECUTION_FIBERS.md",
		".agentspack/docs/cloudflare-agent-stack/19_CHANGELOG.md",
		".agentspack/docs/cloudflare-agent-stack/_directive.md",
	}
	for _, rel := range expectedDocs {
		path := filepath.Join(tmpDir, rel)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected doc pack file %s does not exist", rel)
		}
	}

	// Cursor should get an alwaysApply rule carrying the directive.
	cursorRulePath := filepath.Join(tmpDir, ".cursor", "rules", "cloudflare-agent-stack", "RULE.md")
	cursorRule, err := os.ReadFile(cursorRulePath)
	if err != nil {
		t.Fatalf("Expected Cursor doc pack rule at %s: %v", cursorRulePath, err)
	}
	if !strings.Contains(string(cursorRule), "alwaysApply: true") {
		t.Errorf("Expected Cursor doc pack rule to be alwaysApply")
	}
	if !strings.Contains(string(cursorRule), "Decision matrix") {
		t.Errorf("Expected Cursor doc pack rule to embed the directive body")
	}

	// Claude Code: directive should be appended to CLAUDE.md.
	claudeMD, err := os.ReadFile(filepath.Join(tmpDir, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("Expected CLAUDE.md: %v", err)
	}
	if !strings.Contains(string(claudeMD), "agentspack:doc-pack:cloudflare-agent-stack") {
		t.Errorf("Expected CLAUDE.md to contain doc pack marker")
	}
	if !strings.Contains(string(claudeMD), "Cloudflare Agent Stack") {
		t.Errorf("Expected CLAUDE.md to include the Cloudflare directive heading")
	}

	// Codex: directive should be appended to AGENTS.md.
	agentsMD, err := os.ReadFile(filepath.Join(tmpDir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("Expected AGENTS.md: %v", err)
	}
	if !strings.Contains(string(agentsMD), "agentspack:doc-pack:cloudflare-agent-stack") {
		t.Errorf("Expected AGENTS.md to contain doc pack marker")
	}

	// Regular tech stacks should still be generated alongside the doc pack.
	reactSkillPath := filepath.Join(tmpDir, ".agents", "skills", "react-guidelines", "SKILL.md")
	if _, err := os.Stat(reactSkillPath); os.IsNotExist(err) {
		t.Errorf("Expected react skill alongside doc pack at %s", reactSkillPath)
	}

	// The Cursor rule directory must not duplicate the doc pack as if it
	// were a normal stack (only the doc-pack rule path should exist).
	unexpected := filepath.Join(tmpDir, ".cursor", "rules", "cloudflare-agent-stack-cloudflare-agent-stack")
	if _, err := os.Stat(unexpected); err == nil {
		t.Errorf("Doc pack incorrectly routed through the regular tech-stack rule path: %s", unexpected)
	}
}

func TestGeneratorAddModeInstallsDocPack(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agentspack-add-docpack-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Simulate an existing repo that was generated previously: CLAUDE.md,
	// AGENTS.md, and a placeholder Cursor rule already exist. The doc pack
	// install should upsert into the base files and (with SkipExisting)
	// leave the user's Cursor rule alone.
	originalClaude := "# CLAUDE.md\n\npreexisting project context\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "CLAUDE.md"), []byte(originalClaude), 0644); err != nil {
		t.Fatalf("Failed to seed CLAUDE.md: %v", err)
	}
	originalAgents := "# AGENTS.md\n\npreexisting project context\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte(originalAgents), 0644); err != nil {
		t.Fatalf("Failed to seed AGENTS.md: %v", err)
	}

	customRulePath := filepath.Join(tmpDir, ".cursor", "rules", "cloudflare-agent-stack", "RULE.md")
	if err := os.MkdirAll(filepath.Dir(customRulePath), 0755); err != nil {
		t.Fatalf("Failed to create custom rule directory: %v", err)
	}
	customRuleContent := "# user-edited rule — do not overwrite\n"
	if err := os.WriteFile(customRulePath, []byte(customRuleContent), 0644); err != nil {
		t.Fatalf("Failed to seed custom rule: %v", err)
	}

	config := &wizard.Config{
		Mode:             wizard.GenerationModeAdd,
		Providers:        []string{"cursor", "claude-code", "codex"},
		OutputDir:        tmpDir,
		ConflictPolicy:   wizard.ConflictPolicySkipExisting,
		SelectedDocPacks: []string{"cloudflare-agent-stack"},
	}

	gen := New(config, "/nonexistent/system")
	if err := gen.Run(); err != nil {
		t.Fatalf("Add-mode doc pack install failed: %v", err)
	}

	// Docs folder should contain the copied reference library + directive.
	docsSample := []string{
		".agentspack/docs/cloudflare-agent-stack/00_START_HERE.md",
		".agentspack/docs/cloudflare-agent-stack/_directive.md",
	}
	for _, rel := range docsSample {
		if _, err := os.Stat(filepath.Join(tmpDir, rel)); os.IsNotExist(err) {
			t.Errorf("Expected doc pack file %s after add-mode install", rel)
		}
	}

	// CLAUDE.md should keep original content AND contain the upserted marker block.
	claude, err := os.ReadFile(filepath.Join(tmpDir, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("Failed to read CLAUDE.md: %v", err)
	}
	if !strings.Contains(string(claude), "preexisting project context") {
		t.Errorf("CLAUDE.md lost its original content")
	}
	if !strings.Contains(string(claude), "<!-- agentspack:doc-pack:cloudflare-agent-stack -->") {
		t.Errorf("CLAUDE.md missing doc-pack start marker")
	}
	if !strings.Contains(string(claude), "<!-- /agentspack:doc-pack:cloudflare-agent-stack -->") {
		t.Errorf("CLAUDE.md missing doc-pack end marker")
	}

	// AGENTS.md: same expectations.
	agents, err := os.ReadFile(filepath.Join(tmpDir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("Failed to read AGENTS.md: %v", err)
	}
	if !strings.Contains(string(agents), "preexisting project context") {
		t.Errorf("AGENTS.md lost its original content")
	}
	if !strings.Contains(string(agents), "<!-- /agentspack:doc-pack:cloudflare-agent-stack -->") {
		t.Errorf("AGENTS.md missing doc-pack end marker")
	}

	// The user's custom Cursor rule must remain intact under SkipExisting.
	preserved, err := os.ReadFile(customRulePath)
	if err != nil {
		t.Fatalf("Failed to read custom rule: %v", err)
	}
	if string(preserved) != customRuleContent {
		t.Errorf("Expected user-edited cursor rule to be preserved; got %q", string(preserved))
	}

	// Running the install a second time must not duplicate the marker block.
	if err := gen.Run(); err != nil {
		t.Fatalf("Second add-mode run failed: %v", err)
	}
	claude2, err := os.ReadFile(filepath.Join(tmpDir, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("Failed to re-read CLAUDE.md: %v", err)
	}
	startCount := strings.Count(string(claude2), "<!-- agentspack:doc-pack:cloudflare-agent-stack -->")
	if startCount != 1 {
		t.Errorf("Expected exactly one doc-pack start marker after rerun, got %d", startCount)
	}
	endCount := strings.Count(string(claude2), "<!-- /agentspack:doc-pack:cloudflare-agent-stack -->")
	if endCount != 1 {
		t.Errorf("Expected exactly one doc-pack end marker after rerun, got %d", endCount)
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
