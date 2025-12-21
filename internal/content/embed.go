package content

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed all:system
var embeddedSystem embed.FS

// FileSystem provides an abstraction over both embedded and local filesystems
type FileSystem interface {
	ReadFile(path string) ([]byte, error)
	ReadDir(path string) ([]fs.DirEntry, error)
	Glob(pattern string) ([]string, error)
	Stat(path string) (fs.FileInfo, error)
}

// EmbeddedFS wraps the embedded filesystem
type EmbeddedFS struct {
	fs embed.FS
}

// NewEmbeddedFS creates a new embedded filesystem
func NewEmbeddedFS() *EmbeddedFS {
	return &EmbeddedFS{fs: embeddedSystem}
}

func (e *EmbeddedFS) ReadFile(path string) ([]byte, error) {
	return e.fs.ReadFile(path)
}

func (e *EmbeddedFS) ReadDir(path string) ([]fs.DirEntry, error) {
	return e.fs.ReadDir(path)
}

func (e *EmbeddedFS) Glob(pattern string) ([]string, error) {
	return fs.Glob(e.fs, pattern)
}

func (e *EmbeddedFS) Stat(path string) (fs.FileInfo, error) {
	return fs.Stat(e.fs, path)
}

// LocalFS wraps the local filesystem with a base directory
type LocalFS struct {
	baseDir string
}

// NewLocalFS creates a new local filesystem rooted at baseDir
func NewLocalFS(baseDir string) *LocalFS {
	return &LocalFS{baseDir: baseDir}
}

func (l *LocalFS) ReadFile(path string) ([]byte, error) {
	// For local FS, path is relative to baseDir but doesn't include "system" prefix
	// We need to handle the path correctly
	fullPath := filepath.Join(l.baseDir, path)
	return os.ReadFile(fullPath)
}

func (l *LocalFS) ReadDir(path string) ([]fs.DirEntry, error) {
	fullPath := filepath.Join(l.baseDir, path)
	return os.ReadDir(fullPath)
}

func (l *LocalFS) Glob(pattern string) ([]string, error) {
	fullPattern := filepath.Join(l.baseDir, pattern)
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return nil, err
	}

	// Convert absolute paths back to relative paths (matching embedded FS behavior)
	result := make([]string, len(matches))
	for i, match := range matches {
		rel, err := filepath.Rel(l.baseDir, match)
		if err != nil {
			result[i] = match
		} else {
			result[i] = rel
		}
	}
	return result, nil
}

func (l *LocalFS) Stat(path string) (fs.FileInfo, error) {
	fullPath := filepath.Join(l.baseDir, path)
	return os.Stat(fullPath)
}

// GetFileSystem returns the appropriate filesystem based on whether a local
// system directory exists. If localSystemDir exists and is valid, use local; otherwise use embedded.
func GetFileSystem(localSystemDir string) (FileSystem, string) {
	// Check if local system directory exists and is non-empty
	if localSystemDir != "" {
		if info, err := os.Stat(localSystemDir); err == nil && info.IsDir() {
			// Use local filesystem - the baseDir is the parent of system/
			baseDir := filepath.Dir(localSystemDir)
			return NewLocalFS(baseDir), "local"
		}
	}

	// Fall back to embedded filesystem
	// Debug: verify embedded FS works
	efs := NewEmbeddedFS()
	if _, err := efs.Stat("system/base"); err != nil {
		panic("EMBEDDED FS ERROR: system/base not found: " + err.Error())
	}
	return efs, "embedded"
}
