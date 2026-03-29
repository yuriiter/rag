package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

func main() {
	topK := flag.Int("k", 3, "Number of top results to return")
	chunkSize := flag.Int("chunk", 800, "Chunk size for text splitting")
	overlap := flag.Int("overlap", 100, "Overlap size for text splitting")
	flag.Parse()

	args := flag.Args()
	if len(args) < 2 {
		fmt.Println("Usage: rag [flags] <search_term> <file_or_glob_or_url...>")
		fmt.Println("Example: rag -k 5 \"machine learning\" ./docs/*.pdf https://example.com/article")
		os.Exit(1)
	}

	query := args[0]
	sources := args[1:]

	fmt.Println("Initializing local embedding model (downloading if needed)...")
	engine, err := NewEngine()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init RAG engine: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	if err := engine.Ingest(ctx, sources, *chunkSize, *overlap); err != nil {
		fmt.Fprintf(os.Stderr, "Error ingesting documents: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nSearching for: \"%s\"\n", query)
	results, err := engine.Search(ctx, query, *topK)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Search error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n=== Top %d Results ===\n", *topK)
	for i, res := range results {
		fmt.Printf("\n--- Result %d (Score: %.4f) ---\n", i+1, res.Score)
		fmt.Printf("Source: %s\n", res.Chunk.Source)
		fmt.Printf("Text:\n%s\n", res.Chunk.Text)
	}
}
