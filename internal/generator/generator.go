package generator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/agentspack/agentspack/internal/content"
	"github.com/agentspack/agentspack/internal/providers"
	"github.com/agentspack/agentspack/internal/wizard"
)

// Generator orchestrates the template conversion process
type Generator struct {
	config *wizard.Config
	fs     content.FileSystem
	fsType string
}

// New creates a new Generator instance
// localSystemDir is the path to check for a local system directory
func New(config *wizard.Config, localSystemDir string) *Generator {
	fs, fsType := content.GetFileSystem(localSystemDir)
	return &Generator{
		config: config,
		fs:     fs,
		fsType: fsType,
	}
}

// Run executes the generation process for all selected providers
func (g *Generator) Run() error {
	// Create output directory
	outputDir := g.config.OutputDir
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Get absolute path for output
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return fmt.Errorf("failed to resolve output path: %w", err)
	}

	fmt.Printf("\nUsing %s templates\n", g.fsType)
	fmt.Printf("Generating files to: %s\n\n", absOutputDir)

	// Process each selected provider
	for _, providerName := range g.config.Providers {
		provider, ok := providers.Get(providerName)
		if !ok {
			fmt.Printf("Warning: provider '%s' is not yet implemented, skipping\n", providerName)
			continue
		}

		fmt.Printf("Generating for %s...\n", providerName)
		if err := provider.Generate(g.config, g.fs, outputDir); err != nil {
			return fmt.Errorf("failed to generate for %s: %w", providerName, err)
		}
		fmt.Println()
	}

	fmt.Println("Generation complete!")
	return nil
}
