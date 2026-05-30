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

	// ToptalTemplateURL is the Toptal gitignore API URL
	ToptalTemplateURL = "https://www.toptal.com/developers/gitignore/api"

	// ConfigFileName is the name of the config file
	ConfigFileName = "gitignorerc"
)

// Config holds the application configuration
type Config struct {
	TemplateURL        string   // GitHub repository URL for templates
	EnableToptal       bool     // Enable Toptal gitignore API as fallback source
	LocalTemplatesPath string   // Path to local templates directory
	DefaultTypes       []string // Default types for init command
}

// DefaultLocalTemplatesPath returns the default local templates path
func DefaultLocalTemplatesPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "gitignore", "templates")
}

// DefaultConfig returns a config with default values
func DefaultConfig() *Config {
	return &Config{
		TemplateURL:        DefaultTemplateURL,
		EnableToptal:       false,
		LocalTemplatesPath: DefaultLocalTemplatesPath(),
		DefaultTypes:       []string{},
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
		case "enable.toptal.gitignore":
			c.EnableToptal = parseBool(value)
		case "gitignore.local-templates-path":
			// Expand ~ to home directory
			if strings.HasPrefix(value, "~/") {
				home, err := os.UserHomeDir()
				if err == nil {
					value = filepath.Join(home, value[2:])
				}
			}
			c.LocalTemplatesPath = value
		case "gitignore.default-types":
			c.DefaultTypes = parseTypesList(value)
		}
	}

	return scanner.Err()
}

// parseBool parses a boolean value from string
func parseBool(value string) bool {
	v := strings.ToLower(value)
	return v == "true" || v == "yes" || v == "1" || v == "on"
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

// CanonicalConfigPath returns the preferred location for a freshly scaffolded
// config file: ~/.config/gitignore/gitignorerc. This is the first entry in
// GetConfigPaths and follows the XDG-style layout used elsewhere in the tool.
func CanonicalConfigPath() (string, error) {
	paths, err := GetConfigPaths()
	if err != nil {
		return "", err
	}
	if len(paths) == 0 {
		return "", fmt.Errorf("no config paths available")
	}
	return paths[0], nil
}

// starterConfigContents is the body written by ScaffoldStarterConfig. It is a
// commented template so a new user can see every supported key, with a sensible
// gitignore.default-types line active out of the box.
const starterConfigContents = `# gitignore configuration
# See: https://github.com/convergent-systems-co/gitignore

# GitHub repository URL for templates
gitignore.template.url = ` + DefaultTemplateURL + `

# Enable Toptal API as a fallback source
# enable.toptal.gitignore = false

# Path to local templates directory
# gitignore.local-templates-path = ~/.config/gitignore/templates

# Default types added by 'gitignore init'
gitignore.default-types = github/go, github/global/macos, github/global/visualstudiocode
`

// ScaffoldStarterConfig writes a starter config to path, creating any missing
// parent directories. It never overwrites an existing file: if path already
// exists it returns (false, nil) so the caller can tell the user the path was
// left untouched. On a successful write it returns (true, nil).
func ScaffoldStarterConfig(path string) (written bool, err error) {
	if _, statErr := os.Stat(path); statErr == nil {
		return false, nil // Already present; do not clobber the user's config.
	} else if !os.IsNotExist(statErr) {
		return false, fmt.Errorf("checking config path %s: %w", path, statErr)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, fmt.Errorf("creating config directory for %s: %w", path, err)
	}

	if err := os.WriteFile(path, []byte(starterConfigContents), 0o644); err != nil {
		return false, fmt.Errorf("writing config to %s: %w", path, err)
	}

	return true, nil
}
