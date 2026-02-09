[![build](https://github.com/codeglyph/go-dotignore/v2/actions/workflows/build.yml/badge.svg)](https://github.com/codeglyph/go-dotignore/v2/actions/workflows/build.yml)
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
- 🏗️ **Nested .gitignore Support** - Hierarchical matching for monorepos (like Git!)
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

### Basic Pattern Matching

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

### Nested .gitignore Support (Monorepos)

```go
// For repositories with nested .gitignore files
matcher, err := dotignore.NewRepositoryMatcher("/path/to/repo")
if err != nil {
    log.Fatal(err)
}

// Automatically handles nested .gitignore files like Git does
ignored, err := matcher.Matches("frontend/node_modules/package.json")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Should ignore: %v\n", ignored)
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

## Nested .gitignore Support

**NEW in v2.1+**: `RepositoryMatcher` provides hierarchical .gitignore matching for real-world repositories with nested .gitignore files.

### Why You Need This

If you're building tools that work with Git repositories (like file walkers, build tools, or code analyzers), you need to respect ALL .gitignore files in the repository, not just the root one. This is exactly how Git works!

### Example: Monorepo with Multiple .gitignore Files

```go
// Repository structure:
// project/
//   .gitignore              # Global rules: *.log, .env
//   frontend/
//     .gitignore            # Frontend rules: node_modules/, dist/
//     node_modules/         # Ignored by frontend/.gitignore
//   backend/
//     .gitignore            # Backend rules: target/, *.class
//     target/               # Ignored by backend/.gitignore

matcher, err := dotignore.NewRepositoryMatcher("/path/to/project")
if err != nil {
    log.Fatal(err)
}

// Global patterns apply everywhere
matcher.Matches("app.log")                              // true (root .gitignore)
matcher.Matches("frontend/debug.log")                   // true (root .gitignore)

// Subproject patterns apply locally
matcher.Matches("frontend/node_modules/pkg/index.js")   // true (frontend/.gitignore)
matcher.Matches("backend/target/classes/Main.class")    // true (backend/.gitignore)

// Source files are not ignored
matcher.Matches("frontend/src/App.js")                  // false
```

### Pattern Override with Negation

Child .gitignore files can override parent patterns using negation (`!`):

```go
// project/
//   .gitignore              # *.txt (ignore all .txt files)
//   important/
//     .gitignore            # !critical.txt (un-ignore this specific file)

matcher.Matches("file.txt")                   // true (ignored by root)
matcher.Matches("important/file.txt")         // true (still ignored by root)
matcher.Matches("important/critical.txt")     // false (un-ignored by important/.gitignore)
```

### Configuration Options

```go
config := &dotignore.RepositoryConfig{
    IgnoreFileName: ".gitignore",  // Name of ignore files (default: ".gitignore")
    MaxDepth:       10,             // Limit directory depth (0 = unlimited)
    FollowSymlinks: false,          // Whether to follow symbolic links
}

matcher, err := dotignore.NewRepositoryMatcherWithConfig("/path/to/repo", config)
```

## Comparison with Other Libraries

### vs. github.com/sabhiram/go-gitignore

| Feature | go-dotignore | go-gitignore |
|---------|--------------|--------------|
| **Nested .gitignore support** | ✅ Yes (`RepositoryMatcher`) | ❌ No |
| **Root-relative patterns** (`/build/`) | ✅ Correct | ⚠️ Issues |
| **Pattern order/precedence** | ✅ Correct | ⚠️ Issues |
| **Negation patterns** | ✅ Full support | ⚠️ Limited |
| **Escaped negation** (`\!`) | ✅ Yes | ❌ No |
| **Cross-platform** | ✅ Yes | ✅ Yes |
| **Active maintenance** | ✅ Active | ⚠️ Unmaintained |
| **Full gitignore spec** | ✅ v2.0+ compliant | ⚠️ Partial |

**TL;DR:** go-dotignore v2+ is a **drop-in replacement** with full Git spec compliance and nested .gitignore support.

### Migration from go-gitignore

```go
// Old (go-gitignore) - single file only
gitignore, err := ignore.CompileIgnoreFile(".gitignore")
if gitignore.MatchesPath("file.txt") {
    // file is ignored
}

// New (go-dotignore) - single file
matcher, err := dotignore.NewPatternMatcherFromFile(".gitignore")
if ignored, _ := matcher.Matches("file.txt"); ignored {
    // file is ignored
}

// New (go-dotignore) - nested files (like Git!)
matcher, err := dotignore.NewRepositoryMatcher("/path/to/repo")
if ignored, _ := matcher.Matches("frontend/node_modules/pkg.json"); ignored {
    // file is ignored
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
