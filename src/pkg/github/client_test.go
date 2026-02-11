package github

import (
	"strings"
	"testing"
)

func TestParseRepoURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "HTTPS URL",
			url:       "https://github.com/github/gitignore",
			wantOwner: "github",
			wantRepo:  "gitignore",
			wantErr:   false,
		},
		{
			name:      "HTTPS URL with .git",
			url:       "https://github.com/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "SSH URL",
			url:       "git@github.com:owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "URL with trailing slash",
			url:       "https://github.com/github/gitignore/",
			wantOwner: "github",
			wantRepo:  "gitignore",
			wantErr:   false,
		},
		{
			name:    "Invalid URL",
			url:     "not-a-valid-url",
			wantErr: true,
		},
		{
			name:    "Incomplete GitHub URL",
			url:     "https://github.com/owner",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := parseRepoURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRepoURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if owner != tt.wantOwner {
					t.Errorf("parseRepoURL() owner = %v, want %v", owner, tt.wantOwner)
				}
				if repo != tt.wantRepo {
					t.Errorf("parseRepoURL() repo = %v, want %v", repo, tt.wantRepo)
				}
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	client, err := NewClient("https://github.com/github/gitignore")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client.Owner() != "github" {
		t.Errorf("Owner() = %v, want github", client.Owner())
	}
	if client.Repo() != "gitignore" {
		t.Errorf("Repo() = %v, want gitignore", client.Repo())
	}
}

func TestNewClientInvalidURL(t *testing.T) {
	_, err := NewClient("invalid-url")
	if err == nil {
		t.Error("NewClient() expected error for invalid URL")
	}
}

func TestParseGitignorePath(t *testing.T) {
	tests := []struct {
		path         string
		wantName     string
		wantCategory string
	}{
		{
			path:         "Go.gitignore",
			wantName:     "Go",
			wantCategory: "",
		},
		{
			path:         "Global/macOS.gitignore",
			wantName:     "macOS",
			wantCategory: "Global",
		},
		{
			path:         "community/PHP/Symfony.gitignore",
			wantName:     "Symfony",
			wantCategory: "community/PHP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			file := parseGitignorePath(tt.path)
			if file.Name != tt.wantName {
				t.Errorf("Name = %v, want %v", file.Name, tt.wantName)
			}
			if file.Category != tt.wantCategory {
				t.Errorf("Category = %v, want %v", file.Category, tt.wantCategory)
			}
		})
	}
}

func TestListGitignoreFilesIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient("https://github.com/github/gitignore")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	files, err := client.ListGitignoreFiles()
	if err != nil {
		t.Fatalf("ListGitignoreFiles() error = %v", err)
	}

	if len(files) == 0 {
		t.Error("ListGitignoreFiles() returned no files")
	}

	foundGo := false
	foundPython := false
	foundNode := false

	for _, file := range files {
		switch file.Name {
		case "Go":
			foundGo = true
		case "Python":
			foundPython = true
		case "Node":
			foundNode = true
		}
	}

	if !foundGo {
		t.Error("Expected to find Go.gitignore")
	}
	if !foundPython {
		t.Error("Expected to find Python.gitignore")
	}
	if !foundNode {
		t.Error("Expected to find Node.gitignore")
	}
}

func TestFindGitignoreFileIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient("https://github.com/github/gitignore")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	file, err := client.FindGitignoreFile("go")
	if err != nil {
		t.Fatalf("FindGitignoreFile() error = %v", err)
	}

	if file.Name != "Go" {
		t.Errorf("FindGitignoreFile() Name = %v, want Go", file.Name)
	}
}

func TestGetGitignoreContentIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient("https://github.com/github/gitignore")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	file, err := client.FindGitignoreFile("Go")
	if err != nil {
		t.Fatalf("FindGitignoreFile() error = %v", err)
	}

	content, err := client.GetGitignoreContent(*file)
	if err != nil {
		t.Fatalf("GetGitignoreContent() error = %v", err)
	}

	if !strings.Contains(content, "*.exe") && !strings.Contains(content, "bin/") && !strings.Contains(content, "*.test") {
		t.Error("GetGitignoreContent() returned unexpected content for Go template")
	}
}
