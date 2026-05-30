package source

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newStubToptalServer returns an httptest server that mimics the Toptal
// gitignore API: GET /list returns a comma/newline separated catalog, and
// GET /<name> returns that template's body (404 for unknown names). It never
// touches the live network. listBody is the catalog payload; bodies maps a
// template name to the content the API should return for it.
func newStubToptalServer(t *testing.T, listBody string, bodies map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "list" {
			_, _ = w.Write([]byte(listBody))
			return
		}
		if body, ok := bodies[path]; ok {
			_, _ = w.Write([]byte(body))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// missingToolTemplates are the names called out in issue #7 that Toptal
// serves directly. pycharm/goland are handled separately via aliasing.
var missingToolTemplates = []string{
	"helm",
	"homebrew",
	"powershell",
	"react",
	"azurefunctions",
	"database",
	"nohup",
}

func TestToptalResolvesMissingToolTemplates(t *testing.T) {
	// The catalog mirrors how Toptal's /list responds: a newline-separated
	// list of available keys. Include the issue #7 names plus a couple of
	// common ones so the linear scan has to discriminate.
	catalog := strings.Join(append([]string{"go", "node", "jetbrains"}, missingToolTemplates...), "\n")

	bodies := make(map[string]string, len(missingToolTemplates))
	for _, name := range missingToolTemplates {
		bodies[name] = "# " + name + " gitignore\n"
	}

	srv := newStubToptalServer(t, catalog, bodies)
	toptal := NewToptalSourceWithURL(srv.URL)

	for _, name := range missingToolTemplates {
		t.Run(name, func(t *testing.T) {
			// Find must locate the template by case-insensitive name.
			file, err := toptal.Find(name)
			if err != nil {
				t.Fatalf("Find(%q) error: %v", name, err)
			}
			if !strings.EqualFold(file.Name, name) {
				t.Errorf("Find(%q) returned name %q", name, file.Name)
			}
			if file.Source != NameToptal {
				t.Errorf("Find(%q) source = %q, want %q", name, file.Source, NameToptal)
			}

			// Get must return the template content.
			_, content, err := toptal.Get(name)
			if err != nil {
				t.Fatalf("Get(%q) error: %v", name, err)
			}
			if !strings.Contains(content, name) {
				t.Errorf("Get(%q) content missing marker: %q", name, content)
			}
		})
	}
}

func TestToptalListIncludesMissingToolTemplates(t *testing.T) {
	catalog := strings.Join(missingToolTemplates, ",")
	srv := newStubToptalServer(t, catalog, nil)
	toptal := NewToptalSourceWithURL(srv.URL)

	files, err := toptal.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	got := make(map[string]bool, len(files))
	for _, f := range files {
		got[strings.ToLower(f.Name)] = true
		if f.Source != NameToptal {
			t.Errorf("List() entry %q has source %q, want %q", f.Name, f.Source, NameToptal)
		}
	}
	for _, name := range missingToolTemplates {
		if !got[name] {
			t.Errorf("List() missing %q", name)
		}
	}
}

func TestToptalAliasesJetBrainsIDEs(t *testing.T) {
	// pycharm and goland are JetBrains IDEs; issue #7 says they may map to the
	// JetBrains template. Toptal's catalog here exposes only "jetbrains", so
	// the alias layer must redirect the IDE-specific names to it.
	catalog := "jetbrains\ngo"
	bodies := map[string]string{"jetbrains": "# JetBrains gitignore\n.idea/\n"}
	srv := newStubToptalServer(t, catalog, bodies)
	toptal := NewToptalSourceWithURL(srv.URL)

	for _, alias := range []string{"pycharm", "goland", "PyCharm", "GoLand"} {
		t.Run(alias, func(t *testing.T) {
			file, err := toptal.Find(alias)
			if err != nil {
				t.Fatalf("Find(%q) error: %v", alias, err)
			}
			if !strings.EqualFold(file.Name, "jetbrains") {
				t.Errorf("Find(%q) resolved to %q, want jetbrains", alias, file.Name)
			}

			_, content, err := toptal.Get(alias)
			if err != nil {
				t.Fatalf("Get(%q) error: %v", alias, err)
			}
			if !strings.Contains(content, "JetBrains") {
				t.Errorf("Get(%q) content = %q, want JetBrains body", alias, content)
			}
		})
	}
}

func TestToptalDirectIDEMatchWins(t *testing.T) {
	// If Toptal's catalog does serve the IDE name directly, the direct match
	// must win over the alias so we don't lose IDE-specific rules.
	catalog := "pycharm\njetbrains"
	bodies := map[string]string{
		"pycharm":   "# PyCharm-specific gitignore\n",
		"jetbrains": "# JetBrains gitignore\n",
	}
	srv := newStubToptalServer(t, catalog, bodies)
	toptal := NewToptalSourceWithURL(srv.URL)

	_, content, err := toptal.Get("pycharm")
	if err != nil {
		t.Fatalf("Get(pycharm) error: %v", err)
	}
	if !strings.Contains(content, "PyCharm-specific") {
		t.Errorf("Get(pycharm) = %q, want direct PyCharm body", content)
	}
}
