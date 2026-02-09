# Release v2.1.0 - Hierarchical .gitignore Support

**Release Date:** 2026-02-09

## 🎉 What's New

### Nested .gitignore Support (Issue #4)

The **most requested feature** is finally here! go-dotignore now supports hierarchical .gitignore matching for real-world repositories with nested .gitignore files - just like Git does!

#### The Problem

Previously, go-dotignore (like go-gitignore) could only process a single .gitignore file. This made it impractical for real-world repositories, especially monorepos, where different subdirectories have their own .gitignore files.

#### The Solution

New `RepositoryMatcher` API that:
- ✅ Automatically discovers all .gitignore files in your repository
- ✅ Applies patterns hierarchically (parent → child)
- ✅ Supports pattern overrides via negation
- ✅ Mirrors Git's exact behavior

### Quick Example

```go
// Before: Only single file support
matcher, err := dotignore.NewPatternMatcherFromFile(".gitignore")

// After: Full repository support with nested .gitignore files
matcher, err := dotignore.NewRepositoryMatcher("/path/to/repo")

// Automatically handles:
// - .gitignore (root)
// - frontend/.gitignore
// - backend/.gitignore
// - docs/.gitignore
// And applies them correctly, just like Git!

ignored, err := matcher.Matches("frontend/node_modules/package.json")
```

## 📦 New API

### `RepositoryMatcher`

```go
type RepositoryMatcher struct { ... }

// Create a new repository matcher
func NewRepositoryMatcher(rootDir string) (*RepositoryMatcher, error)

// Create with custom configuration
func NewRepositoryMatcherWithConfig(rootDir string, config *RepositoryConfig) (*RepositoryMatcher, error)

// Check if a file should be ignored
func (rm *RepositoryMatcher) Matches(path string) (bool, error)

// Helper methods
func (rm *RepositoryMatcher) RootDir() string
func (rm *RepositoryMatcher) IgnoreFileCount() int
func (rm *RepositoryMatcher) IgnoreFilePaths() []string
```

### `RepositoryConfig`

```go
type RepositoryConfig struct {
    IgnoreFileName string  // Default: ".gitignore"
    MaxDepth       int     // Default: 0 (unlimited)
    FollowSymlinks bool    // Default: false
}
```

## 🏗️ Use Cases

Perfect for:
- **Monorepo projects** with multiple subprojects
- **Build tools** that need to respect .gitignore rules
- **File walkers** and code analyzers
- **Git-aware tools** and utilities
- **Migration from go-gitignore** (drop-in replacement + more features)

## 📊 Real-World Example: Monorepo

```go
// Repository structure:
// project/
//   .gitignore              # Global: *.log, .env, .DS_Store
//   frontend/
//     .gitignore            # Frontend: node_modules/, dist/, .cache/
//     node_modules/         # ← Ignored by frontend/.gitignore
//     src/                  # ← Not ignored
//   backend/
//     .gitignore            # Backend: target/, *.class, logs/
//     target/               # ← Ignored by backend/.gitignore
//     src/                  # ← Not ignored
//   docs/
//     .gitignore            # Docs: _build/, *.pyc
//     _build/               # ← Ignored by docs/.gitignore

matcher, err := dotignore.NewRepositoryMatcher("/path/to/project")

// Global patterns apply everywhere
matcher.Matches("app.log")                    // true (root .gitignore)
matcher.Matches("frontend/debug.log")         // true (root .gitignore)
matcher.Matches(".DS_Store")                  // true (root .gitignore)

// Subproject-specific patterns
matcher.Matches("frontend/node_modules/...")  // true (frontend/.gitignore)
matcher.Matches("backend/target/...")         // true (backend/.gitignore)
matcher.Matches("docs/_build/...")            // true (docs/.gitignore)

// Source files are NOT ignored
matcher.Matches("frontend/src/App.js")        // false
matcher.Matches("backend/src/Main.java")      // false
matcher.Matches("docs/index.rst")             // false
```

## 🔄 Pattern Override Example

Child .gitignore files can override parent patterns:

```go
// project/
//   .gitignore              # *.txt (ignore all .txt files)
//   important/
//     .gitignore            # !critical.txt (but keep this one)

matcher.Matches("file.txt")                   // true (ignored by root)
matcher.Matches("important/file.txt")         // true (still ignored)
matcher.Matches("important/critical.txt")     // false (un-ignored by child!)
```

## ⚙️ Configuration

```go
config := &dotignore.RepositoryConfig{
    IgnoreFileName: ".ignore",     // Use custom ignore file names
    MaxDepth:       5,              // Only search 5 levels deep
    FollowSymlinks: false,          // Don't follow symbolic links
}

matcher, err := dotignore.NewRepositoryMatcherWithConfig("/path/to/repo", config)
fmt.Printf("Found %d ignore files\n", matcher.IgnoreFileCount())

// Get list of all loaded ignore files
for _, path := range matcher.IgnoreFilePaths() {
    fmt.Printf("Loaded: %s\n", path)
}
```

## 📈 Comparison with Other Libraries

### vs. go-gitignore

| Feature | go-dotignore v2.1+ | go-gitignore |
|---------|-------------------|--------------|
| **Nested .gitignore support** | ✅ Yes | ❌ No |
| **Root-relative patterns** | ✅ Fully compliant | ⚠️ Issues |
| **Negation patterns** | ✅ Full support | ⚠️ Limited |
| **Escaped negation** | ✅ Yes | ❌ No |
| **Active maintenance** | ✅ Yes | ⚠️ Unmaintained |
| **Full gitignore spec** | ✅ v2.0+ compliant | ⚠️ Partial |

**go-dotignore is now a complete, drop-in replacement for go-gitignore with significantly more features!**

## 🧪 Testing

Comprehensive test suite with 15+ new test cases:
- ✅ Simple hierarchy matching
- ✅ Pattern negation across levels
- ✅ Monorepo scenarios (frontend/backend/docs)
- ✅ Pattern override with negation
- ✅ Root-relative patterns in nested files
- ✅ Absolute path handling
- ✅ Configuration options (MaxDepth, custom file names)
- ✅ Edge cases and error handling

**All tests passing!**

## 📚 Documentation

- ✅ Updated README with nested .gitignore section
- ✅ Comparison table with go-gitignore
- ✅ Migration guide from go-gitignore
- ✅ Example code and usage patterns
- ✅ Detailed API documentation

## 🚀 Installation

```bash
go get github.com/codeglyph/go-dotignore/v2@v2.1.0
```

Or update your `go.mod`:
```go
require github.com/codeglyph/go-dotignore/v2 v2.1.0
```

## 🔧 Migration from go-gitignore

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

// New (go-dotignore) - nested files (NEW!)
matcher, err := dotignore.NewRepositoryMatcher("/path/to/repo")
if ignored, _ := matcher.Matches("frontend/node_modules/pkg.json"); ignored {
    // file is ignored
}
```

## ⚡ Performance

- No performance regression for single-file usage (`PatternMatcher`)
- Efficient pattern caching per directory level
- Lazy evaluation of patterns
- Minimal memory overhead

## 🎯 What's Next?

This release completes the core feature set for hierarchical gitignore support. Future releases will focus on:
- Performance optimizations
- Additional pattern syntax features
- Community-requested enhancements

## 📝 Full Changelog

See [CHANGELOG.md](CHANGELOG.md) for complete details.

## 🙏 Acknowledgments

Thanks to the community for requesting this feature and providing valuable feedback!

Special mention to the Reddit discussion that highlighted the need for nested .gitignore support in Go libraries.

---

**Upgrading?** This is a minor version bump with no breaking changes. Simply update your dependency and start using `RepositoryMatcher` for repositories with nested .gitignore files!
