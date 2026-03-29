package main

import (
	"encoding/gob"
	"os"
	"path/filepath"
)

type CacheEntry struct {
	Chunks []Chunk
}

func getCacheDir() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	path := filepath.Join(dir, "rag-cli-cache")
	os.MkdirAll(path, 0755)
	return path
}

func getCachePath(contentHash string) string {
	return filepath.Join(getCacheDir(), contentHash+".gob")
}

func loadCache(contentHash string) ([]Chunk, bool) {
	path := getCachePath(contentHash)
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer f.Close()

	var entry CacheEntry
	if err := gob.NewDecoder(f).Decode(&entry); err != nil {
		return nil, false
	}

	return entry.Chunks, true
}

func saveCache(contentHash string, chunks []Chunk) error {
	path := getCachePath(contentHash)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	entry := CacheEntry{
		Chunks: chunks,
	}
	return gob.NewEncoder(f).Encode(entry)
}

func clearCacheDir() {
	os.RemoveAll(getCacheDir())
}
