[![build](https://github.com/codeglyph/go-dotignore/actions/workflows/build.yml/badge.svg)](https://github.com/codeglyph/go-dotignore/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/codeglyph/go-dotignore/v2)](https://goreportcard.com/report/github.com/codeglyph/go-dotignore/v2)
[![GoDoc](https://godoc.org/github.com/codeglyph/go-dotignore/v2?status.svg)](https://godoc.org/github.com/codeglyph/go-dotignore/v2)
[![GitHub release](https://img.shields.io/github/v/release/codeglyph/go-dotignore)](https://github.com/codeglyph/go-dotignore/v2/releases)

# go-dotignore

> **⚠️ IMPORTANT:** If you're using v1.x.x, please upgrade to **v2.0.1+** immediately.
> Versions v1.0.0-v1.1.1 contain **critical bugs** and have been retracted:
> - ❌ Root-relative patterns (`/pattern`) don't work at all
> - ❌ Substring matching causes false positives
> - ❌ No escaped negation support
>
> **➡️ Upgrade now:** `go get github.com/codeglyph/go-dotignore/v2@latest`
>
> **📦 Note:** v2+ requires the `/v2` suffix in import paths (Go module requirement).
> See [Release Notes](https://github.com/codeglyph/go-dotignore/v2/releases/tag/v2.0.0) for details.

**go-dotignore** is a high-performance Go library for parsing `.gitignore`-style files and matching file paths against specified ignore patterns. It provides full support for advanced ignore rules, negation patterns, and wildcards, making it an ideal choice for file exclusion in Go projects.

## Features

- 🚀 **High Performance** - Optimized pattern matching with efficient regex compilation
- 📁 **Complete .gitignore Support** - Full compatibility with Git's ignore specification
- 🔄 **Negation Patterns** - Use `!` to override ignore rules
- 🌟 **Advanced Wildcards** - Support for `*`, `?`, and `**` patterns
- 📂 **Directory Matching** - Proper handling of directory-only patterns with `/`
- 🔒 **Cross-Platform** - Consistent behavior across Windows, macOS, and Linux
- ⚡ **Memory Efficient** - Minimal memory footprint with lazy evaluation
- 🛡️ **Error Handling** - Comprehensive error reporting and validation
- 📝 **Well Documented** - Extensive examples and godoc documentation

## Installation

**Recommended (v2.0.1+):**
```bash
go get github.com/codeglyph/go-dotignore/v2@latest
```

Or in your `go.mod`:
```go
require github.com/codeglyph/go-dotignore/v2 v2.0.1
```

**⚠️ Important Notes:**
- v2+ requires the `/v2` suffix in the import path (Go module semantic versioning requirement)
- v2.0.0 was released with incorrect module path - use v2.0.1+
- Versions v1.0.0-v1.1.1 are retracted due to critical bugs

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    "github.com/codeglyph/go-dotignore/v2"
)

func main() {
    // Create matcher from patterns
    patterns := []string{
        "*.log",           // Ignore all .log files
        "!important.log",  // But keep important.log
        "temp/",           // Ignore temp directory
        "**/*.tmp",        // Ignore .tmp files anywhere
    }

    matcher, err := dotignore.NewPatternMatcher(patterns)
    if err != nil {
        log.Fatal(err)
    }

    // Check if files should be ignored
    files := []string{
        "app.log",          // true - matches *.log
        "important.log",    // false - negated by !important.log
        "temp/cache.txt",   // true - in temp/ directory
        "src/backup.tmp",   // true - matches **/*.tmp
    }

    for _, file := range files {
        ignored, err := matcher.Matches(file)
        if err != nil {
            log.Printf("Error checking %s: %v", file, err)
            continue
        }
        fmt.Printf("%-20s ignored: %v\n", file, ignored)
    }
}
```

## Usage Examples

### Loading from File

```go
// Parse .gitignore file
matcher, err := dotignore.NewPatternMatcherFromFile(".gitignore")
if err != nil {
    log.Fatal(err)
}

ignored, err := matcher.Matches("build/output.js")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Should ignore: %v\n", ignored)
```

### Loading from Reader

```go
import (
    "strings"
    "github.com/codeglyph/go-dotignore/v2"
)

patterns := `
# Dependencies
node_modules/
vendor/

# Build outputs
*.exe
*.so
*.dylib
dist/

# Logs
*.log
!debug.log

# OS generated files
.DS_Store
Thumbs.db
`

reader := strings.NewReader(patterns)
matcher, err := dotignore.NewPatternMatcherFromReader(reader)
if err != nil {
    log.Fatal(err)
}
```

### Advanced Pattern Examples

```go
patterns := []string{
    // Basic wildcards
    "*.txt",              // All .txt files
    "file?.log",          // file1.log, fileA.log, etc.

    // Directory patterns
    "cache/",             // Only directories named cache
    "logs/**",            // Everything in logs directory

    // Recursive patterns
    "**/*.test.js",       // All .test.js files anywhere
    "**/node_modules/",   // node_modules at any level

    // Negation patterns
    "build/",             // Ignore build directory
    "!build/README.md",   // But keep README.md in build

    // Complex patterns
    "src/**/temp/",       // temp directories anywhere under src
    "*.{log,tmp,cache}",  // Multiple extensions (if supported)
}
```

## Pattern Syntax

### Wildcards

| Pattern | Description                 | Example Matches                            |
| ------- | --------------------------- | ------------------------------------------ |
| `*`     | Any characters except `/`   | `*.txt` → `file.txt`, `data.txt`           |
| `?`     | Single character except `/` | `file?.txt` → `file1.txt`, `fileA.txt`     |
| `**`    | Zero or more directories    | `**/test` → `test`, `src/test`, `a/b/test` |

### Directory Patterns

| Pattern   | Description            | Example Matches                     |
| --------- | ---------------------- | ----------------------------------- |
| `dir/`    | Directory only         | `build/` → `build/` (directory)     |
| `dir/**`  | Directory contents     | `src/**` → everything in `src/`     |
| `**/dir/` | Directory at any level | `**/temp/` → `temp/`, `cache/temp/` |

### Negation

```go
patterns := []string{
    "*.log",           // Ignore all .log files
    "!important.log",  // Exception: keep important.log
    "temp/",           // Ignore temp directory
    "!temp/keep.txt",  // Exception: keep temp/keep.txt
}
```

**Note**: Pattern order matters! Later patterns override earlier ones.
