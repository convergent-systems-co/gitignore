package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.TemplateURL != DefaultTemplateURL {
		t.Errorf("expected default URL %s, got %s", DefaultTemplateURL, cfg.TemplateURL)
	}
}

func TestLoadFromPath(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "testconfig")

	content := `# Test config file
gitignore.template.url = https://github.com/example/templates
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	cfg, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	expected := "https://github.com/example/templates"
	if cfg.TemplateURL != expected {
		t.Errorf("expected URL %s, got %s", expected, cfg.TemplateURL)
	}
}

func TestLoadFromPathWithQuotes(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "testconfig")

	// Test with quoted value
	content := `gitignore.template.url = "https://github.com/quoted/url"
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	cfg, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	expected := "https://github.com/quoted/url"
	if cfg.TemplateURL != expected {
		t.Errorf("expected URL %s, got %s", expected, cfg.TemplateURL)
	}
}

func TestLoadFromPathWithComments(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "testconfig")

	content := `# This is a comment
; This is also a comment
gitignore.template.url = https://github.com/test/repo

# Another comment
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	cfg, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	expected := "https://github.com/test/repo"
	if cfg.TemplateURL != expected {
		t.Errorf("expected URL %s, got %s", expected, cfg.TemplateURL)
	}
}

func TestLoadFromNonExistentPath(t *testing.T) {
	_, err := LoadFromPath("/nonexistent/path/config")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestLoadReturnsDefaultOnMissingFiles(t *testing.T) {
	// Load should return default config when no config files exist
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.TemplateURL != DefaultTemplateURL {
		t.Errorf("expected default URL when no config exists, got %s", cfg.TemplateURL)
	}
}

func TestLoadDefaultTypes(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "testconfig")

	content := `gitignore.template.url = https://github.com/github/gitignore
gitignore.default-types = Go, Global/macOS, Python
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	cfg, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	expectedTypes := []string{"Go", "Global/macOS", "Python"}
	if len(cfg.DefaultTypes) != len(expectedTypes) {
		t.Fatalf("expected %d default types, got %d", len(expectedTypes), len(cfg.DefaultTypes))
	}

	for i, expected := range expectedTypes {
		if cfg.DefaultTypes[i] != expected {
			t.Errorf("default type %d: expected %s, got %s", i, expected, cfg.DefaultTypes[i])
		}
	}
}

func TestLoadDefaultTypesWithWhitespace(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "testconfig")

	content := `gitignore.default-types =   Go ,  Global/macOS ,Python
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	cfg, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	expectedTypes := []string{"Go", "Global/macOS", "Python"}
	if len(cfg.DefaultTypes) != len(expectedTypes) {
		t.Fatalf("expected %d default types, got %d", len(expectedTypes), len(cfg.DefaultTypes))
	}

	for i, expected := range expectedTypes {
		if cfg.DefaultTypes[i] != expected {
			t.Errorf("default type %d: expected '%s', got '%s'", i, expected, cfg.DefaultTypes[i])
		}
	}
}

func TestLoadDefaultTypesEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "testconfig")

	content := `gitignore.template.url = https://github.com/github/gitignore
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	cfg, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(cfg.DefaultTypes) != 0 {
		t.Errorf("expected empty default types, got %v", cfg.DefaultTypes)
	}
}

func TestLoadEnableToptal(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"true", "true", true},
		{"True", "True", true},
		{"TRUE", "TRUE", true},
		{"yes", "yes", true},
		{"1", "1", true},
		{"on", "on", true},
		{"false", "false", false},
		{"no", "no", false},
		{"0", "0", false},
		{"off", "off", false},
		{"random", "random", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "testconfig")

			content := fmt.Sprintf("enable.toptal.gitignore = %s\n", tt.value)
			if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
				t.Fatalf("failed to create test config: %v", err)
			}

			cfg, err := LoadFromPath(configPath)
			if err != nil {
				t.Fatalf("failed to load config: %v", err)
			}

			if cfg.EnableToptal != tt.expected {
				t.Errorf("enable.toptal.gitignore = %s: expected %v, got %v", tt.value, tt.expected, cfg.EnableToptal)
			}
		})
	}
}

func TestDefaultConfigEnableToptalFalse(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.EnableToptal != false {
		t.Errorf("expected default EnableToptal to be false, got %v", cfg.EnableToptal)
	}
}

func TestLoadLocalTemplatesPath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "testconfig")

	content := `gitignore.local-templates-path = /custom/templates/path
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	cfg, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	expected := "/custom/templates/path"
	if cfg.LocalTemplatesPath != expected {
		t.Errorf("expected LocalTemplatesPath %s, got %s", expected, cfg.LocalTemplatesPath)
	}
}

func TestLoadLocalTemplatesPathWithTilde(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "testconfig")

	content := `gitignore.local-templates-path = ~/my-templates
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	cfg, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, "my-templates")
	if cfg.LocalTemplatesPath != expected {
		t.Errorf("expected LocalTemplatesPath %s, got %s", expected, cfg.LocalTemplatesPath)
	}
}

func TestDefaultLocalTemplatesPath(t *testing.T) {
	cfg := DefaultConfig()

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".config", "gitignore", "templates")
	if cfg.LocalTemplatesPath != expected {
		t.Errorf("expected default LocalTemplatesPath %s, got %s", expected, cfg.LocalTemplatesPath)
	}
}
