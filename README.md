# Local Semantic Search CLI

A fast, standalone, and completely local command-line tool for performing semantic search over your documents and web pages. 

Instead of searching for exact keyword matches like `grep`, this tool uses local machine learning models (Vector Embeddings) to understand the *meaning* of your query and find relevant context across multiple file formats and URLs.

## Features

* **100% Local & Private:** Uses the local `sentence-transformers/all-MiniLM-L6-v2` model via `cybertron`. No API keys, no OpenAI, and no internet connection required.
* **Content-Addressable Caching:** Blazing fast. Documents are securely hashed and their AI embeddings are saved to your local `~/.cache`. If you search the same file again (even if you moved it to a new folder), it loads instantly.
* **Beautiful UI:** Colored output formatting and live-loading spinners.
* **Multi-Format Support:** Parse text from `.pdf`, `.docx`, `.xlsx`, `.epub`, `.md`, `.txt`, and various source code files.
* **Web Page Support:** Pass a URL (`https://...`) to automatically fetch, strip HTML tags, and search its contents.
* **Glob Patterns:** Easily search through entire directories using wildcards (`./docs/**/*.md`).

## Installation

### Prerequisites
* Go 1.21 or higher

### Install
Clone the repository, fetch dependencies, and build the binary:

```bash
git clone <your-repo-url>
cd semantic-search
go mod tidy
go install .
```
*(This will install the binary to your `$GOPATH/bin`, usually `~/go/bin`. Make sure this is in your system's `$PATH`.)*

> **Note:** On the very first run, the tool will automatically download the local embedding model to `~/.cybertron`.

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
Pass URLs directly. The tool will download the page, strip the HTML, and search the text:
```bash
rag "what is the main argument?" https://paulgraham.com/articles.html
```

## Flags Reference

You can fine-tune the search results using flags.

| Flag | Default | Description |
| :--- | :--- | :--- |
| `-k`, `--topK` | `3` | Number of top search results to return. |
| `--chunk` | `800` | The number of characters per text chunk. Smaller chunks give precise snippets; larger chunks give broader context. |
| `--overlap` | `100` | Overlapping characters between chunks. Prevents cutting off context mid-sentence. |
| `--clear-cache`| `false`| Deletes all saved AI embeddings in your system cache to free up space. |

### Example with Flags
Retrieve the top 5 results, breaking the text into smaller 400-character chunks:
```bash
rag -k 5 --chunk 400 --overlap 50 "neural networks" ./research_papers/*.pdf
```
