package content

import (
	"testing"
)

func TestEmbeddedFS(t *testing.T) {
	efs := NewEmbeddedFS()

	// Test reading base.md
	content, err := efs.ReadFile("system/base/base.md")
	if err != nil {
		t.Fatalf("Failed to read system/base/base.md: %v", err)
	}
	if len(content) == 0 {
		t.Error("base.md is empty")
	}
	t.Logf("base.md content length: %d bytes", len(content))

	// Test Glob
	files, err := efs.Glob("system/base/*.md")
	if err != nil {
		t.Fatalf("Failed to glob system/base/*.md: %v", err)
	}
	t.Logf("Found %d files in system/base/: %v", len(files), files)
	if len(files) == 0 {
		t.Error("Expected to find files in system/base/")
	}

	// Test Stat
	info, err := efs.Stat("system/base")
	if err != nil {
		t.Fatalf("Failed to stat system/base: %v", err)
	}
	if !info.IsDir() {
		t.Error("system/base should be a directory")
	}
}
