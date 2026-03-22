package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agentspack/agentspack/internal/content"
)

func TestListBaseSkillsRequiresDescription(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agentspack-catalog-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	systemDir := filepath.Join(tmpDir, "system", "base-skills")
	if err := os.MkdirAll(systemDir, 0755); err != nil {
		t.Fatalf("Failed to create system dir: %v", err)
	}

	contentStr := `---
name: bad-skill
---

# Missing Description
`
	if err := os.WriteFile(filepath.Join(systemDir, "bad-skill.md"), []byte(contentStr), 0644); err != nil {
		t.Fatalf("Failed to write base skill: %v", err)
	}

	_, err = ListBaseSkills(content.NewLocalFS(tmpDir))
	if err == nil {
		t.Fatal("Expected an error for a base skill without description")
	}
	if !strings.Contains(err.Error(), "requires a non-empty description") {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestListBaseSkillsRejectsInvalidFrontmatter(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agentspack-catalog-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	systemDir := filepath.Join(tmpDir, "system", "base-skills")
	if err := os.MkdirAll(systemDir, 0755); err != nil {
		t.Fatalf("Failed to create system dir: %v", err)
	}

	contentStr := `---
name: bad-skill
description: [unterminated
---

# Broken
`
	if err := os.WriteFile(filepath.Join(systemDir, "bad-skill.md"), []byte(contentStr), 0644); err != nil {
		t.Fatalf("Failed to write base skill: %v", err)
	}

	_, err = ListBaseSkills(content.NewLocalFS(tmpDir))
	if err == nil {
		t.Fatal("Expected an error for invalid base skill frontmatter")
	}
	if !strings.Contains(err.Error(), "failed to parse base skill frontmatter") {
		t.Fatalf("Unexpected error: %v", err)
	}
}
