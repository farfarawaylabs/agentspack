package providers

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/agentspack/agentspack/internal/content"
	"gopkg.in/yaml.v3"
)

type baseSkillFrontmatter struct {
	Name                   string         `yaml:"name"`
	Description            string         `yaml:"description"`
	Metadata               map[string]any `yaml:"metadata,omitempty"`
	DisableModelInvocation *bool          `yaml:"disable-model-invocation,omitempty"`
	UserInvocable          *bool          `yaml:"user-invocable,omitempty"`
}

func listBaseSkillTemplates(fs content.FileSystem) ([]string, error) {
	if _, err := fs.Stat("system/base-skills"); err != nil {
		// Base skills are optional.
		return nil, nil
	}

	files, err := fs.Glob("system/base-skills/*.md")
	if err != nil {
		return nil, err
	}

	subFiles, err := fs.Glob("system/base-skills/**/*.md")
	if err == nil {
		files = mergeUniquePaths(files, subFiles)
	}

	return files, nil
}

func buildBaseSkillFromTemplate(fs content.FileSystem, sourcePath string, invocationSettings SkillInvocationSettings, includeUserInvocable bool) (string, string, error) {
	fileContent, err := fs.ReadFile(sourcePath)
	if err != nil {
		return "", "", err
	}

	contentStr := string(fileContent)
	frontmatter, body, hasFrontmatter := splitFrontmatter(contentStr)

	var meta baseSkillFrontmatter
	skillBody := strings.TrimSpace(contentStr)

	if hasFrontmatter {
		if err := yaml.Unmarshal([]byte(frontmatter), &meta); err != nil {
			return "", "", fmt.Errorf("failed to parse base skill frontmatter '%s': %w", sourcePath, err)
		}
		skillBody = strings.TrimSpace(body)
	}

	skillName := strings.TrimSpace(meta.Name)
	if skillName == "" {
		skillName = baseSkillNameFromSourcePath(sourcePath)
	}
	if skillName == "" {
		return "", "", fmt.Errorf("failed to derive base skill name from path '%s'", sourcePath)
	}

	description := strings.TrimSpace(meta.Description)
	if description == "" {
		return "", "", fmt.Errorf("base skill template '%s' requires a non-empty description", sourcePath)
	}

	disableModelInvocation := meta.DisableModelInvocation
	if invocationSettings.DisableModelInvocation != nil {
		disableModelInvocation = invocationSettings.DisableModelInvocation
	}

	var userInvocable *bool
	if includeUserInvocable {
		userInvocable = meta.UserInvocable
		if invocationSettings.UserInvocable != nil {
			userInvocable = invocationSettings.UserInvocable
		}
	}

	skillMarkdown, err := buildSkillMarkdown(SkillDocOptions{
		Name:                   skillName,
		Description:            description,
		Body:                   skillBody,
		Metadata:               meta.Metadata,
		DisableModelInvocation: disableModelInvocation,
		UserInvocable:          userInvocable,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to build base skill markdown for '%s': %w", sourcePath, err)
	}

	return skillName, skillMarkdown, nil
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
