package main

import (
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"os"
	"path/filepath"
)

type CacheEntry struct {
	ModTime int64
	Chunks  []Chunk
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

func getCachePath(source string) string {
	hash := sha256.Sum256([]byte(source))
	filename := hex.EncodeToString(hash[:]) + ".gob"
	return filepath.Join(getCacheDir(), filename)
}

func loadCache(source string, currentMTime int64) ([]Chunk, bool) {
	path := getCachePath(source)
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer f.Close()

	var entry CacheEntry
	if err := gob.NewDecoder(f).Decode(&entry); err != nil {
		return nil, false
	}

	if entry.ModTime != currentMTime {
		return nil, false
	}

	return entry.Chunks, true
}

func saveCache(source string, mtime int64, chunks []Chunk) error {
	path := getCachePath(source)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	entry := CacheEntry{
		ModTime: mtime,
		Chunks:  chunks,
	}
	return gob.NewEncoder(f).Encode(entry)
}

func clearCacheDir() {
	os.RemoveAll(getCacheDir())
}
