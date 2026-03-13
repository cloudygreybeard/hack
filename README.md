# hack

A CLI tool for managing development workspaces with pattern-based scaffolding, workspace metadata, and plugin support.

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

The shell wrapper enables `hack` to change your working directory. Install it with:

```bash
hack bootstrap --install
```

This detects your shell and appends the integration to the appropriate rc file (`~/.bashrc`, `~/.zshrc`, or `~/.config/fish/config.fish`).

To print the snippet without installing:

```bash
hack bootstrap              # auto-detect shell
hack bootstrap --shell zsh  # specific shell
```

## Usage

### Workspace Navigation

```bash
hack                # cd to the most recently modified workspace
hack api            # cd to the best-matching workspace (fuzzy ranked)
hack list           # list all workspaces
hack list api       # filter by substring
```

### Creating Workspaces

```bash
hack create my-project                    # empty workspace with README.md
hack create my-project -p go-cli          # apply a pattern
hack create my-project -p go-cli -a myapp # custom app directory name
hack create my-project -p go-cli -i       # interactive variable prompts
hack create my-project --label lang=go    # set labels at creation time
hack create my-project -p go-cli --dry-run # preview without writing files
```

### Editing and Lifecycle

```bash
hack edit my-project        # open in editor/IDE (configurable)
hack rm my-project          # remove workspace (with confirmation)
hack archive my-project     # move to .archive/
hack archive --list         # list archived workspaces
hack archive --restore foo  # restore from archive
```

### Workspace Metadata

Workspaces store Kubernetes-style labels and annotations in `.hack.yaml`:

```bash
hack label my-project domain=aro lang=go   # set labels
hack label my-project domain-              # remove a label (trailing dash)
hack label my-project --list               # show labels
hack annotate my-project jira=OCPBUGS-123  # set annotations
hack list -l domain=aro                    # filter by label selector
hack list -l domain=aro,lang=go            # multiple labels (AND)
hack list --show-labels                    # display labels alongside names
```

## Patterns

Patterns are reusable project templates stored in `~/.hack/patterns/`. Each contains a `pattern.yaml` and a `template/` directory with files processed through Go `text/template`.

### Managing Patterns

```bash
hack pattern list                          # list installed patterns
hack pattern show go-cli                   # show details, variables, hooks
hack pattern install ./my-pattern          # install from local directory
hack pattern install org/repo              # install from GitHub (shorthand)
hack pattern install org/repo//subpath     # install from repo subdirectory
hack pattern install https://example.com/p.tar.gz  # install from tarball
hack pattern sync ./patterns              # bulk install from a directory
hack pattern update                        # re-install all from recorded sources
hack pattern update go-cli                 # update a specific pattern
hack pattern list --outdated               # check for newer versions
```

### Extracting Patterns

Reverse-scaffold a pattern from an existing workspace:

```bash
hack pattern extract my-project            # extract to ./my-project/
hack pattern extract my-project -n my-pat  # custom pattern name
hack pattern extract my-project --install  # extract and install directly
hack pattern extract my-project --no-templatise  # copy files without substitution
hack pattern extract my-project --app-only myapp # extract only an app subdirectory
```

The extract command replaces concrete values (workspace name, app name, module path, year) with template variables and escapes pre-existing template expressions for round-trip safety.

### Pattern Structure

```
~/.hack/patterns/my-pattern/
├── pattern.yaml
└── template/
    ├── README.md.tmpl        # processed through text/template
    ├── static-file.txt       # copied as-is
    └── {{app_name}}/         # directory name expanded from variables
        ├── main.go.tmpl
        └── go.mod.tmpl
```

### pattern.yaml

```yaml
name: my-pattern
description: Example pattern
version: 1.0.0
weight: 10
labels:
  lang: go
  type: cli
default_labels:
  lang: go
inherits:
  - pattern: base-pattern
  - patternSelector:
      matchLabels:
        layer: common
variables:
  - name: name
    description: Project name
    required: true
  - name: module
    description: Go module path
    default: ""
post_create:
  - "cd {{.app_name}} && go mod tidy"
```

### Template Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `{{.name}}` | Workspace name | `my-project` |
| `{{.app_name}}` | App directory name | `my-tool` |
| `{{.Name}}` | TitleCase app name | `MyTool` |
| `{{.module}}` | Go module path | `github.com/org/my-tool` |
| `{{.year}}` | Current year | `2026` |
| `{{.date}}` | Creation date | `2026-01-26` |

### Pattern Inheritance

Patterns can inherit from others via `inherits` in `pattern.yaml`. Each entry is either a direct name reference or a label-based selector:

```yaml
inherits:
  - pattern: base-refs
  - patternSelector:
      matchLabels:
        layer: common
```

Inheritance is resolved with topological sort. The `weight` field controls application order (lower weight applied first). Cycles are detected and rejected.

### Example Pattern

A minimal example pattern is included in `examples/patterns/hello/`:

```bash
hack pattern install examples/patterns/hello
hack create my-project -p hello
```

## Plugins

Executables in `~/.hack/plugins/` are automatically discovered and registered as subcommands. A file named `hack-deploy` becomes available as `hack deploy`.

```bash
hack plugin list     # list installed plugins
```

Plugins receive these environment variables:

- `HACK_ROOT_DIR` -- root directory for workspaces
- `HACK_PATTERNS_DIR` -- patterns directory
- `HACK_PLUGINS_DIR` -- plugins directory

## Configuration

Configuration is read from (in ascending order of precedence):

1. Built-in defaults
2. Config file (`~/.hack.yaml`)
3. Environment variables (`HACK_*`)
4. Command-line flags

Create a config file:

```bash
hack config init     # create ~/.hack.yaml with defaults
hack config show     # display current configuration
hack config path     # show config file location
```

### Configuration Options

```yaml
# ~/.hack.yaml
root_dir: ~/hack
patterns_dir: ~/.hack/patterns
plugins_dir: ~/.hack/plugins
editor: vim
ide: ""
edit_mode: auto       # auto, terminal, or ide
git_init: true
create_readme: true
interactive: false
default_org: ""       # GitHub org for module paths (empty uses example.com)
```

## Shell Completions

```bash
hack completion bash | source /dev/stdin   # bash
hack completion zsh > "${fpath[1]}/_hack"  # zsh
hack completion fish | source              # fish
hack completion powershell | Out-String | Invoke-Expression  # powershell
```

## Demo

The demo recording runs real `hack` commands inside a container for reproducibility. To re-record:

```bash
./hack/record-demo.sh
```

See `hack/demo/` for the Containerfile, demo script, and stub patterns used in the recording.

## License

Apache 2.0. See [LICENSE](LICENSE).
