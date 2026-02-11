package gitignore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewManager(t *testing.T) {
	testDir := filepath.Join("tmp", "test")
	manager := NewManager(testDir)
	expected := filepath.Join(testDir, ".gitignore")
	if manager.Path() != expected {
		t.Errorf("Path() = %v, want %v", manager.Path(), expected)
	}
}

func TestNewManagerWithPath(t *testing.T) {
	customPath := filepath.Join("custom", "path", ".gitignore")
	manager := NewManagerWithPath(customPath)
	if manager.Path() != customPath {
		t.Errorf("Path() = %v, want %v", manager.Path(), customPath)
	}
}

func TestExists(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	if manager.Exists() {
		t.Error("Exists() = true, want false for non-existent file")
	}

	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("# test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if !manager.Exists() {
		t.Error("Exists() = false, want true for existing file")
	}
}

func TestRead(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	content, err := manager.Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if content != "" {
		t.Errorf("Read() = %q, want empty string for non-existent file", content)
	}

	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	expected := "*.log\n*.tmp\n"
	if err := os.WriteFile(gitignorePath, []byte(expected), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	content, err = manager.Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if content != expected {
		t.Errorf("Read() = %q, want %q", content, expected)
	}
}

func TestAdd(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	templateContent := "*.exe\n*.dll\nbin/\n"
	if err := manager.Add("Go", templateContent); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	content, err := manager.Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if !strings.Contains(content, "### START: Go") {
		t.Error("Add() did not include start marker")
	}
	if !strings.Contains(content, "### END: Go") {
		t.Error("Add() did not include end marker")
	}
	if !strings.Contains(content, "*.exe") {
		t.Error("Add() did not include template content")
	}
}

func TestAddWithCategory(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	templateContent := ".DS_Store\n"
	if err := manager.Add("Global/macOS", templateContent); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	content, err := manager.Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if !strings.Contains(content, "### START: Global/macOS") {
		t.Error("Add() did not include category in section name")
	}
}

func TestAddMultipleSections(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	if err := manager.Add("Go", "*.exe\n"); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if err := manager.Add("Python", "__pycache__/\n"); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	content, err := manager.Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if !strings.Contains(content, "### START: Go") {
		t.Error("Missing Go section")
	}
	if !strings.Contains(content, "### START: Python") {
		t.Error("Missing Python section")
	}
}

func TestAddDuplicateSection(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	if err := manager.Add("Go", "*.exe\n"); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	err := manager.Add("Go", "*.exe\n")
	if err == nil {
		t.Error("Add() should return error for duplicate section")
	}
}

func TestAddToExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	gitignorePath := filepath.Join(tmpDir, ".gitignore")

	existing := "# My existing rules\n*.tmp\n"
	if err := os.WriteFile(gitignorePath, []byte(existing), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	manager := NewManager(tmpDir)
	if err := manager.Add("Go", "*.exe\n"); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	content, err := manager.Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if !strings.Contains(content, "# My existing rules") {
		t.Error("Add() did not preserve existing content")
	}
	if !strings.Contains(content, "*.tmp") {
		t.Error("Add() did not preserve existing rules")
	}
	if !strings.Contains(content, "### START: Go") {
		t.Error("Add() did not add new section")
	}
}

func TestDelete(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	if err := manager.Add("Go", "*.exe\nbin/\n"); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if err := manager.Delete("Go"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	content, err := manager.Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if strings.Contains(content, "### START: Go") {
		t.Error("Delete() did not remove section start marker")
	}
	if strings.Contains(content, "### END: Go") {
		t.Error("Delete() did not remove section end marker")
	}
	if strings.Contains(content, "*.exe") {
		t.Error("Delete() did not remove section content")
	}
}

func TestDeleteNonExistentSection(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	if err := manager.Add("Go", "*.exe\n"); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	err := manager.Delete("Python")
	if err == nil {
		t.Error("Delete() should return error for non-existent section")
	}
}

func TestDeletePreservesOtherSections(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	if err := manager.Add("Go", "*.exe\n"); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if err := manager.Add("Python", "__pycache__/\n"); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if err := manager.Delete("Go"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	content, err := manager.Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if strings.Contains(content, "### START: Go") {
		t.Error("Delete() did not remove Go section")
	}
	if !strings.Contains(content, "### START: Python") {
		t.Error("Delete() removed Python section")
	}
	if !strings.Contains(content, "__pycache__") {
		t.Error("Delete() removed Python content")
	}
}

func TestHasSection(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	has, err := manager.HasSection("Go")
	if err != nil {
		t.Fatalf("HasSection() error = %v", err)
	}
	if has {
		t.Error("HasSection() = true, want false for non-existent section")
	}

	if err := manager.Add("Go", "*.exe\n"); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	has, err = manager.HasSection("Go")
	if err != nil {
		t.Fatalf("HasSection() error = %v", err)
	}
	if !has {
		t.Error("HasSection() = false, want true for existing section")
	}
}

func TestListSections(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	sections, err := manager.ListSections()
	if err != nil {
		t.Fatalf("ListSections() error = %v", err)
	}
	if len(sections) != 0 {
		t.Errorf("ListSections() = %v, want empty list", sections)
	}

	if err := manager.Add("Go", "*.exe\n"); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if err := manager.Add("Python", "__pycache__/\n"); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	sections, err = manager.ListSections()
	if err != nil {
		t.Fatalf("ListSections() error = %v", err)
	}

	if len(sections) != 2 {
		t.Errorf("ListSections() returned %d sections, want 2", len(sections))
	}

	foundGo := false
	foundPython := false
	for _, s := range sections {
		if s == "Go" {
			foundGo = true
		}
		if s == "Python" {
			foundPython = true
		}
	}

	if !foundGo {
		t.Error("ListSections() missing Go section")
	}
	if !foundPython {
		t.Error("ListSections() missing Python section")
	}
}

func TestDeleteFromEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	err := manager.Delete("Go")
	if err == nil {
		t.Error("Delete() should return error for empty/non-existent file")
	}
}

func TestAddCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "nested", "path")
	manager := NewManager(nestedDir)

	if err := manager.Add("Go", "*.exe\n"); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if _, err := os.Stat(nestedDir); os.IsNotExist(err) {
		t.Error("Add() did not create nested directory")
	}

	gitignorePath := filepath.Join(nestedDir, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		t.Error("Add() did not create .gitignore file")
	}
}

func TestFullWorkflowAddAndDeleteAll(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	// Sample content for each template
	goContent := "# Go binaries\n*.exe\n*.dll\nbin/\n"
	macOSContent := "# macOS files\n.DS_Store\n.AppleDouble\n"
	pythonContent := "# Python\n__pycache__/\n*.py[cod]\n.venv/\n"

	// Step 1: Add Go, Global/macOS, Python
	if err := manager.Add("Go", goContent); err != nil {
		t.Fatalf("Add(Go) error = %v", err)
	}
	if err := manager.Add("Global/macOS", macOSContent); err != nil {
		t.Fatalf("Add(Global/macOS) error = %v", err)
	}
	if err := manager.Add("Python", pythonContent); err != nil {
		t.Fatalf("Add(Python) error = %v", err)
	}

	// Verify all three exist
	content, _ := manager.Read()
	if !strings.Contains(content, "### START: Go") {
		t.Fatal("Go section not added")
	}
	if !strings.Contains(content, "### START: Global/macOS") {
		t.Fatal("Global/macOS section not added")
	}
	if !strings.Contains(content, "### START: Python") {
		t.Fatal("Python section not added")
	}

	// Step 2: Delete Go
	if err := manager.Delete("Go"); err != nil {
		t.Fatalf("Delete(Go) error = %v", err)
	}

	// Validate Go is gone, others remain
	content, _ = manager.Read()
	if strings.Contains(content, "### START: Go") || strings.Contains(content, "### END: Go") {
		t.Error("Go section still exists after delete")
	}
	if !strings.Contains(content, "### START: Global/macOS") {
		t.Error("Global/macOS section was incorrectly removed")
	}
	if !strings.Contains(content, "### START: Python") {
		t.Error("Python section was incorrectly removed")
	}

	// Step 3: Delete Python
	if err := manager.Delete("Python"); err != nil {
		t.Fatalf("Delete(Python) error = %v", err)
	}

	// Validate Python is gone, macOS remains
	content, _ = manager.Read()
	if strings.Contains(content, "### START: Python") || strings.Contains(content, "### END: Python") {
		t.Error("Python section still exists after delete")
	}
	if !strings.Contains(content, "### START: Global/macOS") {
		t.Error("Global/macOS section was incorrectly removed")
	}

	// Step 4: Delete Global/macOS
	if err := manager.Delete("Global/macOS"); err != nil {
		t.Fatalf("Delete(Global/macOS) error = %v", err)
	}

	// Validate gitignore is empty (or only whitespace)
	content, _ = manager.Read()
	trimmed := strings.TrimSpace(content)
	if trimmed != "" {
		t.Errorf("Expected empty .gitignore after deleting all sections, got:\n%s", content)
	}
}
