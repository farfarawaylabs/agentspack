package providers

import (
	"github.com/agentspack/agentspack/internal/content"
	"github.com/agentspack/agentspack/internal/docpacks"
)

// DocPack is re-exported so existing provider code that references the
// providers package keeps compiling. New code should prefer the docpacks
// package directly.
type DocPack = docpacks.DocPack

// IsDocPack reports whether the given tech-stack name corresponds to a
// registered doc pack.
func IsDocPack(name string) bool { return docpacks.Is(name) }

// DocPackDisplayName returns the human-readable label for a pack.
func DocPackDisplayName(name string) string { return docpacks.DisplayName(name) }

// LoadDocPack loads a registered doc pack from the embedded filesystem.
func LoadDocPack(fs content.FileSystem, name string) (*DocPack, error) {
	return docpacks.Load(fs, name)
}

// DocPackOutputRelPath returns the canonical output folder for a pack's docs.
func DocPackOutputRelPath(name string) string { return docpacks.OutputRelPath(name) }

// CopyDocPackDocs copies a pack's reference docs and directive into the
// output directory.
func CopyDocPackDocs(fs content.FileSystem, pack *DocPack, outputDir string) error {
	return docpacks.Copy(fs, pack, outputDir)
}

// AppendDirectiveToBaseFile upserts the doc pack's directive in an existing
// base file. The name is retained for backwards compatibility with callers;
// the behavior is idempotent via marker-based replacement.
func AppendDirectiveToBaseFile(baseFilePath string, pack *DocPack) error {
	return docpacks.UpsertIntoBaseFile(baseFilePath, pack)
}

// DocPackBaseFileSection returns the marker-wrapped directive section ready
// to be concatenated into CLAUDE.md / AGENTS.md.
func DocPackBaseFileSection(pack *DocPack) string {
	return docpacks.BaseFileSection(pack)
}

// ListDocPacks returns metadata for every registered doc pack.
func ListDocPacks(fs content.FileSystem) ([]*DocPack, error) {
	return docpacks.List(fs)
}

// DocPackShortDescription is re-exported for external callers.
func DocPackShortDescription(pack *DocPack) string {
	return docpacks.ShortDescription(pack)
}
