# agentspack

A command-line tool that generates **provider-specific AI agent context files** for your projects. Stop manually maintaining separate configuration files for Cursor, Claude Code, and Codex — let agentspack generate them all from a unified template library.

## What's New

- **Automatic Postman collection generation** — Agents now create and maintain a Postman collection (under a `postman/` folder) whenever they finish coding or updating an API. Every endpoint includes full documentation, request parameters, body schemas, and realistic examples.
- **API Guide documentation** — Agents automatically keep a `docs/API_GUIDE.md` file up to date with clear instructions on how to use every endpoint, including all parameters, request/response formats, and example usage.

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

6. **GitHub sync** (if `sync_repos.md` exists) — Optionally sync generated files to multiple GitHub repositories:
   - Create Pull Requests for review, or
   - Merge directly to a target branch

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

? Sync generated files to GitHub repositories? Yes

? How should changes be applied?
  > Create Pull Request (for review)

? Target branch for PR/merge main

=== Configuration Summary ===

Providers:   cursor, claude-code
Tech Stacks: backend, react
Base file:   yes
Output:      ./dist/agentspack
Claude Code: rules mode
GitHub Sync: Yes (PR to main)

Using embedded templates
Generating files to: /path/to/your/project/dist/agentspack

Generating for cursor...
Generating for claude-code...

Generation complete!

Syncing to 2 GitHub repositories...

  → myorg/frontend-app ... PR created: https://github.com/myorg/frontend-app/pull/42
  → myorg/backend-api ... PR created: https://github.com/myorg/backend-api/pull/17

Sync complete! 2 PRs created.
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
│   │   ├── syncer/          # GitHub sync functionality
│   │   │   ├── syncer.go    # Sync orchestration
│   │   │   └── github.go    # GitHub CLI wrapper
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

#### Development Workflows

Unlike planning workflows that follow a sequential multi-step process, development workflows are standalone commands you can run at any time:

- **Start Session** — Run this whenever you begin a new session (e.g., launching `claude`, opening a fresh Cursor chat, or after clearing the agent's context). It ensures the agent reads all key documentation files — the PRD, technical requirements, and existing todos — so it has full project context before you start giving instructions.
- **Next Todo** — Tells the agent to pick up the next incomplete item from the todos file and work on it following the project's established development best practices: planning, coding, reviewing, and marking the task as complete.

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

| Provider    | Status      | Output Format                      |
| ----------- | ----------- | ---------------------------------- |
| Cursor      | Implemented | `.cursorrules` and rules directory |
| Claude Code | Implemented | `CLAUDE.md` and rules/skills       |
| Codex       | Implemented | `AGENTS.md` and guidance files     |

## Development

### Adding a New Provider

1. Create a new file in `internal/providers/` implementing the `Provider` interface
2. Register the provider in the `init()` function
3. Add the provider option to `internal/wizard/wizard.go`

### Adding New Templates

1. Add markdown files to the appropriate folder under `system/`
2. Run `./build.sh` to embed the new templates
3. Update provider adapters if needed to include the new content

## GitHub Sync Feature

agentspack can automatically distribute generated files to multiple GitHub repositories. This is useful for teams that maintain AI agent configurations across several codebases.

### Prerequisites

1. Install the [GitHub CLI](https://cli.github.com/) (`gh`)
2. Authenticate with GitHub:
   ```bash
   gh auth login
   ```

### Setup

Create a `sync_repos.md` file in the directory where you run agentspack:

```markdown
# Repositories to sync agentspack files to

# One repo per line in owner/repo format

myorg/frontend-app
myorg/backend-api
myorg/shared-lib

# Lines starting with # are comments

# Empty lines are ignored
```

### Usage

When `sync_repos.md` exists, the wizard will automatically ask additional questions after the standard generation options:

1. **Sync to GitHub?** — Whether to sync the generated files to the listed repositories
2. **PR or merge?** — Choose how changes are applied:
   - **Create Pull Request** — Creates a PR for review (recommended for teams)
   - **Merge directly** — Pushes changes directly to the target branch
3. **Target branch** — Which branch to create PRs against or merge into (default: `main`)

### Example Sync Output

```
Syncing to 3 GitHub repositories...

  → myorg/frontend-app ... PR created: https://github.com/myorg/frontend-app/pull/42
  → myorg/backend-api ... PR created: https://github.com/myorg/backend-api/pull/17
  → myorg/shared-lib ... skipped (no changes)

Sync complete! 2 PRs created, 1 skipped.
```

### How It Works

For each repository in `sync_repos.md`:

1. Clones the repo (shallow clone for speed)
2. Creates a new branch: `agentspack/update-YYYYMMDD-HHMMSS`
3. Copies all generated files to the repository root
4. Commits changes (if any)
5. Creates a PR or merges directly, based on your selection

### Special Cases

- **Empty repositories** — If a repo has no commits yet, agentspack will initialize it by pushing directly to the target branch (PRs require an existing base branch to compare against)
- **No changes detected** — If the generated files are identical to what's already in the repo, it will be skipped
- **Failed repos** — If a repo fails to sync, agentspack continues with the remaining repos and reports failures at the end

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
