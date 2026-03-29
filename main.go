package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	topK       int
	chunkSize  int
	overlap    int
	clearCache bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "rag <search_term> <file_or_glob_or_url...>",
		Short: "A local, completely private Semantic Search CLI.",
		Long: `Local Semantic Search CLI

Search over your documents and web pages using local vector embeddings.
Understand the *meaning* of your query rather than relying on exact keyword matches.`,
		Run: runSearch,
	}

	rootCmd.Flags().IntVarP(&topK, "topK", "k", 3, "Number of top results to return")
	rootCmd.Flags().IntVar(&chunkSize, "chunk", 800, "Characters per text chunk")
	rootCmd.Flags().IntVar(&overlap, "overlap", 100, "Overlapping characters between chunks")
	rootCmd.Flags().BoolVar(&clearCache, "clear-cache", false, "Clear the local embedding cache")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runSearch(cmd *cobra.Command, args []string) {
	if clearCache {
		clearCacheDir()
		color.Green("✅ Cache cleared successfully.")
		if len(args) == 0 {
			return
		}
	}

	if len(args) < 2 {
		cmd.Help()
		os.Exit(1)
	}

	query := args[0]
	sources := args[1:]

	cyan := color.New(color.FgCyan, color.Bold)
	yellow := color.New(color.FgYellow)
	magenta := color.New(color.FgMagenta, color.Bold)
	green := color.New(color.FgGreen, color.Bold)

	fmt.Println()
	cyan.Println("🧠 Local Semantic Search")
	fmt.Println(strings.Repeat("-", 40))

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Color("cyan")

	s.Suffix = " Initializing local AI model (downloading to ~/.cybertron if needed)..."
	s.Start()

	engine, err := NewEngine()
	if err != nil {
		s.Stop()
		color.Red("Failed to init RAG engine: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	s.Suffix = " Ingesting & embedding documents..."

	err = engine.Ingest(ctx, sources, chunkSize, overlap, func(msg string) {
		s.Suffix = fmt.Sprintf(" %s", msg)
	})

	if err != nil {
		s.Stop()
		color.Red("\nError ingesting documents: %v\n", err)
		os.Exit(1)
	}

	s.Suffix = fmt.Sprintf(" Searching for: \"%s\"...", query)
	results, err := engine.Search(ctx, query, topK)
	s.Stop()

	if err != nil {
		color.Red("\nSearch error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	cyan.Printf("🎯 Top %d Results for \"%s\"\n", len(results), query)
	fmt.Println(strings.Repeat("=", 60))

	for i, res := range results {
		header := magenta.Sprintf("Result %d", i+1)
		score := green.Sprintf("[Score: %.4f]", res.Score)
		sourceInfo := yellow.Sprintf("File/URL: %s", res.Chunk.Source)

		fmt.Printf("%s %s\n%s\n\n", header, score, sourceInfo)

		lines := strings.Split(strings.TrimSpace(res.Chunk.Text), "\n")
		for _, line := range lines {
			fmt.Printf("  │ %s\n", line)
		}

		fmt.Println(strings.Repeat("-", 60))
	}
}
