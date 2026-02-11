# Usage Guide

This guide walks you through installing, configuring, and using the `gitignore` CLI tool.

## Step 1: Install

Build and install the CLI:

```bash
# Clone the repository
git clone https://github.com/polliard/gitignore.git
cd gitignore

# Build for your platform
make build

# Install to your GOPATH/bin
make install
```

Or download a pre-built binary from the releases page and add it to your PATH.

## Step 2: Configure

Create a configuration file to set your default templates:

```bash
# Create config directory
mkdir -p ~/.config/gitignore

# Copy the example config
cp gitignorerc.example ~/.config/gitignore/gitignorerc
```

Edit `~/.config/gitignore/gitignorerc` to customize your defaults:

```ini
# Templates to add when running 'gitignore init'
gitignore.default-types = github/global/macos, github/global/visualstudiocode

# Optional: Enable Toptal API for additional templates
enable.toptal.gitignore = true

# Optional: Path for custom local templates
gitignore.local-templates-path = ~/.config/gitignore/templates
```

## Step 3: Initialize a Repository

Navigate to your project and initialize with your default templates:

```bash
cd ~/my-project

# Initialize .gitignore with your configured defaults
gitignore init
```

This creates (or updates) a `.gitignore` file with all templates specified in `gitignore.default-types`.

## Step 4: Add Additional Templates

Add more templates as needed:

```bash
# Add a template (searches local -> GitHub -> Toptal)
gitignore add node

# Add from a specific source
gitignore add github/rust
gitignore add toptal/python
gitignore add github/global/macos

# Search for templates
gitignore search rust

# List all available templates
gitignore list
```

## Quick Reference

| Command                      | Description                                |
| ---------------------------- | ------------------------------------------ |
| `gitignore init`             | Initialize with default templates          |
| `gitignore add <type>`       | Add a template (e.g., `go`, `github/rust`) |
| `gitignore delete <type>`    | Remove a previously added template         |
| `gitignore search <pattern>` | Search templates by name                   |
| `gitignore list`             | List all available templates               |

## Custom Templates

Create custom templates in your local templates directory:

```bash
mkdir -p ~/.config/gitignore/templates

# Create a custom template
cat > ~/.config/gitignore/templates/mycompany.gitignore << 'EOF'
# Company-specific ignores
.internal/
*.company-secret
EOF
```

Then use it like any other template:

```bash
gitignore add mycompany
```

Local templates always take precedence over remote sources.
