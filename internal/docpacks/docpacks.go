// Package docpacks holds shared metadata and install helpers for
// "doc packs" — curated reference libraries that agentspack copies verbatim
// into generated projects and whose directive is injected into per-provider
// base files.
//
// This package intentionally lives above both catalog and providers so both
// can depend on it without creating an import cycle. Providers call the
// install helpers; catalog calls the listing helpers for the wizard.
package docpacks

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/agentspack/agentspack/internal/content"
)

// DocPack is the loaded metadata for one pack.
//
// Doc packs live under system/doc-packs/<name>/ and must contain:
//   - _directive.md : short prompt injected into CLAUDE.md / AGENTS.md / Cursor rule
//   - docs/*.md      : the reference files copied into the target project
type DocPack struct {
	Name        string
	DisplayName string
	SourceDir   string
	Directive   string
}

// registry lists every known doc pack. Add a new entry here plus a
// system/doc-packs/<name>/ folder to make a pack available.
var registry = map[string]struct {
	DisplayName string
}{
	"cloudflare-agent-stack": {
		DisplayName: "Cloudflare Agent Stack",
	},
}

// Is returns true when the given name matches a registered doc pack. Used
// by providers to branch off the default rule/skill flow.
func Is(name string) bool {
	_, ok := registry[name]
	return ok
}

// DisplayName returns the human-readable label for a pack, or the raw name
// if the pack is not in the registry.
func DisplayName(name string) string {
	if entry, ok := registry[name]; ok {
		return entry.DisplayName
	}
	return name
}

// Load reads a doc pack's directive from the content filesystem and returns
// a populated DocPack. The docs themselves are streamed lazily by Copy; we
// only eagerly load the directive since it's the part that is injected
// into base files.
func Load(fs content.FileSystem, name string) (*DocPack, error) {
	entry, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown doc pack %q", name)
	}

	sourceDir := filepath.ToSlash(filepath.Join("system", "doc-packs", name))

	directivePath := sourceDir + "/_directive.md"
	directiveBytes, err := fs.ReadFile(directivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read doc pack directive %q: %w", directivePath, err)
	}

	return &DocPack{
		Name:        name,
		DisplayName: entry.DisplayName,
		SourceDir:   sourceDir,
		Directive:   strings.TrimSpace(string(directiveBytes)),
	}, nil
}

// List returns metadata for every registered doc pack. Each entry's
// directive is loaded so callers can compute a description without
// reloading the pack twice.
func List(fs content.FileSystem) ([]*DocPack, error) {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)

	packs := make([]*DocPack, 0, len(names))
	for _, name := range names {
		pack, err := Load(fs, name)
		if err != nil {
			return nil, err
		}
		packs = append(packs, pack)
	}
	return packs, nil
}

// OutputRelPath is the folder (relative to the output root) where a doc
// pack's reference files are copied. Kept stable so agents and humans can
// reference the same path regardless of provider.
func OutputRelPath(name string) string {
	return filepath.Join(".agentspack", "docs", name)
}

// Copy copies every markdown file under <pack.SourceDir>/docs/ into
// <outputDir>/<OutputRelPath(pack.Name)>/ preserving filenames, and also
// writes the directive alongside them as `_directive.md` so it is
// discoverable even when the base file is not generated. Existing files are
// overwritten in full-generate mode; callers that need selective-install
// behavior should gate this call behind their own check.
func Copy(fs content.FileSystem, pack *DocPack, outputDir string) error {
	if pack == nil {
		return fmt.Errorf("doc pack is nil")
	}

	pattern := pack.SourceDir + "/docs/*.md"
	files, err := fs.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob %s: %w", pattern, err)
	}
	if len(files) == 0 {
		return fmt.Errorf("doc pack %q has no documents under %s/docs/", pack.Name, pack.SourceDir)
	}

	sort.Strings(files)

	targetDir := filepath.Join(outputDir, OutputRelPath(pack.Name))
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create doc pack directory %s: %w", targetDir, err)
	}

	for _, sourcePath := range files {
		data, err := fs.ReadFile(sourcePath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", sourcePath, err)
		}

		baseName := filepath.Base(sourcePath)
		outputPath := filepath.Join(targetDir, baseName)
		if err := os.WriteFile(outputPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", outputPath, err)
		}

		fmt.Printf("  Created: %s\n", outputPath)
	}

	directiveOutputPath := filepath.Join(targetDir, "_directive.md")
	if err := os.WriteFile(directiveOutputPath, []byte(pack.Directive+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", directiveOutputPath, err)
	}
	fmt.Printf("  Created: %s\n", directiveOutputPath)

	return nil
}

// UpsertIntoBaseFile upserts the doc pack's directive section in an
// existing base file (CLAUDE.md or AGENTS.md) already created by the
// provider. The directive is wrapped between markers so re-running the
// generator (or invoking `agentspack add` multiple times) replaces the
// existing block in place rather than appending duplicates.
//
// If the base file does not exist (e.g. GenerateBase was false), this is a
// no-op; the docs folder still contains a standalone copy of the directive
// so the user doesn't silently lose it.
func UpsertIntoBaseFile(baseFilePath string, pack *DocPack) error {
	if pack == nil {
		return fmt.Errorf("doc pack is nil")
	}

	info, err := os.Stat(baseFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to stat %s: %w", baseFilePath, err)
	}
	if info.IsDir() {
		return fmt.Errorf("base file path %s is a directory", baseFilePath)
	}

	section := BaseFileSection(pack)
	if section == "" {
		return nil
	}

	existing, err := os.ReadFile(baseFilePath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", baseFilePath, err)
	}

	updated, action := upsertSection(string(existing), pack.Name, section)
	if err := os.WriteFile(baseFilePath, []byte(updated), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", baseFilePath, err)
	}

	fmt.Printf("  Updated: %s (%s %s directive)\n", baseFilePath, action, pack.Name)
	return nil
}

func upsertSection(existing, packName, section string) (string, string) {
	startMarker, endMarker := markers(packName)

	startIdx := strings.Index(existing, startMarker)
	if startIdx == -1 {
		return existing + section, "appended"
	}

	endIdx := strings.Index(existing[startIdx:], endMarker)
	if endIdx == -1 {
		prefix := strings.TrimRight(existing[:startIdx], " \n")
		return prefix + section, "replaced"
	}
	endIdx += startIdx + len(endMarker)

	prefix := strings.TrimRight(existing[:startIdx], " \n")
	suffix := existing[endIdx:]
	return prefix + section + suffix, "replaced"
}

func markers(packName string) (string, string) {
	return fmt.Sprintf("<!-- agentspack:doc-pack:%s -->", packName),
		fmt.Sprintf("<!-- /agentspack:doc-pack:%s -->", packName)
}

// BaseFileSection returns the directive wrapped in a clearly marked section
// suitable for concatenating into CLAUDE.md / AGENTS.md. The section is
// bracketed by HTML comment markers so it can be safely upserted on
// repeated runs via UpsertIntoBaseFile.
func BaseFileSection(pack *DocPack) string {
	if pack == nil || strings.TrimSpace(pack.Directive) == "" {
		return ""
	}

	startMarker, endMarker := markers(pack.Name)

	var sb strings.Builder
	sb.WriteString("\n\n---\n\n")
	sb.WriteString(startMarker)
	sb.WriteString("\n\n")
	sb.WriteString(pack.Directive)
	sb.WriteString("\n\n")
	sb.WriteString(endMarker)
	sb.WriteString("\n")
	return sb.String()
}

// ShortDescription returns a single-line description suitable for wizard
// option labels. It prefers the first paragraph of the directive, falling
// back to a generic string.
func ShortDescription(pack *DocPack) string {
	if pack == nil {
		return ""
	}
	directive := strings.TrimSpace(pack.Directive)
	if directive == "" {
		return "Reference documentation pack"
	}

	for _, raw := range strings.Split(directive, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "<!--") {
			continue
		}
		if len(line) > 160 {
			return line[:157] + "..."
		}
		return line
	}
	return "Reference documentation pack"
}
