# gitignore

A command-line tool to manage `.gitignore` files using templates from GitHub repositories.

## Features

- List all available gitignore templates from a GitHub repository
- Add gitignore templates to your project's `.gitignore` file
- Remove previously added templates
- Configurable template source
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
# Add a root-level template
gitignore add Go

# Add a template from a subdirectory
gitignore add Global/macOS
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

You can configure the template source by creating a configuration file at one of these locations:

- `~/.config/gitignore/gitignorerc`
- `~/.gitignorerc`

### Configuration Format

```ini
# gitignore configuration
gitignore.template.url = https://github.com/github/gitignore
gitignore.default-types = Go, Global/macOS, Global/VisualStudioCode
```

### Configuration Options

| Option                    | Description                                          | Default                               |
| ------------------------- | ---------------------------------------------------- | ------------------------------------- |
| `gitignore.template.url`  | GitHub repository URL for templates                  | `https://github.com/github/gitignore` |
| `gitignore.default-types` | Comma-separated list of templates for `init` command | (empty)                               |

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
