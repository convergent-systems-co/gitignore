# gitignore

A command-line tool to manage `.gitignore` files using templates from multiple sources.

## Features

- List all available gitignore templates from multiple sources
- Add gitignore templates to your project's `.gitignore` file
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

Or clone and build:

```bash
git clone https://github.com/polliard/gitignore.git
cd gitignore
make build
```

## Usage

### List Available Templates

```bash
gitignore --list
```

This will fetch and display all available `.gitignore` templates from the configured source (default: https://github.com/github/gitignore).

### Add a Template

```bash
# Add a template (uses priority order: local -> GitHub -> Toptal)
gitignore add Go

# Add a template from a subdirectory
gitignore add Global/macOS

# Add from a specific source when duplicates exist
gitignore add github/rust
gitignore add toptal/rust
gitignore add local/mytemplate
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
gitignore delete Go
```

This removes the specified section from your `.gitignore` file.

### Initialize with Default Types

If you have configured default types in your config file, you can initialize a `.gitignore` with all of them at once:

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

You can configure the template sources by creating a configuration file at one of these locations:

- `~/.config/gitignore/gitignorerc`
- `~/.gitignorerc`

### Configuration Format

```ini
# gitignore configuration

# GitHub repository URL for templates
gitignore.template.url = https://github.com/github/gitignore

# Enable Toptal gitignore API as a fallback source
enable.toptal.gitignore = true

# Path to local templates directory
gitignore.local-templates-path = ~/.config/gitignore/templates

# Default types for 'init' command
gitignore.default-types = Go, Global/macOS, Global/VisualStudioCode
```

### Configuration Options

| Option                           | Description                                          | Default                               |
| -------------------------------- | ---------------------------------------------------- | ------------------------------------- |
| `gitignore.template.url`         | GitHub repository URL for templates                  | `https://github.com/github/gitignore` |
| `enable.toptal.gitignore`        | Enable Toptal API as fallback (true/false)           | `false`                               |
| `gitignore.local-templates-path` | Directory for local template files                   | `~/.config/gitignore/templates`       |
| `gitignore.default-types`        | Comma-separated list of templates for `init` command | (empty)                               |

### Example: Enabling Toptal Fallback

```ini
# Use GitHub as primary, Toptal as fallback when templates aren't found
gitignore.template.url = https://github.com/github/gitignore
enable.toptal.gitignore = true
```

### Example: Using a Custom Template Repository

```ini
# Use a company-specific template repository
gitignore.template.url = https://github.com/mycompany/gitignore-templates
```

### Example: Setting Default Types

```ini
# Configure templates to add when running 'gitignore init'
gitignore.default-types = Go, Global/macOS, Global/VisualStudioCode, Global/JetBrains
```

## Local Templates (Custom Overrides)

You can create custom gitignore templates that **always take precedence** over remote sources.

### Setup

1. Create the local templates directory (or configure a custom path):
   ```bash
   mkdir -p ~/.config/gitignore/templates
   ```

2. Add your custom template files with the `.gitignore` extension:
   ```bash
   # Example: Create a custom "myproject" template
   cat > ~/.config/gitignore/templates/myproject.gitignore << 'EOF'
   # My company-specific ignores
   .internal/
   *.secret
   .env.local
   EOF
   ```

3. Use it just like any other template:
   ```bash
   gitignore add myproject
   ```

### Custom Templates Path

You can configure a different directory for local templates:

```ini
# Use a different directory for local templates
gitignore.local-templates-path = ~/my-gitignore-templates
```

### Priority Order

Templates are searched in this order:

1. **Local** (configured path, default: `~/.config/gitignore/templates/`) - Always checked first
2. **GitHub** - The repository specified in `gitignore.template.url`
3. **Toptal** - If `enable.toptal.gitignore = true`

This means:
- If you have `myproject.gitignore` in your local templates directory, it will be used
- You can override any built-in template with your own custom version
- Custom templates don't need to exist in any remote source

### Specifying a Source

When the same template exists in multiple sources, use the `source/name` syntax to pick a specific one:

```bash
# Use Rust from Toptal instead of GitHub
gitignore add toptal/rust

# Explicitly use the GitHub version
gitignore add github/Rust

# Use a local custom template
gitignore add local/myproject
```

The `gitignore list` command shows all templates with their source prefix for easy reference.

### Use Cases

- **Company-specific patterns**: Add internal tooling, proprietary files
- **Extended templates**: Start with a standard template and add more rules
- **Offline usage**: Keep templates locally for use without internet
- **Quick prototyping**: Test new ignore patterns before contributing upstream

## Supported Remote Sources

### GitHub Repositories

Any GitHub repository containing `.gitignore` files can be used:
```ini
gitignore.template.url = https://github.com/github/gitignore
```

### Toptal gitignore API

The Toptal gitignore.io API (https://www.toptal.com/developers/gitignore/api) provides additional generated templates. Enable it as a fallback source:
```ini
enable.toptal.gitignore = true
```

## Development

### Prerequisites

- Go 1.21 or later
- Make (optional, for using Makefile)

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Run tests
make test

# Run tests with coverage
make test-coverage

# Format code
make fmt

# Clean build artifacts
make clean
```

### Running Tests

```bash
# Run all tests
make test

# Run short tests (skip integration tests)
make test-short

# Run only integration tests
make test-integration
```

### Cross-Platform Builds

The Makefile supports building for multiple platforms:

```bash
# Build all platforms
make build-all

# Build specific platform
make darwin-amd64    # macOS Intel
make darwin-arm64    # macOS Silicon
make linux-amd64     # Linux 64-bit
make windows-amd64   # Windows 64-bit
```

### Creating a Release

To create a new release:

1. Tag the commit: `git tag v1.0.0`
2. Push the tag: `git push origin v1.0.0`
3. GitHub Actions will automatically build and create the release

## How It Works

1. **List**: Fetches the repository tree from GitHub API and filters for `.gitignore` files
2. **Add**: Downloads the raw content, wraps it in section markers, and appends to `.gitignore`
3. **Delete**: Scans `.gitignore` for section markers and removes the matching section

### Section Markers

Templates are added with markers to enable selective removal:

```gitignore
### START: Go
<template content>
### END: Go
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request
