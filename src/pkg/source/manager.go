// Package source provides abstraction for different gitignore template sources
package source

import (
	"fmt"
	"strings"
)

// SourceManager manages multiple template sources with priority ordering.
// The first source in the priority list wins for List/Get fallback.
type SourceManager struct {
	sources []Source
}

// NewSourceManager creates a manager with the standard source set:
// local first, then GitHub, then Toptal (if enabled). The factories come
// from the package-level registry, so additional sources registered via
// RegisterSource will be eligible if their names are added to this list.
func NewSourceManager(localPath, templateURL string, enableToptal bool) (*SourceManager, error) {
	names := []string{NameLocal, NameGitHub}
	if enableToptal {
		names = append(names, NameToptal)
	}
	return newSourceManagerFromNames(names, localPath, templateURL)
}

func newSourceManagerFromNames(names []string, localPath, templateURL string) (*SourceManager, error) {
	sm := &SourceManager{sources: make([]Source, 0, len(names))}
	for _, name := range names {
		s, err := buildSource(name, localPath, templateURL)
		if err != nil {
			return nil, fmt.Errorf("failed to create %s source: %w", name, err)
		}
		sm.sources = append(sm.sources, s)
	}
	return sm, nil
}

// List returns all templates from all sources, in priority order.
// Templates already provided by an earlier source are deduplicated by
// case-insensitive name from later sources.
func (sm *SourceManager) List() ([]TemplateFile, error) {
	var allFiles []TemplateFile
	seen := make(map[string]bool)

	for i, s := range sm.sources {
		files, err := s.List()
		if err != nil {
			if i == 0 {
				return nil, fmt.Errorf("failed to list templates from %s: %w", s.Name(), err)
			}
			continue
		}
		for _, f := range files {
			key := strings.ToLower(f.Name)
			if !seen[key] {
				allFiles = append(allFiles, f)
				seen[key] = true
			}
		}
	}

	return allFiles, nil
}

// SourceListResult is one source's contribution to ListBySource.
type SourceListResult struct {
	Name        string
	Description string // human-readable context (path or URL) for diagnostics
	Files       []TemplateFile
	Error       error
}

// ListBySource returns per-source results in priority order. Sources that
// failed are included with their error so callers can surface diagnostics
// without losing the rest of the data.
func (sm *SourceManager) ListBySource() []SourceListResult {
	results := make([]SourceListResult, 0, len(sm.sources))
	for _, s := range sm.sources {
		r := SourceListResult{Name: s.Name(), Description: s.Describe()}
		files, err := s.List()
		if err != nil {
			r.Error = err
		} else {
			r.Files = files
		}
		results = append(results, r)
	}
	return results
}

// Get retrieves a template by name, walking sources in priority order.
func (sm *SourceManager) Get(name string) (*TemplateFile, string, error) {
	var lastErr error
	for _, s := range sm.sources {
		file, content, err := s.Get(name)
		if err == nil {
			return file, content, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, "", fmt.Errorf("template '%s' not found in any source", name)
	}
	return nil, "", fmt.Errorf("template '%s' not found", name)
}

// GetFromSource retrieves a template from a specific source.
func (sm *SourceManager) GetFromSource(sourceName, templateName string) (*TemplateFile, string, error) {
	for _, s := range sm.sources {
		if s.Name() == sourceName {
			return s.Get(templateName)
		}
	}
	return nil, "", fmt.Errorf("unknown source: %s", sourceName)
}

// SourceNames returns the names of all configured sources in priority order.
func (sm *SourceManager) SourceNames() []string {
	names := make([]string, len(sm.sources))
	for i, s := range sm.sources {
		names[i] = s.Name()
	}
	return names
}

// ParseSourcePrefix checks if the template type starts with a known source name.
// Returns (sourceName, templateName, hasPrefix).
// e.g., "github/global/macos" -> ("github", "global/macos", true)
// e.g., "go" -> ("", "go", false)
func (sm *SourceManager) ParseSourcePrefix(templateType string) (string, string, bool) {
	parts := strings.SplitN(templateType, "/", 2)
	if len(parts) < 2 {
		return "", templateType, false
	}
	potentialSource := parts[0]
	for _, s := range sm.sources {
		if s.Name() == potentialSource {
			return potentialSource, parts[1], true
		}
	}
	return "", templateType, false
}

// GetAny retrieves a template, handling source prefixes automatically.
// If templateType has a source prefix (e.g., "github/rust"), fetches from
// that source. Otherwise uses priority order.
func (sm *SourceManager) GetAny(templateType string) (*TemplateFile, string, error) {
	sourceName, templateName, hasPrefix := sm.ParseSourcePrefix(templateType)
	if hasPrefix {
		return sm.GetFromSource(sourceName, templateName)
	}
	return sm.Get(templateType)
}

// Find finds a template by name, walking sources in priority order.
func (sm *SourceManager) Find(name string) (*TemplateFile, error) {
	for _, s := range sm.sources {
		file, err := s.Find(name)
		if err == nil {
			return file, nil
		}
	}
	return nil, fmt.Errorf("template '%s' not found in any source", name)
}
