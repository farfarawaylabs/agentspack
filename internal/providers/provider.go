package providers

import (
	"github.com/agentspack/agentspack/internal/content"
	"github.com/agentspack/agentspack/internal/wizard"
)

// Provider defines the interface for converting templates to provider-specific formats
type Provider interface {
	// Name returns the provider identifier (e.g., "cursor", "claude-code")
	Name() string

	// Generate creates provider-specific output files based on the configuration
	// fs is the filesystem to read templates from (embedded or local)
	// outputDir is the base output directory
	Generate(config *wizard.Config, fs content.FileSystem, outputDir string) error
}

// Registry holds all available providers
var Registry = make(map[string]Provider)

// Register adds a provider to the registry
func Register(p Provider) {
	Registry[p.Name()] = p
}

// Get retrieves a provider by name
func Get(name string) (Provider, bool) {
	p, ok := Registry[name]
	return p, ok
}
