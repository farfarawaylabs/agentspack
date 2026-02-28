package providers

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// parseAgentFrontmatter extracts name, description, and body from markdown.
// It supports YAML frontmatter with multiline values and ignores unknown fields.
func parseAgentFrontmatter(content string) (name, description, body string) {
	frontmatter, parsedBody, ok := splitFrontmatter(content)
	if !ok {
		return "", "", content
	}

	var meta struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal([]byte(frontmatter), &meta); err != nil {
		return "", "", parsedBody
	}

	return strings.TrimSpace(meta.Name), strings.TrimSpace(meta.Description), parsedBody
}

func splitFrontmatter(content string) (frontmatter, body string, ok bool) {
	lines := strings.Split(content, "\n")
	if len(lines) < 3 || (lines[0] != "---" && lines[0] != "---\r") {
		return "", "", false
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		// Frontmatter delimiter must start at column 0 to avoid matching
		// literal '---' lines inside indented YAML block scalars.
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

func agentNameFromSourcePath(sourcePath string) string {
	pathWithoutExt := strings.TrimSuffix(sourcePath, ".md")
	pathWithoutExt = strings.ReplaceAll(pathWithoutExt, "\\", "/")
	pathWithoutPrefix := strings.TrimPrefix(pathWithoutExt, "system/agents/")
	name := strings.ReplaceAll(pathWithoutPrefix, "/", "-")
	name = strings.ReplaceAll(name, "_", "-")
	return strings.Trim(name, "-")
}
