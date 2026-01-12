# Product Requirements Document: agentspack

**Version:** 1.0  
**Last Updated:** January 2026  
**Status:** Active Development

---

## Executive Summary

**agentspack** is a command-line tool that generates provider-specific AI agent context files from a unified template library. It eliminates the tedious work of manually maintaining separate configuration files for different AI coding assistants (Cursor, Claude Code, Codex) by generating them all from a single source of truth.

---

## Problem Statement

### The Pain Point

Development teams using multiple AI coding assistants face a recurring problem:

1. **Different providers require different formats** — Cursor uses `.cursor/rules/`, Claude Code uses `.claude/`, Codex uses `.codex/skills/`
2. **Manual duplication is error-prone** — Teams copy-paste the same guidance across multiple files, leading to inconsistencies
3. **Maintenance becomes a burden** — Updating a single coding standard requires editing 3+ files across different formats
4. **No standardization** — Teams waste time creating their own templates from scratch

### The Solution

agentspack provides:

- **Unified template library** — Curated best-practice templates for common scenarios
- **Multi-provider generation** — One command generates all provider-specific files
- **Embedded distribution** — Self-contained binary with templates baked in
- **Extensible architecture** — Easy to add new providers and templates

---

## Project Goals

### Primary Goals

1. **Eliminate manual duplication** — Generate all provider files from a single source
2. **Provide production-ready templates** — Ship with curated best practices for common tech stacks
3. **Simple UX** — Interactive wizard requires zero configuration files
4. **Self-contained distribution** — Single binary with all templates embedded

### Non-Goals (Out of Scope)

- Runtime AI agent orchestration or execution
- Template validation or linting
- Multi-language support (English only initially)
- Cloud-hosted template registry

---

## How It Works

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      agentspack CLI                          │
│                                                              │
│  ┌────────────┐    ┌─────────────┐    ┌────────────────┐  │
│  │   Wizard   │───▶│  Generator  │───▶│   Providers    │  │
│  │  (huh UI)  │    │  (Orchestr) │    │  (Adapters)    │  │
│  └────────────┘    └─────────────┘    └────────────────┘  │
│                           │                     │           │
│                           ▼                     ▼           │
│                    ┌─────────────┐      ┌────────────┐    │
│                    │  Embedded   │      │   Output   │    │
│                    │  Templates  │      │   Files    │    │
│                    └─────────────┘      └────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

### Component Breakdown

#### 1. **Wizard** (`internal/wizard/`)

- Interactive CLI using `charmbracelet/huh` library
- Collects user preferences (providers, tech stacks, output path)
- Validates inputs and provides defaults
- Handles path expansion (`~/` → home directory, absolute paths)

#### 2. **Generator** (`internal/generator/`)

- Orchestrates the generation process
- Determines whether to use embedded or local filesystem
- Creates output directories
- Invokes provider-specific generators

#### 3. **Content/FileSystem** (`internal/content/`)

- Abstraction layer over embedded and local filesystems
- Embedded templates are baked into the binary at build time
- Supports local override for development (checks for `system/` folder)
- Provides consistent interface for reading templates

#### 4. **Providers** (`internal/providers/`)

Each provider implements the `Provider` interface:

```go
type Provider interface {
    Name() string
    Generate(config *wizard.Config, fs content.FileSystem, outputDir string) error
}
```

**Current Providers:**

- **Cursor** — Generates `.cursor/rules/` with RULE.md files and frontmatter
- **Claude Code** — Generates `.claude/` with rules, skills, agents, and commands
- **Codex** — Generates `.codex/` with AGENTS.md and skills

#### 5. **Templates** (`internal/templates/`)

- Workflow orchestration logic
- Normalizes step names and descriptions
- Generates multi-step workflow references

---

## Template Library Structure

### `system/base/`

Provider-specific base instructions that set workflow expectations:

- `base.md` — Shared workflow guidance (planning, development phases, code reviews)
- `Cursor.md` — Cursor-specific instructions
- `Claude.md` — Claude Code-specific instructions
- `Codex.md` — Codex-specific instructions

### `system/rules/`

Tech stack and domain-specific coding guidelines:

- `global/` — Coding styles, error handling, validation (always applied)
- `backend/` — API development, database queries, data modeling
- `frontend/react/` — React components, responsive design

### `system/agents/`

Specialized AI agent personas for specific tasks:

- Root level: `ui-designer.md`, `ux_researcher.md`, `trend-researcher.md`, `visual_storyteller.md`, `ux-enhancer.md`
- `backend/` subfolder: `backend-typescript-developer.md`, `backend-python-developer.md`, `cloudflare-workers-developer.md`

Each agent has frontmatter with metadata:

```yaml
---
name: agent-name
description: When to use this agent
model: opus # optional
color: red # optional
---
```

### `system/workflows/`

Multi-step process templates:

- `planning/` — PRD creation, market research, UX/UI design, todo generation
- `development/` — Development workflow guidance

Workflow files are numbered (e.g., `01_create_prd.md`, `02_run_market_research.md`) and generate both:

- Individual step files (for manual invocation)
- Orchestrator file (runs complete workflow)

---

## Provider Output Formats

### Cursor

```
.cursor/
├── rules/
│   ├── global/
│   │   └── RULE.md              # All global rules concatenated
│   ├── backend-{rule}/
│   │   └── RULE.md              # Individual rule with globs
│   ├── agent-{name}/
│   │   └── RULE.md              # Agent as invokable rule
│   └── workflow-{name}/
│       └── RULE.md              # Workflow orchestrator
└── AGENTS.md                    # Base instructions (if requested)
```

Each RULE.md has YAML frontmatter:

```yaml
---
description: "Rule description"
alwaysApply: true|false
globs: ["*.tsx", "src/**"] # optional
---
```

### Claude Code

```
.claude/
├── rules/
│   ├── global.md                # All global rules
│   └── backend.md               # Tech stack rules (if rules mode)
├── skills/
│   └── backend-guidelines/      # Tech stack skills (if skills mode)
│       └── SKILL.md
├── agents/
│   └── {agent-name}.md          # Sub-agents with frontmatter
└── commands/
    └── {workflow-name}.md       # Slash commands for workflows
CLAUDE.md                        # Base instructions (if requested)
```

### Codex

```
.codex/
└── skills/
    ├── backend-guidelines/
    │   └── SKILL.md
    ├── {agent-name}/
    │   └── SKILL.md
    └── workflow-{name}/
        └── SKILL.md
AGENTS.md                        # Base + global rules (always generated)
```

---

## Key Features

### 1. Interactive Wizard

No config files needed — just run `agentspack` and follow the prompts:

1. Select providers (multi-select)
2. Choose Claude Code mode (rules vs skills) if applicable
3. Select tech stacks (multi-select)
4. Choose whether to generate base file
5. Specify output directory (supports `~/`, absolute, and relative paths)

### 2. Embedded Templates

Templates are embedded in the binary at build time:

- `./build.sh` syncs `system/` → `internal/content/system/`
- Go's `//go:embed` directive bakes files into the binary
- Binary is self-contained and portable
- Local `system/` folder takes precedence for development

### 3. Path Intelligence

- **Absolute paths** (`/Users/you/projects/output`) → used as-is
- **Tilde expansion** (`~/Documents/output`) → expands to home directory
- **Relative paths** (`./dist/output`) → resolved from current directory

### 4. Multi-Provider Support

Generate for multiple providers in one run:

- Each provider gets its own subdirectory
- Providers transform the same source templates into their native format
- Adding new providers requires implementing a single interface

### 5. Subdirectory Agent Support

Agents can be organized in subdirectories (e.g., `system/agents/backend/`):

- All providers recursively scan for `*.md` files
- Maintains folder-based organization for better template management

---

## Technology Stack

### Core Technologies

- **Language:** Go 1.21+
- **CLI Framework:** Cobra (command structure)
- **TUI Library:** charmbracelet/huh (interactive wizard)
- **Embedding:** Go 1.16+ embed.FS

### Build System

- Bash script (`build.sh`) for template syncing and binary compilation
- No external build tools required
- Platform: macOS (primary), Linux/Windows (future)

### Testing

- Go standard testing (`go test ./...`)
- Unit tests for content and generator packages
- No integration tests yet (future improvement)

---

## Development Workflow

### Building the Binary

```bash
./build.sh
```

This script:

1. Copies `system/` → `internal/content/system/`
2. Runs `go build` with embedded templates
3. Outputs `agentspack` binary

### Running Tests

```bash
go test ./...
```

### Adding a New Provider

1. Create `internal/providers/{provider}.go`
2. Implement the `Provider` interface
3. Call `Register()` in `init()`
4. Add to wizard options in `internal/wizard/wizard.go`

### Adding Templates

1. Add `.md` files to appropriate `system/` subfolder
2. Run `./build.sh` to embed
3. Templates automatically discovered by glob patterns

---

## File Naming Conventions

### Agents

- Filename: `{agent-name}.md`
- Underscores and hyphens both supported
- Frontmatter required with `name:` and `description:`

### Workflows

- Filename: `{NN}_{step-name}.md` (e.g., `01_create_prd.md`)
- Number prefix determines ordering
- Generates both individual steps and orchestrator

### Rules

- Filename: `{rule-name}.md`
- Underscores converted to hyphens in output
- Nested directories supported

---

## Constraints & Limitations

### Current Limitations

1. **Platform:** macOS only (Linux/Windows support planned)
2. **Language:** English templates only
3. **No validation:** Templates not validated for correctness
4. **No overrides:** Can't selectively override embedded templates
5. **No configuration file:** Wizard-only (no `agentspack.yaml`)

### Design Decisions

- **Embedded-first:** Prioritizes portability over customization
- **Markdown-only:** No custom DSL or complex formats
- **Provider adapters:** Each provider owns its output format
- **No runtime:** Pure code generation tool, not an agent runtime

---

## Future Roadmap

### Near-term (Next 3-6 months)

- [ ] Windows and Linux support
- [ ] `agentspack init` for configuration file generation
- [ ] User-provided markdown folder conversion
- [ ] Template overrides via local files
- [ ] More tech stacks (Vue, Angular, Python, Go)

### Medium-term (6-12 months)

- [ ] Template validation and linting
- [ ] Community template registry
- [ ] Version control for templates
- [ ] Template dependency management
- [ ] Agent chaining and orchestration

### Long-term (12+ months)

- [ ] Multi-language template support
- [ ] Cloud-based template sharing
- [ ] AI-powered template suggestions
- [ ] Runtime agent execution support

---

## Success Metrics

### Adoption Metrics

- Number of downloads/installations
- Number of providers generated per session
- Number of tech stacks selected per session

### Quality Metrics

- Build success rate
- Test coverage percentage
- Number of reported bugs/issues
- Community contributions (PRs, issues)

### Usage Patterns

- Most popular providers
- Most popular tech stacks
- Most common output paths
- Most used templates

---

## Open Questions / Decisions Needed

1. **Template versioning:** How do we handle breaking changes to templates?
2. **Community templates:** Should we support a plugin/extension system?
3. **Provider priority:** Which providers should we prioritize next (GitHub Copilot, Aider, etc.)?
4. **Configuration file format:** If we add `agentspack.yaml`, what should it look like?
5. **Template validation:** Should we enforce schema/structure for templates?

---

## Contributing Guidelines

### For Template Contributors

1. Add markdown files to appropriate `system/` folder
2. Follow naming conventions
3. Include frontmatter where required
4. Test generation with `./build.sh && ./agentspack`

### For Provider Contributors

1. Implement `Provider` interface
2. Add comprehensive comments
3. Include example output structure
4. Update this PRD with new provider details

### For Core Contributors

1. Maintain backward compatibility
2. Add tests for new features
3. Update README and PRD
4. Follow Go best practices

---

## References

- **Repository:** [GitHub link when available]
- **Issues:** [GitHub Issues link]
- **Discussions:** [GitHub Discussions link]
- **Documentation:** `README.md` in repository root

---

## Appendix: Technical Details

### Provider Interface

```go
type Provider interface {
    Name() string
    Generate(config *wizard.Config, fs content.FileSystem, outputDir string) error
}
```

### Config Structure

```go
type Config struct {
    Providers       []string
    TechStacks      []string
    GenerateBase    bool
    OutputDir       string
    ClaudeCodeMode  ClaudeCodeMode // "rules" or "skills"
}
```

### FileSystem Interface

```go
type FileSystem interface {
    ReadFile(path string) ([]byte, error)
    ReadDir(path string) ([]fs.DirEntry, error)
    Glob(pattern string) ([]string, error)
    Stat(path string) (fs.FileInfo, error)
}
```

---

**End of PRD**
