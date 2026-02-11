package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"sort"
	"strings"

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
	case "delete", "remove", "rm":
		if len(args) < 2 {
			return fmt.Errorf("usage: gitignore delete <type>")
		}
		return cmdDelete(args[1])
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

	// Print warnings first
	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, w)
	}
	if len(warnings) > 0 {
		fmt.Fprintln(os.Stderr)
	}

	// Print paths
	if len(allPaths) == 0 {
		if searchPattern != "" {
			fmt.Printf("No templates matching '%s'\n", searchPattern)
		} else {
			fmt.Println("No templates available")
		}
		return nil
	}

	for _, path := range allPaths {
		fmt.Println(path)
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

	fmt.Printf("Added '%s' to .gitignore (from %s)\n", sectionName, formatSourceName(file.Source))
	return nil
}

func cmdDelete(templateType string) error {
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

	fmt.Printf("Removed '%s' from .gitignore\n", templateType)
	return nil
}

func cmdInit(cfg *config.Config) error {
	if len(cfg.DefaultTypes) == 0 {
		fmt.Println("No default types configured.")
		fmt.Println("Add 'gitignore.default-types = Go, Global/macOS' to your config file.")
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

	fmt.Printf("Initializing .gitignore with default types: %s\n\n", strings.Join(cfg.DefaultTypes, ", "))

	for _, templateType := range cfg.DefaultTypes {
		// Check if already exists
		exists, err := manager.HasSection(templateType)
		if err != nil {
			fmt.Printf("  Warning: could not check for '%s': %v\n", templateType, err)
			continue
		}
		if exists {
			fmt.Printf("  Skipping '%s' (already exists)\n", templateType)
			skippedCount++
			continue
		}

		// GetAny handles source prefixes automatically (e.g., "github/rust" vs "rust")
		file, content, err := sm.GetAny(templateType)
		if err != nil {
			fmt.Printf("  Warning: template '%s' not found\n", templateType)
			continue
		}

		// Create section name (include category if present)
		sectionName := file.Name
		if file.Category != "" {
			sectionName = file.Category + "/" + file.Name
		}

		// Add to gitignore
		if err := manager.Add(sectionName, content); err != nil {
			fmt.Printf("  Warning: failed to add '%s': %v\n", templateType, err)
			continue
		}

		fmt.Printf("  Added '%s' (from %s)\n", sectionName, formatSourceName(file.Source))
		addedCount++
	}

	fmt.Printf("\nDone: %d added, %d skipped\n", addedCount, skippedCount)
	return nil
}

func printUsage() {
	usage := `gitignore - Manage .gitignore templates from multiple sources

Usage:
  gitignore list                List all available templates
  gitignore search <pattern>    Search templates by name
  gitignore add <type>          Add a gitignore template to .gitignore
  gitignore delete <type>       Remove a gitignore template from .gitignore
  gitignore init                Initialize .gitignore with configured default types
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
  gitignore init                # Add all default types from config

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
    gitignore.default-types = Go, Global/macOS, Global/VisualStudioCode

  The ~/.gitignorerc file takes precedence if both exist.

Local Templates:
  Place custom templates in your local templates directory (default: ~/.config/gitignore/templates/)
  Name files as <type>.gitignore (e.g., myproject.gitignore)
  Local templates always take precedence over remote sources.

Default source: https://github.com/github/gitignore
`
	fmt.Print(usage)
}
