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

// TemplatePath renders the canonical lowercase "source/[category/]name" path
// for a template. This is the single string against which both `search` and
// `add` match a user query, so the two commands stay consistent (issue #6).
func TemplatePath(f TemplateFile) string {
	source := strings.ToLower(f.Source)
	name := strings.ToLower(f.Name)
	if f.Category == "" {
		return source + "/" + name
	}
	return source + "/" + strings.ToLower(f.Category) + "/" + name
}

// MatchesQuery is the shared predicate behind both `gitignore search` and the
// fuzzy fallback in `gitignore add`: a query matches when it is a
// case-insensitive substring of the template's canonical path. Keeping a
// single predicate guarantees that anything `search` surfaces, `add` can also
// resolve.
func MatchesQuery(templatePath, query string) bool {
	return strings.Contains(strings.ToLower(templatePath), strings.ToLower(query))
}

// Resolve maps a user query to a single template using the same matching that
// `gitignore search` uses for display. It first honors an exact match (by
// canonical path or bare name, case-insensitive) so unambiguous, fully
// qualified requests are stable; failing that it falls back to the search
// substring match. An ambiguous substring query (more than one match, none
// exact) is an error rather than a silent arbitrary pick.
func (sm *SourceManager) Resolve(query string) (*TemplateFile, error) {
	files, err := sm.List()
	if err != nil {
		return nil, err
	}

	queryLower := strings.ToLower(query)
	var substringMatches []TemplateFile
	for _, f := range files {
		path := TemplatePath(f)
		// Exact match on the canonical path or the bare name wins immediately.
		if path == queryLower || strings.ToLower(f.Name) == queryLower {
			match := f
			return &match, nil
		}
		if MatchesQuery(path, query) {
			substringMatches = append(substringMatches, f)
		}
	}

	switch len(substringMatches) {
	case 0:
		return nil, fmt.Errorf("template '%s' not found in any source", query)
	case 1:
		match := substringMatches[0]
		return &match, nil
	default:
		paths := make([]string, len(substringMatches))
		for i, f := range substringMatches {
			paths[i] = TemplatePath(f)
		}
		return nil, fmt.Errorf("template '%s' is ambiguous; matches: %s", query, strings.Join(paths, ", "))
	}
}

// GetAny retrieves a template, handling source prefixes automatically.
// If templateType has a source prefix (e.g., "github/rust"), fetches from
// that source. Otherwise it resolves the name with the same matcher `search`
// uses, then fetches the resolved template from its owning source so that
// every name `search` finds is also addable (issue #6).
func (sm *SourceManager) GetAny(templateType string) (*TemplateFile, string, error) {
	sourceName, templateName, hasPrefix := sm.ParseSourcePrefix(templateType)
	if hasPrefix {
		return sm.GetFromSource(sourceName, templateName)
	}

	resolved, err := sm.Resolve(templateType)
	if err != nil {
		return nil, "", err
	}
	// Fetch by the resolved template's real name from its owning source. The
	// resolved name is exact, so this is a direct hit, not another fuzzy pass.
	return sm.GetFromSource(resolved.Source, resolved.Name)
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
