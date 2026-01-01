# agentspack

A command-line tool that generates **provider-specific AI agent context files** for your projects. Stop manually maintaining separate configuration files for Cursor, Claude Code, and Codex — let agentspack generate them all from a unified template library.

## What is agentspack?

Different AI coding assistants expect different "context" formats and folder structures. Teams waste time manually copying, reshaping, and maintaining multiple variants of the same guidance for each tool they use.

**agentspack** solves this by:
- Providing a curated library of best-practice templates
- Generating provider-specific output with one command
- Supporting multiple tech stacks in a single run

## Quick Start

### Prerequisites

- Go 1.21 or later
- macOS (other platforms planned for future releases)

### Installation

```bash
# Clone the repository
git clone https://github.com/agentspack/agentspack.git
cd agentspack/agentspack

# Build the binary
./build.sh

# Optionally, move to your PATH
mv agentspack /usr/local/bin/
```

### Usage

Simply run the interactive wizard:

```bash
agentspack
```

The wizard will guide you through:

1. **Select providers** — Choose which AI coding tools you want to generate files for:
   - Cursor
   - Claude Code
   - Codex

2. **Claude Code mode** (if selected) — Choose how tech stack rules should be generated:
   - **Rules** — Always loaded, path-scoped rule files
   - **Skills** — Loaded on-demand when relevant

3. **Select tech stacks** — Choose which technology templates to include:
   - Backend
   - React

4. **Base file** — Optionally generate a base instructions file (`CLAUDE.md`, `AGENTS.md`, etc.)

5. **Output directory** — Specify where to write the generated files (default: `./dist/agentspack`)

### Example Session

```
Welcome to agentspack!

? Select providers to generate for
  ✓ Cursor
  ✓ Claude Code

? Claude Code: How should tech stack guidelines be generated?
  > Rule files (always loaded, path-scoped)

? Select tech stacks
  ✓ Backend
  ✓ React

? Generate base instructions file? Yes

? Output directory ./dist/agentspack

=== Configuration Summary ===

Providers:   cursor, claude-code
Tech Stacks: backend, react
Base file:   yes
Output:      ./dist/agentspack
Claude Code: rules mode

Using embedded templates
Generating files to: /path/to/your/project/dist/agentspack

Generating for cursor...
Generating for claude-code...

Generation complete!
```

## Project Structure

```
agents/
├── agentspack/              # Main application code
│   ├── cmd/                 # CLI command definitions (Cobra)
│   ├── internal/
│   │   ├── content/         # Embedded filesystem handling
│   │   ├── generator/       # Core generation logic
│   │   ├── providers/       # Provider-specific adapters
│   │   │   ├── cursor.go    # Cursor output format
│   │   │   ├── claude_code.go # Claude Code output format
│   │   │   └── codex.go     # Codex output format
│   │   ├── templates/       # Template processing
│   │   └── wizard/          # Interactive prompt logic
│   ├── system/              # Source markdown templates
│   │   ├── agents/          # Agent definitions (UI designer, UX researcher, etc.)
│   │   ├── base/            # Base configuration files per provider
│   │   ├── rules/           # Tech-stack specific rules
│   │   │   ├── backend/     # Backend development rules
│   │   │   ├── frontend/    # Frontend development rules
│   │   │   └── global/      # Global coding standards
│   │   └── workflows/       # Workflow templates (planning, development)
│   ├── build.sh             # Build script
│   ├── go.mod
│   └── main.go
└── .agents/                 # Project documentation
    └── PRD.md               # Product Requirements Document
```

## Template Categories

### Base Templates
Provider-specific base configuration files that set up the foundation for AI assistant interaction.

### Rules
Organized by domain:
- **Global** — Coding styles, error handling, input validation
- **Backend** — API development, database queries, data modeling
- **Frontend** — React components, responsive design

### Agents
Specialized AI agent prompts:
- UI Designer
- UX Researcher
- Visual Storyteller

### Workflows
Step-by-step process templates:
- **Planning** — PRD creation, market research, development phases, UX research, UI design, todo generation

## Building from Source

The build script performs two steps:

```bash
./build.sh
```

1. **Syncs templates** — Copies `system/` folder to `internal/content/system/` for embedding
2. **Builds binary** — Compiles the Go application with embedded templates

The resulting binary is self-contained with all templates embedded — no external files needed at runtime.

## Running Tests

```bash
cd agentspack
go test ./...
```

## Output Structure

After running agentspack, your output directory will contain:

```
dist/agentspack/
├── cursor/
│   └── [provider-specific files]
├── claude-code/
│   └── [provider-specific files]
└── codex/
    └── [provider-specific files]
```

Each provider folder contains the generated context files in the format expected by that tool.

## Supported Providers

| Provider | Status | Output Format |
|----------|--------|---------------|
| Cursor | Implemented | `.cursorrules` and rules directory |
| Claude Code | Implemented | `CLAUDE.md` and rules/skills |
| Codex | Implemented | `AGENTS.md` and guidance files |

## Development

### Adding a New Provider

1. Create a new file in `internal/providers/` implementing the `Provider` interface
2. Register the provider in the `init()` function
3. Add the provider option to `internal/wizard/wizard.go`

### Adding New Templates

1. Add markdown files to the appropriate folder under `system/`
2. Run `./build.sh` to embed the new templates
3. Update provider adapters if needed to include the new content

## Future Plans

- User-provided markdown folder conversion
- `agentspack init` for repeatable configuration
- Windows/Linux support
- Template overrides via local files
- More tech stack options

## License

See LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.
