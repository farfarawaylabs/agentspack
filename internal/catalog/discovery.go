package catalog

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/agentspack/agentspack/internal/content"
	"github.com/agentspack/agentspack/internal/docpacks"
	"github.com/agentspack/agentspack/internal/templates"
	"gopkg.in/yaml.v3"
)

type BaseSkill struct {
	Name        string
	Description string
	SourcePath  string
}

type SystemCommand struct {
	Name        string
	Description string
	SourcePath  string
}

type Workflow struct {
	Name        string
	Description string
	SourceDir   string
}

type DocPack struct {
	Name        string
	DisplayName string
	Description string
}

// ListDocPacks enumerates every registered doc pack (system/doc-packs/<name>).
// Descriptions are derived from the pack's directive so wizard options remain
// informative without loading the full set of reference docs.
func ListDocPacks(fs content.FileSystem) ([]DocPack, error) {
	packs, err := docpacks.List(fs)
	if err != nil {
		return nil, err
	}

	result := make([]DocPack, 0, len(packs))
	for _, pack := range packs {
		result = append(result, DocPack{
			Name:        pack.Name,
			DisplayName: pack.DisplayName,
			Description: docpacks.ShortDescription(pack),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result, nil
}

func ListBaseSkills(fs content.FileSystem) ([]BaseSkill, error) {
	if _, err := fs.Stat("system/base-skills"); err != nil {
		return nil, nil
	}

	files, err := globTemplates(fs, "system/base-skills/*.md", "system/base-skills/**/*.md")
	if err != nil {
		return nil, err
	}

	skills := make([]BaseSkill, 0, len(files))
	for _, file := range files {
		fileContent, err := fs.ReadFile(file)
		if err != nil {
			return nil, err
		}

		name, description, _, err := parseFrontmatterStrict(string(fileContent))
		if err != nil {
			return nil, fmt.Errorf("failed to parse base skill frontmatter '%s': %w", file, err)
		}
		if name == "" {
			name = baseSkillNameFromSourcePath(file)
		}
		if name == "" {
			return nil, fmt.Errorf("failed to derive base skill name from %s", file)
		}
		if description == "" {
			return nil, fmt.Errorf("base skill template '%s' requires a non-empty description", file)
		}

		skills = append(skills, BaseSkill{
			Name:        name,
			Description: description,
			SourcePath:  file,
		})
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})

	return skills, nil
}

func ListSystemCommands(fs content.FileSystem) ([]SystemCommand, error) {
	if _, err := fs.Stat("system/commands"); err != nil {
		return nil, nil
	}

	files, err := globTemplates(fs, "system/commands/*.md", "system/commands/**/*.md")
	if err != nil {
		return nil, err
	}

	commands := make([]SystemCommand, 0, len(files))
	for _, file := range files {
		fileContent, err := fs.ReadFile(file)
		if err != nil {
			return nil, err
		}

		name, description, body := parseFrontmatter(string(fileContent))
		if name == "" {
			name = commandNameFromSourcePath(file)
		}
		if name == "" {
			return nil, fmt.Errorf("failed to derive command name from %s", file)
		}
		if description == "" {
			description = fallbackDescription(body, "Command", name)
		}

		commands = append(commands, SystemCommand{
			Name:        name,
			Description: description,
			SourcePath:  file,
		})
	}

	sort.Slice(commands, func(i, j int) bool {
		return commands[i].Name < commands[j].Name
	})

	return commands, nil
}

func ListWorkflows(fs content.FileSystem) ([]Workflow, error) {
	if _, err := fs.Stat("system/workflows"); err != nil {
		return nil, nil
	}

	entries, err := fs.ReadDir("system/workflows")
	if err != nil {
		return nil, err
	}

	workflows := make([]Workflow, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		workflowName := entry.Name()
		files, err := fs.Glob(fmt.Sprintf("system/workflows/%s/*.md", workflowName))
		if err != nil {
			return nil, err
		}
		sort.Strings(files)

		description := fmt.Sprintf("%s workflow", templates.NormalizeWorkflowName(workflowName))
		if len(files) > 0 {
			firstStep, err := fs.ReadFile(files[0])
			if err != nil {
				return nil, err
			}
			if stepDesc := templates.ExtractStepDescription(string(firstStep)); stepDesc != "" {
				description = stepDesc
			}
		}

		workflows = append(workflows, Workflow{
			Name:        workflowName,
			Description: description,
			SourceDir:   filepath.ToSlash(filepath.Join("system/workflows", workflowName)),
		})
	}

	sort.Slice(workflows, func(i, j int) bool {
		return workflows[i].Name < workflows[j].Name
	})

	return workflows, nil
}

func globTemplates(fs content.FileSystem, patterns ...string) ([]string, error) {
	var files []string
	for _, pattern := range patterns {
		matches, err := fs.Glob(pattern)
		if err != nil {
			return nil, err
		}
		files = mergeUniquePaths(files, matches)
	}
	sort.Strings(files)
	return files, nil
}

func mergeUniquePaths(base, extra []string) []string {
	seen := make(map[string]struct{}, len(base)+len(extra))
	merged := make([]string, 0, len(base)+len(extra))

	for _, item := range base {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		merged = append(merged, item)
	}
	for _, item := range extra {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		merged = append(merged, item)
	}

	return merged
}

func parseFrontmatter(content string) (name, description, body string) {
	name, description, body, _ = parseFrontmatterStrict(content)
	return name, description, body
}

func parseFrontmatterStrict(content string) (name, description, body string, err error) {
	frontmatter, parsedBody, ok := splitFrontmatter(content)
	if !ok {
		return "", "", strings.TrimSpace(content), nil
	}

	var meta struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal([]byte(frontmatter), &meta); err != nil {
		return "", "", strings.TrimSpace(parsedBody), err
	}

	return strings.TrimSpace(meta.Name), strings.TrimSpace(meta.Description), strings.TrimSpace(parsedBody), nil
}

func splitFrontmatter(content string) (frontmatter, body string, ok bool) {
	lines := strings.Split(content, "\n")
	if len(lines) < 3 || (lines[0] != "---" && lines[0] != "---\r") {
		return "", "", false
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" || lines[i] == "---\r" {
			end = i
			break
		}
	}
	if end == -1 {
		return "", "", false
	}

	frontmatter = strings.Join(lines[1:end], "\n")
	body = strings.TrimSpace(strings.Join(lines[end+1:], "\n"))
	return frontmatter, body, true
}

func fallbackDescription(body, prefix, fallback string) string {
	if extracted := extractHeadingOrParagraph(body); extracted != "" {
		return fmt.Sprintf("%s: %s", prefix, extracted)
	}
	return fmt.Sprintf("%s: %s", prefix, fallback)
}

func extractHeadingOrParagraph(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))
		}
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
		}
		return trimmed
	}
	return ""
}

func baseSkillNameFromSourcePath(sourcePath string) string {
	normalizedPath := strings.ReplaceAll(sourcePath, "\\", "/")
	pathWithoutExt := strings.TrimSuffix(normalizedPath, filepath.Ext(normalizedPath))
	pathWithoutPrefix := strings.TrimPrefix(pathWithoutExt, "system/base-skills/")
	name := strings.ReplaceAll(pathWithoutPrefix, "/", "-")
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ToLower(name)
	return strings.Trim(name, "-")
}

func commandNameFromSourcePath(sourcePath string) string {
	normalizedPath := strings.ReplaceAll(sourcePath, "\\", "/")
	pathWithoutExt := strings.TrimSuffix(normalizedPath, filepath.Ext(normalizedPath))
	pathWithoutPrefix := strings.TrimPrefix(pathWithoutExt, "system/commands/")
	name := strings.ReplaceAll(pathWithoutPrefix, "/", "-")
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ToLower(name)
	return strings.Trim(name, "-")
}
