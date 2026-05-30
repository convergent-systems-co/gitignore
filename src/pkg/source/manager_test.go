package source

import (
	"errors"
	"testing"
)

// mockSource is a test source that can be configured to fail
type mockSource struct {
	name    string
	files   []TemplateFile
	content map[string]string
	listErr error
	getErr  error
	findErr error
}

func (m *mockSource) Name() string { return m.name }

func (m *mockSource) List() ([]TemplateFile, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.files, nil
}

func (m *mockSource) Get(name string) (*TemplateFile, string, error) {
	if m.getErr != nil {
		return nil, "", m.getErr
	}
	for _, f := range m.files {
		if f.Name == name {
			return &f, m.content[name], nil
		}
	}
	return nil, "", errors.New("not found")
}

func (m *mockSource) Find(query string) (*TemplateFile, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	for _, f := range m.files {
		if f.Name == query {
			return &f, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockSource) Describe() string { return "mock:" + m.name }

func TestListBySource_GracefulDegradation(t *testing.T) {
	// This test ensures that when one source fails, we still get results from others
	// This was a bug where ListBySource would return an error if any source failed,
	// even though other sources might be working fine.

	tests := []struct {
		name           string
		sources        []Source
		wantNumSources int
	}{
		{
			name: "all sources work",
			sources: []Source{
				&mockSource{
					name:  "source1",
					files: []TemplateFile{{Name: "Go"}},
				},
				&mockSource{
					name:  "source2",
					files: []TemplateFile{{Name: "Python"}},
				},
			},
			wantNumSources: 2,
		},
		{
			name: "one source fails - should continue with others",
			sources: []Source{
				&mockSource{
					name:    "failing-source",
					listErr: errors.New("API error 404"),
				},
				&mockSource{
					name:  "working-source",
					files: []TemplateFile{{Name: "Go"}},
				},
			},
			wantNumSources: 1, // Only the working source should have results
		},
		{
			name: "first source fails - should still get second source",
			sources: []Source{
				&mockSource{
					name:    "github",
					listErr: errors.New("network error"),
				},
				&mockSource{
					name:  "toptal",
					files: []TemplateFile{{Name: "Node"}, {Name: "Python"}},
				},
			},
			wantNumSources: 1,
		},
		{
			name: "all sources fail - should return empty, not error",
			sources: []Source{
				&mockSource{
					name:    "source1",
					listErr: errors.New("error 1"),
				},
				&mockSource{
					name:    "source2",
					listErr: errors.New("error 2"),
				},
			},
			wantNumSources: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &SourceManager{sources: tt.sources}

			results := sm.ListBySource()

			numSourcesWithResults := 0
			for _, r := range results {
				if r.Error == nil && len(r.Files) > 0 {
					numSourcesWithResults++
				}
			}

			if numSourcesWithResults != tt.wantNumSources {
				t.Errorf("got %d sources with results, want %d", numSourcesWithResults, tt.wantNumSources)
			}
		})
	}
}

// findResult locates a source's entry in a ListBySource slice by name.
func findResult(rs []SourceListResult, name string) (SourceListResult, bool) {
	for _, r := range rs {
		if r.Name == name {
			return r, true
		}
	}
	return SourceListResult{}, false
}

func TestListBySource_IncludesFailedSourcesWithError(t *testing.T) {
	// When a source fails, we should still include it in the result with the error
	// This helps the UI know that we tried that source but it failed
	sources := []Source{
		&mockSource{
			name:    "github",
			listErr: errors.New("404 not found"),
		},
		&mockSource{
			name:  "local",
			files: []TemplateFile{{Name: "Custom"}},
		},
	}

	sm := &SourceManager{sources: sources}
	results := sm.ListBySource()

	if githubResult, ok := findResult(results, "github"); !ok {
		t.Error("expected github source in result")
	} else if githubResult.Error == nil {
		t.Error("expected github source to have an error")
	}

	if localResult, ok := findResult(results, "local"); !ok {
		t.Error("expected local source in result")
	} else if localResult.Error != nil {
		t.Errorf("unexpected error for local: %v", localResult.Error)
	} else if len(localResult.Files) != 1 {
		t.Errorf("expected local source to have 1 file, got %d", len(localResult.Files))
	}
}

func TestGet_FallbackOnError(t *testing.T) {
	// Test that Get falls back to next source when one fails
	sm := &SourceManager{
		sources: []Source{
			&mockSource{
				name:   "github",
				getErr: errors.New("network error"),
			},
			&mockSource{
				name:    "toptal",
				files:   []TemplateFile{{Name: "Go"}},
				content: map[string]string{"Go": "# Go gitignore"},
			},
		},
	}

	file, content, err := sm.Get("Go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if file.Name != "Go" {
		t.Errorf("expected file name 'Go', got %q", file.Name)
	}
	if content != "# Go gitignore" {
		t.Errorf("unexpected content: %q", content)
	}
}

// resolveFixture mirrors the real github/gitignore layout closely enough to
// reproduce issue #6: some templates are reachable only via a path-substring
// match (what `search` does), not via the exact name/full-path match that
// `add` historically used.
func resolveFixture() *SourceManager {
	return &SourceManager{
		sources: []Source{
			&mockSource{
				name: "github",
				files: []TemplateFile{
					{Name: "Go", Category: "", Source: "github"},
					{Name: "Python", Category: "", Source: "github"},
					{Name: "Terraform", Category: "", Source: "github"},
					{Name: "Perl", Category: "", Source: "github"},
					{Name: "macOS", Category: "Global", Source: "github"},
					{Name: "JetBrains", Category: "Global", Source: "github"},
					{Name: "MicrosoftOffice", Category: "Global", Source: "github"},
				},
				content: map[string]string{
					"Go":              "# Go",
					"Python":          "# Python",
					"Terraform":       "# Terraform",
					"Perl":            "# Perl",
					"macOS":           "# macOS",
					"JetBrains":       "# JetBrains",
					"MicrosoftOffice": "# MS Office",
				},
			},
		},
	}
}

// searchResolves reports the single template path that `gitignore search
// <query>` would surface, or "" when search returns zero or many results. It
// is the reference the resolver must agree with (issue #6 acceptance).
func searchResolves(sm *SourceManager, query string) string {
	files, err := sm.List()
	if err != nil {
		return ""
	}
	var matched []string
	for _, f := range files {
		if MatchesQuery(TemplatePath(f), query) {
			matched = append(matched, TemplatePath(f))
		}
	}
	if len(matched) != 1 {
		return ""
	}
	return matched[0]
}

// TestResolveMatchesSearch is the issue #6 regression: every name that
// `search` resolves to exactly one template must be resolvable by `add`'s
// resolver, returning that same template. The previously-broken cases (perl,
// microsoftoffice) sit alongside the cases that already worked.
func TestResolveMatchesSearch(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantPath string
	}{
		// Cases the issue reports as already working.
		{"go top-level", "go", "github/go"},
		{"python top-level", "python", "github/python"},
		{"terraform top-level", "terraform", "github/terraform"},
		{"macos global", "macos", "github/global/macos"},
		{"jetbrains global", "jetbrains", "github/global/jetbrains"},
		// Cases the issue reports as broken under `add`.
		{"perl top-level", "perl", "github/perl"},
		{"microsoftoffice global", "microsoftoffice", "github/global/microsoftoffice"},
		// Substring-only match: query is not an exact name, so the old
		// exact-match `add` could never resolve it even though `search` did.
		{"office substring", "office", "github/global/microsoftoffice"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := resolveFixture()

			// Guard: confirm search itself resolves to the expected single path.
			if got := searchResolves(sm, tt.query); got != tt.wantPath {
				t.Fatalf("search(%q) = %q, want %q (test fixture wrong)", tt.query, got, tt.wantPath)
			}

			file, err := sm.Resolve(tt.query)
			if err != nil {
				t.Fatalf("Resolve(%q) error: %v", tt.query, err)
			}
			if got := TemplatePath(*file); got != tt.wantPath {
				t.Errorf("Resolve(%q) = %q, want %q (must match search)", tt.query, got, tt.wantPath)
			}
		})
	}
}

// TestGetAnyMatchesSearch exercises the same parity through the public GetAny
// path that `add` actually calls, including content retrieval.
func TestGetAnyMatchesSearch(t *testing.T) {
	for _, query := range []string{"perl", "microsoftoffice", "office", "macos", "go"} {
		sm := resolveFixture()
		want := searchResolves(sm, query)
		if want == "" {
			t.Fatalf("fixture: search(%q) did not resolve to a single template", query)
		}

		file, content, err := sm.GetAny(query)
		if err != nil {
			t.Fatalf("GetAny(%q) error: %v", query, err)
		}
		if got := TemplatePath(*file); got != want {
			t.Errorf("GetAny(%q) resolved %q, want %q", query, got, want)
		}
		if content == "" {
			t.Errorf("GetAny(%q) returned empty content", query)
		}
	}
}

func TestGetFromSource(t *testing.T) {
	// Test that GetFromSource retrieves from a specific source
	sm := &SourceManager{
		sources: []Source{
			&mockSource{
				name:    "github",
				files:   []TemplateFile{{Name: "Rust", Source: "github"}},
				content: map[string]string{"Rust": "# GitHub Rust"},
			},
			&mockSource{
				name:    "toptal",
				files:   []TemplateFile{{Name: "Rust", Source: "toptal"}},
				content: map[string]string{"Rust": "# Toptal Rust"},
			},
		},
	}

	// Get from toptal specifically
	file, content, err := sm.GetFromSource("toptal", "Rust")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "# Toptal Rust" {
		t.Errorf("expected toptal content, got: %q", content)
	}
	if file.Name != "Rust" {
		t.Errorf("expected file name 'Rust', got %q", file.Name)
	}

	// Get from github specifically
	file, content, err = sm.GetFromSource("github", "Rust")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "# GitHub Rust" {
		t.Errorf("expected github content, got: %q", content)
	}
	if file.Name != "Rust" {
		t.Errorf("expected file name 'Rust' from github, got %q", file.Name)
	}

	// Unknown source should error
	_, _, err = sm.GetFromSource("unknown", "Rust")
	if err == nil {
		t.Error("expected error for unknown source")
	}
}
