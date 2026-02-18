package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/polliard/gitignore/src/pkg/config"
	"github.com/polliard/gitignore/src/pkg/gitignore"
	"github.com/polliard/gitignore/src/pkg/source"
)

// version is set via ldflags at build time, or detected from module info
var version = "dev"

func getVersion() string {
	// If version was set via ldflags, use it
	if version != "dev" {
		return version
	}
	// Otherwise try to get version from module info (set by go install @version)
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return version
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Parse command
	cmd := args[0]

	switch cmd {
	case "--list", "-l", "list":
		return cmdList(cfg, "")
	case "search", "-s":
		if len(args) < 2 {
			return fmt.Errorf("usage: gitignore search <pattern>")
		}
		return cmdList(cfg, args[1])
	case "add":
		if len(args) < 2 {
			return fmt.Errorf("usage: gitignore add <type>")
		}
		return cmdAdd(cfg, args[1])
	case "init":
		return cmdInit(cfg)
	case "delete", "rm":
		if len(args) < 2 {
			return fmt.Errorf("usage: gitignore delete <type>")
		}
		return cmdDelete(args[1])
	case "ignore":
		if len(args) < 2 {
			return fmt.Errorf("usage: gitignore ignore <pattern> [pattern...]")
		}
		return cmdIgnore(args[1:])
	case "remove":
		if len(args) < 2 {
			return fmt.Errorf("usage: gitignore remove <pattern> [pattern...]")
		}
		return cmdRemove(args[1:])
	case "serve":
		return cmdServe()
	case "--help", "-h", "help":
		printUsage()
		return nil
	case "--version", "-v", "version":
		fmt.Printf("gitignore version %s\n", getVersion())
		return nil
	default:
		return fmt.Errorf("unknown command: %s\nRun 'gitignore --help' for usage", cmd)
	}
}

func cmdList(cfg *config.Config, searchPattern string) error {
	return cmdListTo(os.Stdout, cfg, searchPattern)
}

func cmdListTo(w io.Writer, cfg *config.Config, searchPattern string) error {
	// Create source manager
	sm, err := source.NewSourceManager(cfg.LocalTemplatesPath, cfg.TemplateURL, cfg.EnableToptal)
	if err != nil {
		return fmt.Errorf("failed to create source manager: %w", err)
	}

	// Get all files grouped by source
	filesBySource, err := sm.ListBySource()
	if err != nil {
		return fmt.Errorf("failed to list templates: %w", err)
	}

	// Build flat list of all template paths
	var allPaths []string
	var warnings []string

	// Process local templates
	if localResult, ok := filesBySource["local"]; ok {
		if localResult.Error != nil {
			warnings = append(warnings, fmt.Sprintf("⚠️  Local templates: %v (path: %s)", localResult.Error, sm.LocalSource().Dir()))
		} else {
			for _, file := range localResult.Files {
				allPaths = append(allPaths, fmt.Sprintf("local/%s", strings.ToLower(file.Name)))
			}
		}
	}

	// Process remote templates
	for _, src := range sm.RemoteSources() {
		result, ok := filesBySource[src.Name()]
		if !ok {
			continue
		}

		if result.Error != nil {
			msg := fmt.Sprintf("⚠️  %s: %v", formatSourceName(src.Name()), result.Error)
			if src.Name() == "github" {
				if gs, ok := src.(*source.GitHubSource); ok {
					msg += fmt.Sprintf(" (url: %s)", gs.URL())
				}
			}
			warnings = append(warnings, msg)
			continue
		}

		for _, file := range result.Files {
			var path string
			if file.Category == "" {
				path = fmt.Sprintf("%s/%s", strings.ToLower(src.Name()), strings.ToLower(file.Name))
			} else {
				path = fmt.Sprintf("%s/%s/%s", strings.ToLower(src.Name()), strings.ToLower(file.Category), strings.ToLower(file.Name))
			}
			allPaths = append(allPaths, path)
		}
	}

	// Sort all paths alphabetically
	sort.Strings(allPaths)

	// Filter by search pattern if provided
	if searchPattern != "" {
		searchLower := strings.ToLower(searchPattern)
		var filtered []string
		for _, path := range allPaths {
			if strings.Contains(path, searchLower) {
				filtered = append(filtered, path)
			}
		}
		allPaths = filtered
	}

	// Print warnings first (always to stderr)
	for _, warn := range warnings {
		fmt.Fprintln(os.Stderr, warn)
	}
	if len(warnings) > 0 {
		fmt.Fprintln(os.Stderr)
	}

	// Print paths
	if len(allPaths) == 0 {
		if searchPattern != "" {
			fmt.Fprintf(w, "No templates matching '%s'\n", searchPattern)
		} else {
			fmt.Fprintln(w, "No templates available")
		}
		return nil
	}

	for _, path := range allPaths {
		fmt.Fprintln(w, path)
	}

	return nil
}

// formatSourceName returns a human-readable source name
func formatSourceName(source string) string {
	switch source {
	case "local":
		return "Local"
	case "github":
		return "GitHub"
	case "toptal":
		return "Toptal"
	default:
		return source
	}
}

func cmdAdd(cfg *config.Config, templateType string) error {
	return cmdAddTo(os.Stdout, cfg, templateType)
}

func cmdAddTo(w io.Writer, cfg *config.Config, templateType string) error {
	// Create source manager
	sm, err := source.NewSourceManager(cfg.LocalTemplatesPath, cfg.TemplateURL, cfg.EnableToptal)
	if err != nil {
		return fmt.Errorf("failed to create source manager: %w", err)
	}

	// GetAny handles source prefixes automatically (e.g., "github/rust" vs "rust")
	file, content, err := sm.GetAny(templateType)
	if err != nil {
		return err
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Create section name (include category if present)
	sectionName := file.Name
	if file.Category != "" {
		sectionName = file.Category + "/" + file.Name
	}

	// Add to gitignore
	manager := gitignore.NewManager(cwd)
	if err := manager.Add(sectionName, content); err != nil {
		return err
	}

	// Build display path like list/search (lowercase source/category/name)
	var displayPath string
	if file.Category == "" {
		displayPath = fmt.Sprintf("%s/%s", strings.ToLower(file.Source), strings.ToLower(file.Name))
	} else {
		displayPath = fmt.Sprintf("%s/%s/%s", strings.ToLower(file.Source), strings.ToLower(file.Category), strings.ToLower(file.Name))
	}
	fmt.Fprintf(w, "Added '%s' to .gitignore\n", displayPath)
	return nil
}

func cmdDelete(templateType string) error {
	return cmdDeleteTo(os.Stdout, templateType)
}

func cmdDeleteTo(w io.Writer, templateType string) error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	manager := gitignore.NewManager(cwd)

	// Try to delete the section
	if err := manager.Delete(templateType); err != nil {
		return err
	}

	fmt.Fprintf(w, "Removed '%s' from .gitignore\n", templateType)
	return nil
}

func cmdInit(cfg *config.Config) error {
	return cmdInitTo(os.Stdout, cfg)
}

func cmdInitTo(w io.Writer, cfg *config.Config) error {
	if len(cfg.DefaultTypes) == 0 {
		fmt.Fprintln(w, "No default types configured.")
		fmt.Fprintln(w, "Add 'gitignore.default-types = github/go, github/global/macos' to your config file.")
		return nil
	}

	// Create source manager
	sm, err := source.NewSourceManager(cfg.LocalTemplatesPath, cfg.TemplateURL, cfg.EnableToptal)
	if err != nil {
		return fmt.Errorf("failed to create source manager: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	manager := gitignore.NewManager(cwd)
	addedCount := 0
	skippedCount := 0

	fmt.Fprintf(w, "Initializing .gitignore with default types: %s\n\n", strings.Join(cfg.DefaultTypes, ", "))

	for _, templateType := range cfg.DefaultTypes {
		// Check if already exists
		exists, err := manager.HasSection(templateType)
		if err != nil {
			fmt.Fprintf(w, "  Warning: could not check for '%s': %v\n", templateType, err)
			continue
		}
		if exists {
			fmt.Fprintf(w, "  Skipping '%s' (already exists)\n", templateType)
			skippedCount++
			continue
		}

		// GetAny handles source prefixes automatically (e.g., "github/rust" vs "rust")
		file, content, err := sm.GetAny(templateType)
		if err != nil {
			fmt.Fprintf(w, "  Warning: template '%s' not found\n", templateType)
			continue
		}

		// Create section name (include category if present)
		sectionName := file.Name
		if file.Category != "" {
			sectionName = file.Category + "/" + file.Name
		}

		// Add to gitignore
		if err := manager.Add(sectionName, content); err != nil {
			fmt.Fprintf(w, "  Warning: failed to add '%s': %v\n", templateType, err)
			continue
		}

		// Build display path like list/search (lowercase source/category/name)
		var displayPath string
		if file.Category == "" {
			displayPath = fmt.Sprintf("%s/%s", strings.ToLower(file.Source), strings.ToLower(file.Name))
		} else {
			displayPath = fmt.Sprintf("%s/%s/%s", strings.ToLower(file.Source), strings.ToLower(file.Category), strings.ToLower(file.Name))
		}
		fmt.Fprintf(w, "  Added '%s'\n", displayPath)
		addedCount++
	}

	fmt.Fprintf(w, "\nDone: %d added, %d skipped\n", addedCount, skippedCount)
	return nil
}

func cmdIgnore(patterns []string) error {
	return cmdIgnoreTo(os.Stdout, patterns)
}

func cmdIgnoreTo(w io.Writer, patterns []string) error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	manager := gitignore.NewManager(cwd)
	added, skipped, err := manager.AddPatterns(patterns)
	if err != nil {
		return err
	}

	for _, pattern := range added {
		fmt.Fprintf(w, "Added '%s' to .gitignore\n", pattern)
	}
	for _, pattern := range skipped {
		fmt.Fprintf(w, "Skipped '%s' (already exists)\n", pattern)
	}

	if len(added) == 0 && len(skipped) == 0 {
		fmt.Fprintln(w, "No patterns to add")
	}

	return nil
}

func cmdRemove(patterns []string) error {
	return cmdRemoveTo(os.Stdout, patterns)
}

func cmdRemoveTo(w io.Writer, patterns []string) error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	manager := gitignore.NewManager(cwd)

	for _, pattern := range patterns {
		if err := manager.RemovePattern(pattern); err != nil {
			fmt.Fprintf(w, "Warning: %v\n", err)
			continue
		}
		fmt.Fprintf(w, "Removed '%s' from .gitignore\n", pattern)
	}

	return nil
}

// cmdServe starts an MCP server that exposes gitignore tools
func cmdServe() error {
	// Load configuration once for reuse across tool calls
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create MCP server
	s := server.NewMCPServer(
		"gitignore",
		getVersion(),
		server.WithToolCapabilities(true),
	)

	// Register gitignore_list tool
	listTool := mcp.NewTool("gitignore_list",
		mcp.WithDescription("List all available gitignore templates from configured sources (local, GitHub, Toptal)"),
	)
	s.AddTool(listTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var buf bytes.Buffer
		if err := cmdListTo(&buf, cfg, ""); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(buf.String()), nil
	})

	// Register gitignore_search tool
	searchTool := mcp.NewTool("gitignore_search",
		mcp.WithDescription("Search for gitignore templates by name pattern"),
		mcp.WithString("pattern",
			mcp.Required(),
			mcp.Description("Search pattern to filter templates (case-insensitive substring match)"),
		),
	)
	s.AddTool(searchTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pattern, err := request.RequireString("pattern")
		if err != nil {
			return mcp.NewToolResultError("pattern parameter is required"), nil
		}
		var buf bytes.Buffer
		if err := cmdListTo(&buf, cfg, pattern); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(buf.String()), nil
	})

	// Register gitignore_add tool
	addTool := mcp.NewTool("gitignore_add",
		mcp.WithDescription("Add a gitignore template to .gitignore file in the current directory"),
		mcp.WithString("type",
			mcp.Required(),
			mcp.Description("Template type to add (e.g., 'go', 'github/rust', 'toptal/python')"),
		),
	)
	s.AddTool(addTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		templateType, err := request.RequireString("type")
		if err != nil {
			return mcp.NewToolResultError("type parameter is required"), nil
		}
		var buf bytes.Buffer
		if err := cmdAddTo(&buf, cfg, templateType); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(buf.String()), nil
	})

	// Register gitignore_delete tool
	deleteTool := mcp.NewTool("gitignore_delete",
		mcp.WithDescription("Remove a gitignore template section from .gitignore file"),
		mcp.WithString("type",
			mcp.Required(),
			mcp.Description("Template type/section name to remove from .gitignore"),
		),
	)
	s.AddTool(deleteTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		templateType, err := request.RequireString("type")
		if err != nil {
			return mcp.NewToolResultError("type parameter is required"), nil
		}
		var buf bytes.Buffer
		if err := cmdDeleteTo(&buf, templateType); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(buf.String()), nil
	})

	// Register gitignore_ignore tool
	ignoreTool := mcp.NewTool("gitignore_ignore",
		mcp.WithDescription("Add one or more patterns directly to .gitignore file"),
		mcp.WithArray("patterns",
			mcp.WithStringItems(),
			mcp.Required(),
			mcp.Description("Array of patterns to add to .gitignore (e.g., ['node_modules', '*.log', 'dist/'])"),
		),
	)
	s.AddTool(ignoreTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		patternsRaw, ok := args["patterns"].([]interface{})
		if !ok || len(patternsRaw) == 0 {
			return mcp.NewToolResultError("patterns parameter is required and must be a non-empty array"), nil
		}
		patterns := make([]string, 0, len(patternsRaw))
		for _, p := range patternsRaw {
			if ps, ok := p.(string); ok {
				patterns = append(patterns, ps)
			}
		}
		if len(patterns) == 0 {
			return mcp.NewToolResultError("patterns must contain at least one string"), nil
		}
		var buf bytes.Buffer
		if err := cmdIgnoreTo(&buf, patterns); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(buf.String()), nil
	})

	// Register gitignore_remove tool
	removeTool := mcp.NewTool("gitignore_remove",
		mcp.WithDescription("Remove one or more patterns from .gitignore file"),
		mcp.WithArray("patterns",
			mcp.WithStringItems(),
			mcp.Required(),
			mcp.Description("Array of patterns to remove from .gitignore"),
		),
	)
	s.AddTool(removeTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		patternsRaw, ok := args["patterns"].([]interface{})
		if !ok || len(patternsRaw) == 0 {
			return mcp.NewToolResultError("patterns parameter is required and must be a non-empty array"), nil
		}
		patterns := make([]string, 0, len(patternsRaw))
		for _, p := range patternsRaw {
			if ps, ok := p.(string); ok {
				patterns = append(patterns, ps)
			}
		}
		if len(patterns) == 0 {
			return mcp.NewToolResultError("patterns must contain at least one string"), nil
		}
		var buf bytes.Buffer
		if err := cmdRemoveTo(&buf, patterns); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(buf.String()), nil
	})

	// Register gitignore_init tool
	initTool := mcp.NewTool("gitignore_init",
		mcp.WithDescription("Initialize .gitignore with configured default template types"),
	)
	s.AddTool(initTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var buf bytes.Buffer
		if err := cmdInitTo(&buf, cfg); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(buf.String()), nil
	})

	// Run the server using stdio transport
	return server.ServeStdio(s)
}

func printUsage() {
	usage := `gitignore - Manage .gitignore templates from multiple sources

Usage:
  gitignore list                List all available templates
  gitignore search <pattern>    Search templates by name
  gitignore add <type>          Add a gitignore template to .gitignore
  gitignore delete <type>       Remove a gitignore template from .gitignore
  gitignore ignore <pattern>    Add a path/pattern directly to .gitignore
  gitignore remove <pattern>    Remove a path/pattern added via ignore
  gitignore init                Initialize .gitignore with configured default types
  gitignore serve               Start MCP server for AI assistant integration
  gitignore --help              Show this help message
  gitignore --version           Show version information

Examples:
  gitignore list                # List all available templates
  gitignore search rust         # Search for templates containing "rust"
  gitignore add Go              # Add Go template (auto-selects source by priority)
  gitignore add github/go       # Add Go template from GitHub
  gitignore add toptal/rust     # Add Rust template from Toptal
  gitignore add local/myproject # Add custom template from local directory
  gitignore delete Go           # Remove Go template
  gitignore ignore /dist/       # Add /dist/ pattern to .gitignore
  gitignore ignore node_modules # Add node_modules to .gitignore
  gitignore ignore *.log tmp/   # Add multiple patterns at once
  gitignore remove /dist/       # Remove /dist/ pattern from .gitignore
  gitignore remove node_modules # Remove node_modules from .gitignore
  gitignore init                # Add all default types from config
  gitignore serve               # Start MCP server (for AI assistants)

Template Sources (in priority order):
  1. Local: Configurable path (default: ~/.config/gitignore/templates/)
     - Custom templates that override remote sources
     - Create your own templates here
  2. GitHub: Repository configured in gitignorerc
  3. Toptal: API fallback (if enable.toptal.gitignore = true)

Configuration:
  Create ~/.config/gitignore/gitignorerc or ~/.gitignorerc with:

    # GitHub repository URL for templates
    gitignore.template.url = https://github.com/github/gitignore

    # Enable Toptal API as fallback source
    enable.toptal.gitignore = true

    # Path to local templates directory
    gitignore.local-templates-path = ~/.config/gitignore/templates

    # Default types for 'init' command
    gitignore.default-types = github/go, github/global/macos, github/global/visualstudiocode

  The ~/.gitignorerc file takes precedence if both exist.

Local Templates:
  Place custom templates in your local templates directory (default: ~/.config/gitignore/templates/)
  Name files as <type>.gitignore (e.g., myproject.gitignore)
  Local templates always take precedence over remote sources.

Default source: https://github.com/github/gitignore
`
	fmt.Print(usage)
}
