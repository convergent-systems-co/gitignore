package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/polliard/gitignore/src/pkg/config"
)

// TestInitErrorsWhenNoDefaultTypes verifies the function backing `gitignore init`
// returns an error (so the CLI exits non-zero) when no default types are
// configured, rather than silently succeeding.
func TestInitErrorsWhenNoDefaultTypes(t *testing.T) {
	cfg := config.DefaultConfig() // DefaultTypes is empty

	var buf bytes.Buffer
	err := cmdInitTo(&buf, cfg)
	if err == nil {
		t.Fatal("expected an error when no default types are configured, got nil")
	}
}

// TestInitErrorMessageIncludesConfigPaths verifies the error names the config
// file path(s) where default types are expected, so the user knows where to look.
func TestInitErrorMessageIncludesConfigPaths(t *testing.T) {
	cfg := config.DefaultConfig()

	var buf bytes.Buffer
	err := cmdInitTo(&buf, cfg)
	if err == nil {
		t.Fatal("expected an error when no default types are configured, got nil")
	}

	paths, perr := config.GetConfigPaths()
	if perr != nil {
		t.Fatalf("GetConfigPaths returned error: %v", perr)
	}
	if len(paths) == 0 {
		t.Fatal("expected at least one config path")
	}

	msg := err.Error()
	for _, p := range paths {
		if !strings.Contains(msg, p) {
			t.Errorf("error message %q does not mention expected config path %q", msg, p)
		}
	}
}
