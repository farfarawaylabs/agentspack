package providers

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/agentspack/agentspack/internal/content"
)

type SystemCommandTemplate struct {
	Name        string
	Description string
	Body        string
	SourcePath  string
}

func listSystemCommandTemplates(fs content.FileSystem) ([]SystemCommandTemplate, error) {
	if _, err := fs.Stat("system/commands"); err != nil {
		// Commands are optional.
		return nil, nil
	}

	files, err := fs.Glob("system/commands/*.md")
	if err != nil {
		return nil, err
	}

	subFiles, err := fs.Glob("system/commands/**/*.md")
	if err == nil {
		files = mergeUniquePaths(files, subFiles)
	}

	templates := make([]SystemCommandTemplate, 0, len(files))
	for _, file := range files {
		fileContent, err := fs.ReadFile(file)
		if err != nil {
			return nil, err
		}

		contentStr := string(fileContent)
		name, description, body := parseAgentFrontmatter(contentStr)
		if strings.TrimSpace(name) == "" {
			name = commandNameFromSourcePath(file)
		}
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, fmt.Errorf("failed to derive command name from %s", file)
		}

		if strings.TrimSpace(body) == "" {
			body = contentStr
		}
		body = strings.TrimSpace(body)
		if body == "" {
			return nil, fmt.Errorf("command '%s' has empty content", file)
		}

		if strings.TrimSpace(description) == "" {
			description = extractDescription(body, "Command", name)
		}

		templates = append(templates, SystemCommandTemplate{
			Name:        name,
			Description: strings.TrimSpace(description),
			Body:        body,
			SourcePath:  file,
		})
	}

	return templates, nil
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
