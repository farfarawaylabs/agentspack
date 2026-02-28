package providers

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type SkillDocOptions struct {
	Name                   string
	Description            string
	Heading                string
	Body                   string
	Metadata               map[string]any
	DisableModelInvocation *bool
	UserInvocable          *bool
}

func buildSkillMarkdown(opts SkillDocOptions) (string, error) {
	if strings.TrimSpace(opts.Name) == "" {
		return "", fmt.Errorf("skill name cannot be empty")
	}
	if strings.TrimSpace(opts.Description) == "" {
		return "", fmt.Errorf("skill description cannot be empty")
	}

	frontmatter := struct {
		Name                   string         `yaml:"name"`
		Description            string         `yaml:"description"`
		Metadata               map[string]any `yaml:"metadata,omitempty"`
		DisableModelInvocation *bool          `yaml:"disable-model-invocation,omitempty"`
		UserInvocable          *bool          `yaml:"user-invocable,omitempty"`
	}{
		Name:                   strings.TrimSpace(opts.Name),
		Description:            strings.TrimSpace(opts.Description),
		Metadata:               opts.Metadata,
		DisableModelInvocation: opts.DisableModelInvocation,
		UserInvocable:          opts.UserInvocable,
	}

	yml, err := yaml.Marshal(frontmatter)
	if err != nil {
		return "", fmt.Errorf("failed to marshal skill frontmatter: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.Write(yml)
	sb.WriteString("---\n\n")

	if heading := strings.TrimSpace(opts.Heading); heading != "" {
		sb.WriteString("# ")
		sb.WriteString(heading)
		sb.WriteString("\n\n")
	}

	if body := strings.TrimSpace(opts.Body); body != "" {
		sb.WriteString(body)
		sb.WriteString("\n")
	}

	return sb.String(), nil
}
