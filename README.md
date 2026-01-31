# hack

A CLI tool for managing hack workspaces with pattern-based scaffolding.

## Overview

`hack` creates date-prefixed directories in `~/hack/` for quick project scaffolding. It supports patterns (templates) for common project types.

## Installation

### Homebrew (macOS/Linux)

```bash
brew install cloudygreybeard/tap/hack
```

### Go install

```bash
go install github.com/cloudygreybeard/hack@latest
```

### From source

```bash
git clone https://github.com/cloudygreybeard/hack
cd hack
make install
```

## Shell Integration

Install automatically:

```bash
hack bootstrap --install
```

This detects your shell and appends the integration to the appropriate rc file (`~/.bashrc`, `~/.zshrc`, or `~/.config/fish/config.fish`).

Or print the snippet to add manually:

```bash
hack bootstrap              # auto-detect shell
hack bootstrap --shell zsh  # force specific shell
```

## Usage

```bash
# Navigate to most recent hack directory
hack

# Navigate to directory matching filter string
hack api

# Create a new workspace
hack create my-project

# Create with a pattern (template)
hack create my-cli -p hello

# Create with interactive prompts for variables
hack create my-cli -p hello -i

# List workspaces
hack list
hack list api    # filter by substring

# Manage patterns (templates)
hack pattern list
hack pattern show hello
hack pattern install ./my-pattern

# Configuration
hack config show
hack config init    # create ~/.hack.yaml

# Show version
hack version
```

## Patterns

Patterns are stored in `~/.hack/patterns/`. Each pattern contains:

- `pattern.yaml` - metadata and variable definitions
- `template/` - files to copy (supports Go text/template)

### Example Pattern

A minimal example pattern is included in `examples/patterns/hello/`:

```bash
hack pattern install examples/patterns/hello
hack create my-project -p hello
```

See `examples/README.md` for details on creating your own patterns.

### Installing Patterns

```bash
hack pattern install <path>    # Install from local directory
hack pattern list              # List installed patterns
hack pattern show <name>       # Show pattern details
```

## Configuration

Configuration is read from `~/.hack.yaml`, environment variables (`HACK_*`), and command-line flags (in that order of precedence).

Create a config file with `hack config init`, or use environment variables:

```yaml
# ~/.hack.yaml

# Root directory for hack workspaces
root_dir: ~/hack

# Directory for patterns (templates)
patterns_dir: ~/.hack/patterns

# Editor to open README.md after creating workspace
editor: vim

# Initialize git repository on create
git_init: true

# Create README.md on create
create_readme: true

# Enable interactive mode by default (prompt for pattern variables)
interactive: false

# Default GitHub organization for module paths (empty uses example.com)
# default_org: your-org
```

Environment variables use the `HACK_` prefix: `HACK_ROOT_DIR`, `HACK_PATTERNS_DIR`, etc.

## License

Apache 2.0. See [LICENSE](LICENSE).
