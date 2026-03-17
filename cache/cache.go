// Package cache provides translation progress caching for resume capability.
package cache

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
)

const cacheRoot = ".babelbook_cache"

// Dir returns the cache directory for a given input file.
func Dir(inputFile string) string {
	hash := fmt.Sprintf("%x", md5.Sum([]byte(inputFile)))
	return filepath.Join(cacheRoot, hash[:12])
}

// SaveChunk writes a translated chunk to the cache.
func SaveChunk(cacheDir, fileName string, chunkIdx int, html string) error {
	dir := filepath.Join(cacheDir, sanitize(fileName))
	os.MkdirAll(dir, 0o755)
	path := filepath.Join(dir, fmt.Sprintf("chunk_%04d.html", chunkIdx))
	return os.WriteFile(path, []byte(html), 0o644)
}

// LoadChunk reads a cached chunk. Returns empty string if not found.
func LoadChunk(cacheDir, fileName string, chunkIdx int) string {
	path := filepath.Join(cacheDir, sanitize(fileName), fmt.Sprintf("chunk_%04d.html", chunkIdx))
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// Clean removes the cache directory after successful completion.
func Clean(cacheDir string) {
	os.RemoveAll(cacheDir)
}

// sanitize replaces path-unsafe characters in filenames.
func sanitize(name string) string {
	safe := ""
	for _, ch := range name {
		switch {
		case ch >= 'a' && ch <= 'z', ch >= 'A' && ch <= 'Z', ch >= '0' && ch <= '9', ch == '-', ch == '_', ch == '.':
			safe += string(ch)
		default:
			safe += "_"
		}
	}
	return safe
}
