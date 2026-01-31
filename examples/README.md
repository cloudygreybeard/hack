# Examples

This directory contains example patterns to demonstrate hack's template system.

## hello

A minimal pattern showing the basic structure:

```
examples/patterns/hello/
├── pattern.yaml           # Pattern metadata and variables
└── template/              # Files to scaffold
    ├── README.md.tmpl     # Processed through Go text/template
    └── .gitignore         # Copied as-is (no .tmpl suffix)
```

### Installation

```bash
hack pattern install examples/patterns/hello
```

### Usage

```bash
hack create my-project -p hello
```

## Creating Your Own Patterns

See the [Pattern System documentation](../README.md#patterns) for details on creating patterns.

Pattern files use Go's `text/template` syntax. Available variables:

| Variable | Description |
|----------|-------------|
| `{{.name}}` | Project name |
| `{{.Name}}` | TitleCase project name |
| `{{.module}}` | Go module path |
| `{{.year}}` | Current year |
| `{{.date}}` | Creation date (YYYY-MM-DD) |

Custom variables can be defined in `pattern.yaml`.
