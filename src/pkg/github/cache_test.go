package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// treeJSON builds a minimal Trees API response containing the given blob paths.
func treeJSON(paths ...string) string {
	tr := TreeResponse{SHA: "abc", URL: "u"}
	for _, p := range paths {
		tr.Tree = append(tr.Tree, TreeItem{Path: p, Type: "blob"})
	}
	b, _ := json.Marshal(tr)
	return string(b)
}

// newTestClient wires a Client to a test server and an isolated cache dir.
func newTestClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	c, err := NewClient("https://github.com/github/gitignore",
		WithBaseURL(srv.URL),
		WithRawBaseURL(srv.URL),
		WithCacheDir(t.TempDir()),
		WithToken(""), // do not pick up ambient credentials in tests
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return c
}

// TestListServedFromCacheOnSecondCall verifies the second list does not hit HTTP.
func TestListServedFromCacheOnSecondCall(t *testing.T) {
	var treeCalls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/git/trees/") {
			atomic.AddInt32(&treeCalls, 1)
			fmt.Fprint(w, treeJSON("Go.gitignore", "Global/macOS.gitignore"))
			return
		}
		http.Error(w, "unexpected", http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)

	files, err := c.ListGitignoreFiles()
	if err != nil {
		t.Fatalf("first ListGitignoreFiles() error = %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("first list got %d files, want 2", len(files))
	}

	files2, err := c.ListGitignoreFiles()
	if err != nil {
		t.Fatalf("second ListGitignoreFiles() error = %v", err)
	}
	if len(files2) != 2 {
		t.Fatalf("second list got %d files, want 2", len(files2))
	}

	if got := atomic.LoadInt32(&treeCalls); got != 1 {
		t.Errorf("tree endpoint hit %d times, want 1 (second call should be cached)", got)
	}
}

// TestContentServedFromCacheOnSecondCall verifies content caching.
func TestContentServedFromCacheOnSecondCall(t *testing.T) {
	var contentCalls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "Go.gitignore") {
			atomic.AddInt32(&contentCalls, 1)
			fmt.Fprint(w, "*.exe\nbin/\n")
			return
		}
		http.Error(w, "unexpected", http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	file := GitignoreFile{Name: "Go", Path: "Go.gitignore"}

	first, err := c.GetGitignoreContent(file)
	if err != nil {
		t.Fatalf("first GetGitignoreContent() error = %v", err)
	}
	second, err := c.GetGitignoreContent(file)
	if err != nil {
		t.Fatalf("second GetGitignoreContent() error = %v", err)
	}
	if first != second {
		t.Errorf("cached content %q != original %q", second, first)
	}
	if got := atomic.LoadInt32(&contentCalls); got != 1 {
		t.Errorf("content endpoint hit %d times, want 1 (second call should be cached)", got)
	}
}

// TestAuthorizationHeaderSet verifies the token is sent when provided.
func TestAuthorizationHeaderSet(t *testing.T) {
	const token = "ghp_testtoken123" // #nosec G101 -- test fixture, not a real credential
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		fmt.Fprint(w, treeJSON("Go.gitignore"))
	}))
	defer srv.Close()

	c, err := NewClient("https://github.com/github/gitignore",
		WithBaseURL(srv.URL),
		WithRawBaseURL(srv.URL),
		WithCacheDir(t.TempDir()),
		WithToken(token),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if _, err := c.ListGitignoreFiles(); err != nil {
		t.Fatalf("ListGitignoreFiles() error = %v", err)
	}

	want := "Bearer " + token
	if gotAuth != want {
		t.Errorf("Authorization header = %q, want %q", gotAuth, want)
	}
}

// TestNoAuthorizationHeaderWhenTokenEmpty ensures we never send an empty Bearer.
func TestNoAuthorizationHeaderWhenTokenEmpty(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		fmt.Fprint(w, treeJSON("Go.gitignore"))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if _, err := c.ListGitignoreFiles(); err != nil {
		t.Fatalf("ListGitignoreFiles() error = %v", err)
	}
	if gotAuth != "" {
		t.Errorf("Authorization header = %q, want empty", gotAuth)
	}
}

// TestListFallsBackToCacheOn403 verifies rate-limit degradation for the list.
func TestListFallsBackToCacheOn403(t *testing.T) {
	var serve403 atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serve403.Load() {
			w.Header().Set("X-RateLimit-Remaining", "0")
			http.Error(w, `{"message":"API rate limit exceeded"}`, http.StatusForbidden)
			return
		}
		fmt.Fprint(w, treeJSON("Go.gitignore", "Python.gitignore"))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)

	// Warm the cache.
	if _, err := c.ListGitignoreFiles(); err != nil {
		t.Fatalf("warm ListGitignoreFiles() error = %v", err)
	}

	// Now force the cache to be considered stale so a live fetch is attempted,
	// then make the server return 403. The client must fall back to cache.
	c.cacheTTL = -1 // expire everything
	serve403.Store(true)

	files, err := c.ListGitignoreFiles()
	if err != nil {
		t.Fatalf("ListGitignoreFiles() after 403 error = %v (expected cache fallback)", err)
	}
	if len(files) != 2 {
		t.Errorf("fallback list got %d files, want 2", len(files))
	}
}

// TestContentFallsBackToCacheOn403 verifies rate-limit degradation for content.
func TestContentFallsBackToCacheOn403(t *testing.T) {
	var serve403 atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serve403.Load() {
			http.Error(w, "rate limited", http.StatusForbidden)
			return
		}
		fmt.Fprint(w, "*.exe\n")
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	file := GitignoreFile{Name: "Go", Path: "Go.gitignore"}

	if _, err := c.GetGitignoreContent(file); err != nil {
		t.Fatalf("warm GetGitignoreContent() error = %v", err)
	}

	c.cacheTTL = -1
	serve403.Store(true)

	content, err := c.GetGitignoreContent(file)
	if err != nil {
		t.Fatalf("GetGitignoreContent() after 403 error = %v (expected cache fallback)", err)
	}
	if content != "*.exe\n" {
		t.Errorf("fallback content = %q, want %q", content, "*.exe\n")
	}
}

// TestHardFailWhenNoCacheAnd403 verifies we error out if nothing is cached.
func TestHardFailWhenNoCacheAnd403(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusForbidden)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if _, err := c.ListGitignoreFiles(); err == nil {
		t.Error("ListGitignoreFiles() expected error when 403 and no cache, got nil")
	}
}
