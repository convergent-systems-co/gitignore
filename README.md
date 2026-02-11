# gitignore

A command-line tool to manage `.gitignore` files using templates from multiple sources.

## Features

- List all available gitignore templates from multiple sources
- Search templates by name
- Add gitignore templates to your project's `.gitignore` file
- **Add paths/patterns directly** - no template needed for simple ignores
- Remove previously added templates
- **Local template overrides** - create custom templates that take precedence
- **Multiple source support** - GitHub repositories and Toptal API
- Configurable template sources with priority ordering
- Cross-platform support (macOS Intel/Silicon, Windows, Linux)

## Installation

### From Release

Download the appropriate binary for your platform from the [Releases](https://github.com/polliard/gitignore/releases) page.

#### macOS

```bash
# Intel Mac
curl -L https://github.com/polliard/gitignore/releases/latest/download/gitignore-darwin-amd64.tar.gz | tar xz
sudo mv gitignore /usr/local/bin/

# Apple Silicon Mac
curl -L https://github.com/polliard/gitignore/releases/latest/download/gitignore-darwin-arm64.tar.gz | tar xz
sudo mv gitignore /usr/local/bin/
```

#### Linux

```bash
curl -L https://github.com/polliard/gitignore/releases/latest/download/gitignore-linux-amd64.tar.gz | tar xz
sudo mv gitignore /usr/local/bin/
```

#### Windows

Download the `.zip` file from the releases page and extract `gitignore.exe` to a directory in your PATH.

### From Source

```bash
go install github.com/polliard/gitignore/src/cmd/gitignore@latest
```

Ensure `$GOPATH/bin` (or `$HOME/go/bin`) is in your PATH.

Or clone and build:

```bash
git clone https://github.com/polliard/gitignore.git
cd gitignore
make build
```

## Usage

For a complete step-by-step guide, see the [Usage Guide](Usage.md).

### List Available Templates

```bash
gitignore list
```

Output shows templates in path format with source prefix:

```
github/actionscript
github/ada
github/go
github/global/macos
github/global/visualstudiocode
local/myproject
toptal/rust
toptal/rust-analyzer
```

### Search Templates

```bash
gitignore search rust
```

Output:

```
github/rust
toptal/rust
toptal/rust-analyzer
```

### Add a Template

```bash
# Add a template (uses priority order: local -> GitHub -> Toptal)
gitignore add go

# Add from a specific source with full path
gitignore add github/go
gitignore add github/global/macos
gitignore add toptal/rust
gitignore add local/myproject
```

This adds the template content to your `.gitignore` file, wrapped in section markers:

```gitignore
### START: Go
# Binaries for programs and plugins
*.exe
*.exe~
...
### END: Go
```

### Remove a Template

```bash
gitignore delete go
```

This removes the specified section from your `.gitignore` file.

### Ignore Local Paths

Add paths or patterns directly without fetching templates:

```bash
# Add a single pattern
gitignore ignore /dist/

# Add multiple patterns at once
gitignore ignore node_modules *.log tmp/
```

Output:

```
Added '/dist/' to .gitignore
```

Patterns are wrapped in section markers (like templates) so they can be tracked and removed. Duplicate patterns are automatically skipped.

### Remove Ignored Patterns

Remove patterns that were added via `ignore`:

```bash
# Remove a single pattern
gitignore remove /dist/

# Remove multiple patterns
gitignore remove node_modules *.log
```

### Initialize with Default Types

If you have configured default types in your config file:

```bash
gitignore init
```

This adds all templates listed in your `gitignore.default-types` configuration.

### Help

```bash
gitignore --help
gitignore --version
```

## Configuration

Create a configuration file at one of these locations:

- `~/.config/gitignore/gitignorerc`
- `~/.gitignorerc`

The `~/.gitignorerc` file takes precedence if both exist.

### Configuration Format

```ini
# GitHub repository URL for templates
gitignore.template.url = https://github.com/github/gitignore

# Enable Toptal gitignore API as a fallback source
enable.toptal.gitignore = true

# Path to local templates directory
gitignore.local-templates-path = ~/.config/gitignore/templates

# Default types for 'init' command
gitignore.default-types = github/go, github/global/macos, github/global/visualstudiocode
```

### Configuration Options

| Option                           | Description                                    | Default                               |
| -------------------------------- | ---------------------------------------------- | ------------------------------------- |
| `gitignore.template.url`         | GitHub repository URL for templates            | `https://github.com/github/gitignore` |
| `enable.toptal.gitignore`        | Enable Toptal API as fallback (`true`/`false`) | `false`                               |
| `gitignore.local-templates-path` | Directory for local template files             | `~/.config/gitignore/templates`       |
| `gitignore.default-types`        | Comma-separated list for `init` command        | (empty)                               |

### Example Configurations

**Enable Toptal fallback:**

```ini
gitignore.template.url = https://github.com/github/gitignore
enable.toptal.gitignore = true
```

**Use a custom template repository:**

```ini
gitignore.template.url = https://github.com/mycompany/gitignore-templates
```

**Set default types for new projects:**

```ini
gitignore.default-types = github/go, github/global/macos, github/global/visualstudiocode
```

## Local Templates

Create custom gitignore templates that **always take precedence** over remote sources.

### Setup

1. Create the local templates directory:

   ```bash
   mkdir -p ~/.config/gitignore/templates
   ```

2. Add custom template files (must end with `.gitignore`):

   ```bash
   cat > ~/.config/gitignore/templates/myproject.gitignore << 'EOF'
   # My company-specific ignores
   .internal/
   *.secret
   .env.local
   EOF
   ```

3. Use it like any other template:

   ```bash
   gitignore add myproject
   # or explicitly:
   gitignore add local/myproject
   ```

### Priority Order

Templates are searched in this order:

1. **Local** - `~/.config/gitignore/templates/` (or configured path)
2. **GitHub** - Repository from `gitignore.template.url`
3. **Toptal** - If `enable.toptal.gitignore = true`

### Specifying a Source

When the same template exists in multiple sources:

```bash
gitignore search rust
# github/rust
# toptal/rust
# toptal/rust-analyzer

# Use specific source
gitignore add github/rust
gitignore add toptal/rust
```

## Supported Sources

### GitHub Repositories

Any GitHub repository containing `.gitignore` files:

```ini
gitignore.template.url = https://github.com/github/gitignore
```

### Toptal gitignore API

The [Toptal gitignore.io API](https://www.toptal.com/developers/gitignore/api) provides additional templates:

```ini
enable.toptal.gitignore = true
```

## Development

### Prerequisites

- Go 1.23 or later
- Make (optional)

### Building

```bash
make build          # Build for current platform
make build-all      # Build for all platforms
make test           # Run tests
make test-coverage  # Run tests with coverage
make fmt            # Format code
make clean          # Clean build artifacts
```

### Cross-Platform Builds

```bash
make darwin-amd64   # macOS Intel
make darwin-arm64   # macOS Silicon
make linux-amd64    # Linux 64-bit
make windows-amd64  # Windows 64-bit
```

### Creating a Release

1. Tag the commit: `git tag v1.2.0`
2. Push the tag: `git push origin v1.2.0`
3. GitHub Actions automatically builds and creates the release

## How It Works

1. **List/Search**: Fetches templates from configured sources, displays as `source/name` paths
2. **Add**: Downloads content, wraps in section markers, appends to `.gitignore`
3. **Delete**: Scans for section markers, removes matching section

### Section Markers

Templates are wrapped for selective removal:

```gitignore
### START: Go
<template content>
### END: Go
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request
