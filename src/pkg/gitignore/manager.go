// Package gitignore handles operations on local .gitignore files
package gitignore

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultFilename    = ".gitignore"
	SectionStartPrefix = "### START:"
	SectionEndPrefix   = "### END:"
)

// Manager handles gitignore file operations
type Manager struct {
	filepath string
}

// NewManager creates a new gitignore manager for the given directory
func NewManager(dir string) *Manager {
	return &Manager{
		filepath: filepath.Join(dir, DefaultFilename),
	}
}

// NewManagerWithPath creates a new gitignore manager for a specific file path
func NewManagerWithPath(path string) *Manager {
	return &Manager{filepath: path}
}

// Exists checks if the gitignore file exists
func (m *Manager) Exists() bool {
	_, err := os.Stat(m.filepath)
	return err == nil
}

// Read reads the current gitignore file content
func (m *Manager) Read() (string, error) {
	content, err := os.ReadFile(m.filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read .gitignore: %w", err)
	}
	return string(content), nil
}

// HasSection checks if a section already exists in the gitignore
func (m *Manager) HasSection(sectionName string) (bool, error) {
	content, err := m.Read()
	if err != nil {
		return false, err
	}
	startMarker := fmt.Sprintf("%s %s", SectionStartPrefix, sectionName)
	return strings.Contains(content, startMarker), nil
}

// Add adds a new section to the gitignore file
func (m *Manager) Add(sectionName, content string) error {
	exists, err := m.HasSection(sectionName)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("section '%s' already exists in .gitignore", sectionName)
	}

	currentContent, err := m.Read()
	if err != nil {
		return err
	}

	var builder strings.Builder
	if currentContent != "" {
		builder.WriteString(currentContent)
		if !strings.HasSuffix(currentContent, "\n") {
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}

	builder.WriteString(fmt.Sprintf("%s %s\n", SectionStartPrefix, sectionName))
	content = strings.TrimSpace(content)
	builder.WriteString(content)
	if !strings.HasSuffix(content, "\n") {
		builder.WriteString("\n")
	}
	builder.WriteString(fmt.Sprintf("%s %s\n", SectionEndPrefix, sectionName))

	return m.write(builder.String())
}

// Delete removes a section from the gitignore file
func (m *Manager) Delete(sectionName string) error {
	content, err := m.Read()
	if err != nil {
		return err
	}

	if content == "" {
		return fmt.Errorf("section '%s' not found in .gitignore", sectionName)
	}

	startMarker := fmt.Sprintf("%s %s", SectionStartPrefix, sectionName)
	endMarker := fmt.Sprintf("%s %s", SectionEndPrefix, sectionName)

	var result strings.Builder
	inSection := false
	foundSection := false

	scanner := bufio.NewScanner(strings.NewReader(content))
	prevLineEmpty := false

	for scanner.Scan() {
		line := scanner.Text()

		if strings.TrimSpace(line) == startMarker {
			inSection = true
			foundSection = true
			continue
		}

		if strings.TrimSpace(line) == endMarker {
			inSection = false
			continue
		}

		if !inSection {
			if strings.TrimSpace(line) == "" {
				if prevLineEmpty {
					continue
				}
				prevLineEmpty = true
			} else {
				prevLineEmpty = false
			}
			result.WriteString(line)
			result.WriteString("\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading .gitignore: %w", err)
	}

	if !foundSection {
		return fmt.Errorf("section '%s' not found in .gitignore", sectionName)
	}

	finalContent := strings.TrimRight(result.String(), "\n\t ")
	if finalContent != "" {
		finalContent += "\n"
	}

	return m.write(finalContent)
}

// ListSections returns all section names currently in the gitignore
func (m *Manager) ListSections() ([]string, error) {
	content, err := m.Read()
	if err != nil {
		return nil, err
	}

	var sections []string
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, SectionStartPrefix) {
			name := strings.TrimSpace(strings.TrimPrefix(line, SectionStartPrefix))
			sections = append(sections, name)
		}
	}

	return sections, scanner.Err()
}

func (m *Manager) write(content string) error {
	dir := filepath.Dir(m.filepath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return os.WriteFile(m.filepath, []byte(content), 0644)
}

// Path returns the gitignore file path
func (m *Manager) Path() string {
	return m.filepath
}

// IgnoredSectionPrefix is the prefix used for patterns added via ignore command
const IgnoredSectionPrefix = "ignored/"

// AddPatterns adds one or more patterns to the gitignore file, each wrapped in section markers.
// Patterns that already have a section are skipped.
func (m *Manager) AddPatterns(patterns []string) (added []string, skipped []string, err error) {
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}

		sectionName := IgnoredSectionPrefix + pattern
		exists, err := m.HasSection(sectionName)
		if err != nil {
			return added, skipped, err
		}
		if exists {
			skipped = append(skipped, pattern)
			continue
		}

		// Add the pattern as a section
		if err := m.Add(sectionName, pattern); err != nil {
			return added, skipped, err
		}
		added = append(added, pattern)
	}

	return added, skipped, nil
}

// RemovePattern removes a pattern that was added via AddPatterns (ignore command)
func (m *Manager) RemovePattern(pattern string) error {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return fmt.Errorf("pattern cannot be empty")
	}

	sectionName := IgnoredSectionPrefix + pattern
	return m.Delete(sectionName)
}
