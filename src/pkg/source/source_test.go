package source

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalSourceList(t *testing.T) {
	// Create a temporary directory with some test templates
	tmpDir := t.TempDir()

	// Create test gitignore files
	templates := map[string]string{
		"Go.gitignore":       "# Go files\n*.exe\n",
		"Python.gitignore":   "# Python files\n__pycache__/\n",
		"MyCustom.gitignore": "# Custom template\n.myfiles/\n",
	}

	for name, content := range templates {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	// Also create a non-gitignore file that should be ignored
	if err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# README"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	local := NewLocalSourceWithDir(tmpDir)

	files, err := local.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("expected 3 templates, got %d", len(files))
	}

	// Verify all templates are found
	foundNames := make(map[string]bool)
	for _, f := range files {
		foundNames[f.Name] = true
		if f.Source != "local" {
			t.Errorf("expected source 'local', got '%s'", f.Source)
		}
	}

	for name := range templates {
		expectedName := name[:len(name)-len(".gitignore")]
		if !foundNames[expectedName] {
			t.Errorf("expected to find template '%s'", expectedName)
		}
	}
}

func TestLocalSourceListEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	local := NewLocalSourceWithDir(tmpDir)

	files, err := local.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("expected 0 templates, got %d", len(files))
	}
}

func TestLocalSourceListNonExistentDir(t *testing.T) {
	local := NewLocalSourceWithDir("/nonexistent/path/that/does/not/exist")

	files, err := local.List()
	if err != nil {
		t.Fatalf("List() should return empty list for nonexistent dir, got error: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("expected 0 templates, got %d", len(files))
	}
}

func TestLocalSourceGet(t *testing.T) {
	tmpDir := t.TempDir()

	content := "# My custom ignores\n.secret/\n*.tmp\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "MyProject.gitignore"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	local := NewLocalSourceWithDir(tmpDir)

	file, gotContent, err := local.Get("MyProject")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}

	if file.Name != "MyProject" {
		t.Errorf("expected name 'MyProject', got '%s'", file.Name)
	}

	if gotContent != content {
		t.Errorf("content mismatch:\nexpected: %q\ngot: %q", content, gotContent)
	}
}

func TestLocalSourceGetCaseInsensitive(t *testing.T) {
	tmpDir := t.TempDir()

	content := "# Go ignores\n*.exe\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "Go.gitignore"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	local := NewLocalSourceWithDir(tmpDir)

	// Try different case variations
	for _, name := range []string{"go", "GO", "gO", "Go"} {
		file, _, err := local.Get(name)
		if err != nil {
			t.Errorf("Get(%q) error: %v", name, err)
			continue
		}
		if file.Name != "Go" {
			t.Errorf("Get(%q) returned name '%s', expected 'Go'", name, file.Name)
		}
	}
}

func TestLocalSourceGetNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	local := NewLocalSourceWithDir(tmpDir)

	_, _, err := local.Get("NonExistent")
	if err == nil {
		t.Error("expected error for non-existent template")
	}
}

func TestLocalSourceFind(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "Python.gitignore"), []byte("# Python"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	local := NewLocalSourceWithDir(tmpDir)

	file, err := local.Find("python") // case insensitive
	if err != nil {
		t.Fatalf("Find() error: %v", err)
	}

	if file.Name != "Python" {
		t.Errorf("expected name 'Python', got '%s'", file.Name)
	}
}

func TestLocalSourceName(t *testing.T) {
	local := NewLocalSourceWithDir("/tmp")
	if local.Name() != "local" {
		t.Errorf("expected name 'local', got '%s'", local.Name())
	}
}

func TestLocalSourceDir(t *testing.T) {
	dir := "/some/test/path"
	local := NewLocalSourceWithDir(dir)
	if local.Dir() != dir {
		t.Errorf("expected dir '%s', got '%s'", dir, local.Dir())
	}
}

func TestLocalSourceExists(t *testing.T) {
	tmpDir := t.TempDir()
	local := NewLocalSourceWithDir(tmpDir)

	if !local.Exists() {
		t.Error("expected Exists() to return true for existing directory")
	}

	local2 := NewLocalSourceWithDir("/nonexistent/path")
	if local2.Exists() {
		t.Error("expected Exists() to return false for non-existent directory")
	}
}

func TestLocalSourceEnsureDir(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "subdir", "gitignore")
	local := NewLocalSourceWithDir(newDir)

	if local.Exists() {
		t.Error("directory should not exist yet")
	}

	if err := local.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir() error: %v", err)
	}

	if !local.Exists() {
		t.Error("directory should exist after EnsureDir()")
	}
}
