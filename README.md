# AI Tokenizer

A simple and efficient Go library for token estimation in text processing pipelines.

This library is ideal for scenarios where a fast, dependency-free approximation of token count is needed without the overhead of a full, model-specific tokenizer.

---

## Features

- **⚡️ High Performance**: Optimized for speed with minimal memory allocations using a single-pass processor and efficient string building.
- **💡 Simple Tokenization**: Uses a straightforward algorithm where special characters are counted individually and regular text is estimated at approximately 2 characters per token.
- **✍️ Unicode Normalization**: Robustly converts Unicode text (e.g., `café`, `Müller`) into its ASCII equivalent by removing diacritics and folding special characters.
- **📦 Zero Dependencies**: Built using only the Go standard library and the official `golang.org/x/text` package.
- **🚀 Command-Line Interface**: Includes a simple CLI for easy integration into scripts and shell pipelines.

---

## Installation

```bash
go get github.com/nnikolov3/ai-tokenizer
```

---

## Quick Start

```go
package main

import (
	"fmt"
	"github.com/nnikolov3/ai-tokenizer"
)

func main() {
	tok := tokenizer.NewTokenizer()

	// Estimate tokens in a string
	count := tok.EstimateTokens("Hello, world!")
	fmt.Printf("Token count: %d\n", count) // Output: 9

	// Normalize Unicode text to clean ASCII
	normalized := tok.Normalize("café naïve")
	fmt.Printf("Normalized: %s\n", normalized) // Output: cafe naive
}
```

---

## API Reference

### `NewTokenizer() *Tokenizer`

Creates and returns a new tokenizer instance.

### `EstimateTokens(text string) int`

Estimates the number of tokens in the given text.

- **Special characters** (whitespace, punctuation, symbols) count as **1 token** each.
- Sequences of **regular characters** (letters and digits) are counted as `ceil(length / 2)`.

### `Normalize(text string) string`

Converts a Unicode string to its closest ASCII equivalent. This process involves removing diacritics (`é` → `e`), converting ligatures (`ß` → `ss`), and filtering out unsupported characters.

### `GetModel() string`

Returns the tokenizer model name, which is currently `"simple"`.

---

## Algorithm Details

The tokenizer uses a two-step process for estimation:

1.  **Normalization**: Text is first normalized using Unicode NFD decomposition. The process then removes combining marks (like accents), folds special characters to ASCII equivalents, and filters out any non-convertible characters.
2.  **Token Counting**: The normalized text is processed character-by-character. Special characters (anything not a letter or digit) are counted as 1 token each. Sequences of regular characters are grouped, and their token count is calculated as `ceil(length / 2)`.

---

## CLI Usage

A command-line interface is included for quick estimations and normalization directly from your terminal.

```bash
# Install the CLI tool
go install github.com/nnikolov3/ai-tokenizer/cmd/ai-tokenizer@latest

# Estimate tokens from stdin
$ echo "Hello, world!" | ai-tokenizer estimate
Token count: 9

# Normalize text from stdin
$ echo "café naïve" | ai-tokenizer normalize
Normalized: cafe naive
```

---

## Contributing

Contributions are welcome\! Please follow these steps:

1.  Fork the repository.
2.  Create a feature branch: `git checkout -b feature-name`.
3.  Make your changes and add or update tests.
4.  Run the test suite: `go test -v`.
5.  Run the linter: `golangci-lint run`.
6.  Submit a pull request.

## License

This project is licensed under the MIT License. See the [LICENSE](https://www.google.com/search?q=LICENSE) file for details.
