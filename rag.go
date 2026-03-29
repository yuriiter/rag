package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/nlpodyssey/cybertron/pkg/models/bert"
	"github.com/nlpodyssey/cybertron/pkg/tasks"
	"github.com/nlpodyssey/cybertron/pkg/tasks/textencoding"
	"github.com/rs/zerolog"
)

type Chunk struct {
	Text   string
	Source string
	Vector []float32
}

type ScoredChunk struct {
	Chunk Chunk
	Score float64
}

type Engine struct {
	model  textencoding.Interface
	mu     sync.Mutex
	Chunks []Chunk
}

func NewEngine() (*Engine, error) {
	zerolog.SetGlobalLevel(zerolog.FatalLevel)

	model, err := tasks.Load[textencoding.Interface](&tasks.Config{
		ModelsDir: filepath.Join(os.Getenv("HOME"), ".cybertron"),
		ModelName: "sentence-transformers/all-MiniLM-L6-v2",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load local model: %w", err)
	}

	return &Engine{
		model:  model,
		Chunks: make([]Chunk, 0),
	}, nil
}

func (e *Engine) embed(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	numWorkers := runtime.NumCPU()
	if len(texts) < numWorkers {
		numWorkers = len(texts)
	}

	type job struct {
		index int
		text  string
	}

	jobs := make(chan job, len(texts))
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				output, err := e.model.Encode(ctx, j.text, int(bert.MeanPooling))
				if err != nil {
					if len(j.text) > 512 {
						output, err = e.model.Encode(ctx, j.text[:512], int(bert.MeanPooling))
					}
					if err != nil {
						continue
					}
				}

				e.mu.Lock()
				results[j.index] = output.Vector.Data().F32()
				e.mu.Unlock()
			}
		}()
	}

	for i, text := range texts {
		jobs <- job{index: i, text: text}
	}
	close(jobs)
	wg.Wait()

	return results, nil
}

func (e *Engine) Ingest(ctx context.Context, sources []string, chunkSize, overlap int, progress func(string)) error {
	var targets []string

	for _, src := range sources {
		if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
			targets = append(targets, src)
		} else {
			targets = append(targets, FindFiles(src)...)
		}
	}

	if len(targets) == 0 {
		return fmt.Errorf("no valid files or URLs found")
	}

	var textsToEmbed []string
	var mapIndexToMeta []struct{ Text, Source string }
	sourceMTime := make(map[string]int64)

	for i, target := range targets {
		var mtime int64
		if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
			mtime = 0
		} else {
			info, err := os.Stat(target)
			if err != nil {
				continue
			}
			mtime = info.ModTime().UnixNano()
		}
		sourceMTime[target] = mtime

		cachedChunks, ok := loadCache(target, mtime)
		if ok {
			progress(fmt.Sprintf("Loading %s from cache (%d/%d)...", filepath.Base(target), i+1, len(targets)))
			e.Chunks = append(e.Chunks, cachedChunks...)
			continue
		}

		progress(fmt.Sprintf("Extracting %s (%d/%d)...", filepath.Base(target), i+1, len(targets)))
		content, err := ExtractContent(target)
		if err != nil {
			continue
		}

		content = cleanText(content)
		if content == "" {
			continue
		}

		chunks := chunkText(content, chunkSize, overlap)
		for _, c := range chunks {
			textsToEmbed = append(textsToEmbed, c)
			mapIndexToMeta = append(mapIndexToMeta, struct{ Text, Source string }{Text: c, Source: target})
		}
	}

	if len(textsToEmbed) == 0 {
		if len(e.Chunks) == 0 {
			return fmt.Errorf("no text content extracted from sources")
		}
		return nil
	}

	batchSize := 100
	newChunksBySource := make(map[string][]Chunk)

	for i := 0; i < len(textsToEmbed); i += batchSize {
		end := i + batchSize
		if end > len(textsToEmbed) {
			end = len(textsToEmbed)
		}

		progress(fmt.Sprintf("Running AI model on new chunks %d/%d...", end, len(textsToEmbed)))

		batch := textsToEmbed[i:end]
		vectors, err := e.embed(ctx, batch)
		if err != nil {
			return fmt.Errorf("embedding error: %w", err)
		}

		for j, vec := range vectors {
			if len(vec) == 0 {
				continue
			}
			meta := mapIndexToMeta[i+j]
			newChunk := Chunk{
				Text:   meta.Text,
				Source: meta.Source,
				Vector: vec,
			}
			e.Chunks = append(e.Chunks, newChunk)
			newChunksBySource[meta.Source] = append(newChunksBySource[meta.Source], newChunk)
		}
	}

	progress("Saving vectors to local cache...")
	for src, chunks := range newChunksBySource {
		saveCache(src, sourceMTime[src], chunks)
	}

	return nil
}

func (e *Engine) Search(ctx context.Context, query string, topK int) ([]ScoredChunk, error) {
	vectors, err := e.embed(ctx, []string{query})
	if err != nil || len(vectors) == 0 || len(vectors[0]) == 0 {
		return nil, fmt.Errorf("failed to embed query: %v", err)
	}

	queryVector := vectors[0]
	var scores []ScoredChunk

	for _, chunk := range e.Chunks {
		score := cosineSimilarity(queryVector, chunk.Vector)
		scores = append(scores, ScoredChunk{Chunk: chunk, Score: score})
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	if len(scores) < topK {
		topK = len(scores)
	}

	return scores[:topK], nil
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func chunkText(text string, chunkSize, overlap int) []string {
	var chunks []string
	runes := []rune(text)
	for i := 0; i < len(runes); i += (chunkSize - overlap) {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))
		if end == len(runes) {
			break
		}
	}
	return chunks
}

func cleanText(s string) string {
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = regexp.MustCompile(`\n{2,}`).ReplaceAllString(s, "\n")
	s = regexp.MustCompile(` {2,}`).ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
