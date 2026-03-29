# Local Semantic Search CLI

A fast, standalone, and completely local command-line tool for performing semantic search over your documents and web pages. 

Instead of searching for exact keyword matches like `grep`, this tool uses local machine learning models (Vector Embeddings) to understand the *meaning* of your query and find relevant context across multiple file formats and URLs.

## Features

* **100% Local & Private:** Uses the local `sentence-transformers/all-MiniLM-L6-v2` model via `cybertron`. No API keys, no OpenAI, and no internet connection required (after the initial model download).
* **Multi-Format Support:** Automatically extracts and parses text from `.pdf`, `.docx`, `.xlsx`, `.epub`, `.md`, `.txt`, and various source code files.
* **Web Page Support:** Pass a URL (e.g., `https://...`) to automatically fetch, strip HTML tags, and search its contents.
* **Glob Patterns:** Easily search through entire directories using wildcards (`./docs/**/*.md`).
* **Customizable Chunking:** Control how documents are split into readable chunks for precise retrieval.

## Installation

### Prerequisites
* Go 1.21 or higher

### Install
Clone the repository and build the binary:

```bash
git clone <your-repo-url>
cd semantic-search
go install .
```
*(This will install the binary to your `$GOPATH/bin`, usually `~/go/bin`. Make sure this is in your system's `$PATH`.)*

> **Note:** On the very first run, the tool will automatically download the local embedding model to `~/.cybertron`. This might take a minute depending on your internet connection.

## Usage

**Syntax:**
```bash
rag [flags] <search_term> <file_or_glob_or_url...>
```

### Basic Search
Search a specific document for a concept:
```bash
rag "how does error handling work?" main.go
```

### Searching Multiple Files & Globs
Search across an entire directory of PDFs and Markdown files:
```bash
rag "machine learning models" "./docs/*.pdf" "./notes/**/*.md"
```

### Searching Web Pages
You can pass URLs directly. The tool will download the page, strip the HTML, and search the text:
```bash
rag "what is the main argument?" https://paulgraham.com/articles.html
```

### Mixed Sources
You can freely mix files, globs, and URLs:
```bash
rag "authentication setup" ./readme.md https://example.com/api-docs ./src/**/*.go
```

## Flags Reference

You can fine-tune the search results using flags. **Note:** Flags must come *before* the search term.

| Flag | Default | Description |
| :--- | :--- | :--- |
| `-k` | `3` | Number of top search results to return. |
| `-chunk` | `800` | The number of characters per text chunk. Smaller chunks give more precise snippets, while larger chunks give broader context. |
| `-overlap` | `100` | The number of overlapping characters between chunks. Prevents cutting off context mid-sentence. |

### Example with Flags
Retrieve the top 5 results, breaking the text into smaller 400-character chunks:
```bash
rag -k 5 -chunk 400 -overlap 50 "neural networks" ./research_papers/*.pdf
```
