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

func TestListBySource_GracefulDegradation(t *testing.T) {
	// This test ensures that when one source fails, we still get results from others
	// This was a bug where ListBySource would return an error if any source failed,
	// even though other sources might be working fine.

	tests := []struct {
		name           string
		sources        []Source
		wantNumSources int
		wantErr        bool
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
			wantErr:        false,
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
			wantErr:        false,
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
			wantErr:        false,
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
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a manager with our mock sources
			sm := &SourceManager{
				sources: tt.sources,
			}

			result, err := sm.ListBySource()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Count sources that returned files (without errors)
			numSourcesWithResults := 0
			for _, sr := range result {
				if sr.Error == nil && len(sr.Files) > 0 {
					numSourcesWithResults++
				}
			}

			if numSourcesWithResults != tt.wantNumSources {
				t.Errorf("got %d sources with results, want %d", numSourcesWithResults, tt.wantNumSources)
			}
		})
	}
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
	result, err := sm.ListBySource()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// github should be in the result with an error
	if githubResult, ok := result["github"]; !ok {
		t.Error("expected github source in result")
	} else if githubResult.Error == nil {
		t.Error("expected github source to have an error")
	}

	// local should have 1 file without error
	if localResult, ok := result["local"]; !ok {
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
		local: &LocalSource{},
		remote: []Source{
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
