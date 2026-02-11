package config

import (
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
