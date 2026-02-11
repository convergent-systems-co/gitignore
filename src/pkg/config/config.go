// Package config handles configuration file parsing and management
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// DefaultTemplateURL is the default GitHub repository for gitignore templates
	DefaultTemplateURL = "https://github.com/github/gitignore"

	// ConfigFileName is the name of the config file
	ConfigFileName = "gitignorerc"
)

// Config holds the application configuration
type Config struct {
	TemplateURL  string
	DefaultTypes []string
}

// DefaultConfig returns a config with default values
func DefaultConfig() *Config {
	return &Config{
		TemplateURL:  DefaultTemplateURL,
		DefaultTypes: []string{},
	}
}

// Load reads configuration from config files
// It checks ~/.config/gitignore/gitignorerc first, then ~/.gitignorerc
// Later values override earlier ones
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Get home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return cfg, nil // Return default config if we can't get home dir
	}

	// Config file locations in order of precedence (later overrides earlier)
	configPaths := []string{
		filepath.Join(home, ".config", "gitignore", ConfigFileName),
		filepath.Join(home, "."+ConfigFileName),
	}

	for _, path := range configPaths {
		if err := cfg.loadFromFile(path); err != nil {
			// Ignore file not found errors
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("error reading config from %s: %w", path, err)
			}
		}
	}

	return cfg, nil
}

// LoadFromPath loads configuration from a specific file path
func LoadFromPath(path string) (*Config, error) {
	cfg := DefaultConfig()
	if err := cfg.loadFromFile(path); err != nil {
		return nil, err
	}
	return cfg, nil
}

// loadFromFile reads and parses a config file
func (c *Config) loadFromFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Parse key = value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue // Skip malformed lines
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		value = strings.Trim(value, `"'`)

		switch key {
		case "gitignore.template.url":
			c.TemplateURL = value
		case "gitignore.default-types":
			c.DefaultTypes = parseTypesList(value)
		}
	}

	return scanner.Err()
}

// parseTypesList parses a comma-separated list of types
func parseTypesList(value string) []string {
	var types []string
	for _, t := range strings.Split(value, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			types = append(types, t)
		}
	}
	return types
}

// GetConfigPaths returns the list of config file paths that would be checked
func GetConfigPaths() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	return []string{
		filepath.Join(home, ".config", "gitignore", ConfigFileName),
		filepath.Join(home, "."+ConfigFileName),
	}, nil
}
