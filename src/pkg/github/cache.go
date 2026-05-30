package github

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// On-disk cache layout, rooted at <cacheDir>/<schema>/<owner>/<repo>/<branch>/:
//
//	list.json          -> cached template listing
//	content/<hash>     -> cached raw template body, keyed by a hash of its path
//
// Freshness is derived from the file's modification time against cacheTTL.
// A negative TTL treats every entry as stale, which the tests use to force a
// live fetch while keeping the on-disk copy available for fallback.

const (
	listCacheFilename  = "list.json"
	contentCacheSubdir = "content"
	cacheDirPerm       = 0o755
	cacheFilePerm      = 0o644
)

func (c *Client) repoCacheDir() string {
	return filepath.Join(c.cacheDir, cacheSchemaVersion, c.owner, c.repo, c.branch)
}

func (c *Client) listCachePath() string {
	return filepath.Join(c.repoCacheDir(), listCacheFilename)
}

// contentCachePath derives a filesystem-safe path for a template's content by
// hashing the template path; template paths can contain slashes and varied
// casing that are awkward as filenames.
func (c *Client) contentCachePath(templatePath string) string {
	sum := sha256.Sum256([]byte(templatePath))
	name := hex.EncodeToString(sum[:])
	return filepath.Join(c.repoCacheDir(), contentCacheSubdir, name)
}

func (c *Client) isFresh(modTime time.Time) bool {
	if c.cacheTTL < 0 {
		return false
	}
	return time.Since(modTime) < c.cacheTTL
}

// readListCache returns the cached listing only when it exists and is fresh.
func (c *Client) readListCache() ([]GitignoreFile, bool) {
	files, ok := c.loadListCache()
	if !ok {
		return nil, false
	}
	info, err := os.Stat(c.listCachePath())
	if err != nil || !c.isFresh(info.ModTime()) {
		return nil, false
	}
	return files, true
}

// readListCacheStale returns the cached listing regardless of freshness, for
// use as a degradation fallback.
func (c *Client) readListCacheStale() ([]GitignoreFile, bool) {
	return c.loadListCache()
}

func (c *Client) loadListCache() ([]GitignoreFile, bool) {
	data, err := os.ReadFile(c.listCachePath())
	if err != nil {
		return nil, false
	}
	var files []GitignoreFile
	if err := json.Unmarshal(data, &files); err != nil {
		return nil, false
	}
	return files, true
}

// writeListCache persists the listing. Cache-write failures are non-fatal: the
// caller already has the data, so we silently skip rather than fail the read.
func (c *Client) writeListCache(files []GitignoreFile) {
	data, err := json.Marshal(files)
	if err != nil {
		return
	}
	_ = c.writeCacheFile(c.listCachePath(), data)
}

// readContentCache returns cached content only when present and fresh.
func (c *Client) readContentCache(templatePath string) (string, bool) {
	path := c.contentCachePath(templatePath)
	info, err := os.Stat(path)
	if err != nil || !c.isFresh(info.ModTime()) {
		return "", false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return string(data), true
}

// readContentCacheStale returns cached content regardless of freshness.
func (c *Client) readContentCacheStale(templatePath string) (string, bool) {
	data, err := os.ReadFile(c.contentCachePath(templatePath))
	if err != nil {
		return "", false
	}
	return string(data), true
}

func (c *Client) writeContentCache(templatePath, content string) {
	_ = c.writeCacheFile(c.contentCachePath(templatePath), []byte(content))
}

func (c *Client) writeCacheFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), cacheDirPerm); err != nil {
		return err
	}
	return os.WriteFile(path, data, cacheFilePerm)
}
