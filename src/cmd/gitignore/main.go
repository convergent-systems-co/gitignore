package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/polliard/gitignore/src/pkg/config"
	"github.com/polliard/gitignore/src/pkg/github"
	"github.com/polliard/gitignore/src/pkg/gitignore"
)

const (
	version = "1.0.0"
)

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
		return cmdList(cfg)
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
		fmt.Printf("gitignore version %s\n", version)
		return nil
	default:
		return fmt.Errorf("unknown command: %s\nRun 'gitignore --help' for usage", cmd)
	}
}

func cmdList(cfg *config.Config) error {
	fmt.Printf("Fetching gitignore templates from: %s\n\n", cfg.TemplateURL)

	client, err := github.NewClient(cfg.TemplateURL)
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	files, err := client.ListGitignoreFiles()
	if err != nil {
		return fmt.Errorf("failed to list gitignore files: %w", err)
	}

	// Group files by category
	categories := make(map[string][]github.GitignoreFile)
	for _, file := range files {
		cat := file.Category
		if cat == "" {
			cat = "(root)"
		}
		categories[cat] = append(categories[cat], file)
	}

	// Sort categories
	var catNames []string
	for cat := range categories {
		catNames = append(catNames, cat)
	}
	sort.Strings(catNames)

	// Print files grouped by category
	totalCount := 0
	for _, cat := range catNames {
		files := categories[cat]
		sort.Slice(files, func(i, j int) bool {
			return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
		})

		if cat == "(root)" {
			fmt.Println("Available templates:")
		} else {
			fmt.Printf("\n%s:\n", cat)
		}

		for _, file := range files {
			fmt.Printf("  %s\n", file.Name)
			totalCount++
		}
	}

	fmt.Printf("\nTotal: %d templates available\n", totalCount)
	return nil
}

func cmdAdd(cfg *config.Config, templateType string) error {
	client, err := github.NewClient(cfg.TemplateURL)
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Find the gitignore file
	file, err := client.FindGitignoreFile(templateType)
	if err != nil {
		return err
	}

	// Get the content
	content, err := client.GetGitignoreContent(*file)
	if err != nil {
		return fmt.Errorf("failed to fetch template content: %w", err)
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

	fmt.Printf("Added '%s' to .gitignore\n", sectionName)
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

	client, err := github.NewClient(cfg.TemplateURL)
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
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

		// Find the gitignore file
		file, err := client.FindGitignoreFile(templateType)
		if err != nil {
			fmt.Printf("  Warning: template '%s' not found\n", templateType)
			continue
		}

		// Get the content
		content, err := client.GetGitignoreContent(*file)
		if err != nil {
			fmt.Printf("  Warning: failed to fetch '%s': %v\n", templateType, err)
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

		fmt.Printf("  Added '%s'\n", sectionName)
		addedCount++
	}

	fmt.Printf("\nDone: %d added, %d skipped\n", addedCount, skippedCount)
	return nil
}

func printUsage() {
	usage := `gitignore - Manage .gitignore templates from GitHub

Usage:
  gitignore --list              List all available gitignore templates
  gitignore add <type>          Add a gitignore template to .gitignore
  gitignore delete <type>       Remove a gitignore template from .gitignore
  gitignore init                Initialize .gitignore with configured default types
  gitignore --help              Show this help message
  gitignore --version           Show version information

Examples:
  gitignore --list              # List all available templates
  gitignore add Go              # Add Go template
  gitignore add Global/macOS    # Add macOS global template
  gitignore delete Go           # Remove Go template
  gitignore init                # Add all default types from config

Configuration:
  Create ~/.config/gitignore/gitignorerc or ~/.gitignorerc with:
    gitignore.template.url = https://github.com/github/gitignore
    gitignore.default-types = Go, Global/macOS, Global/VisualStudioCode

  The ~/.gitignorerc file takes precedence if both exist.

Default source: https://github.com/github/gitignore
`
	fmt.Print(usage)
}
