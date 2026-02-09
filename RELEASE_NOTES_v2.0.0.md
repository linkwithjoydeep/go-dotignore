# Release v2.0.0

## 🎉 Major Update: Full gitignore Specification Support

This major release fixes **three critical bugs** and adds full support for root-relative patterns, bringing go-dotignore into complete compliance with the gitignore specification.

**⚠️ Important:** v2.0.0 was released with incorrect module path. Please use **v2.0.1+** which includes the proper `/v2` import path required by Go modules.

---

## 🐛 Critical Bug Fixes

### 1. Fixed Root-Relative Pattern Support ([#5](https://github.com/linkwithjoydeep/go-dotignore/issues/5)) ⭐ NEW!
**Impact:** HIGH - Critical gitignore feature was missing

Root-relative patterns (starting with `/`) now correctly match only at the repository root level, not in subdirectories.

**Before (❌ Broken):**
```go
// Pattern: /build/
matcher.Matches("build/")          // false (WRONG - didn't match at all!)
matcher.Matches("src/build/")      // false
```

**After (✅ Correct):**
```go
// Pattern: /build/
matcher.Matches("build/")          // true ✓ (matches at root)
matcher.Matches("build/file.txt")  // true ✓ (matches inside root build/)
matcher.Matches("src/build/")      // false ✓ (correctly excludes nested)
matcher.Matches("src/build/file")  // false ✓ (correctly excludes nested)

// Pattern: logs/ (no leading slash - matches anywhere)
matcher.Matches("logs/")           // true ✓
matcher.Matches("src/logs/")       // true ✓ (matches in subdirectories too)
```

**Migration:** If you created workarounds for this bug (e.g., using negation patterns), you can now use proper root-relative patterns:

```go
// OLD WORKAROUND (can remove):
patterns := []string{
    "mydir/",
    "!example/mydir/",  // Had to negate subdirectories
}

// NEW (proper way):
patterns := []string{
    "/mydir/",  // Only matches at root, not in subdirectories
}
```

---

### 2. Fixed Incorrect Substring Matching
**Impact:** HIGH - Could cause false positive matches

Pattern matching with path separators was incorrectly using substring matching without boundary checks.

**Before (❌ Incorrect):**
```go
// Pattern: "src/test"
matcher.Matches("mysrc/test")  // Returned true (WRONG!)
matcher.Matches("src/test2")   // Returned true (WRONG!)
```

**After (✅ Correct):**
```go
// Pattern: "src/test"
matcher.Matches("mysrc/test")      // Returns false ✓
matcher.Matches("src/test2")       // Returns false ✓
matcher.Matches("src/test")        // Returns true ✓
matcher.Matches("foo/src/test")    // Returns true ✓
matcher.Matches("src/test/file")   // Returns true ✓
```

---

### 3. Added Support for Escaped Negation Patterns
**Impact:** MEDIUM - New capability for literal `!` matching

Patterns starting with `\!` now correctly match files with literal `!` in the name, following gitignore specification.

**New Capability:**
```go
patterns := []string{
    "*.log",           // Ignore all .log files
    "!important.log",  // Exception: don't ignore important.log
    `\!special.log`,   // NEW: Match files literally named "!special.log"
}
```

---

## ✨ New Features

### Root-Relative Pattern Support (`/pattern`)
- **`/build/`** → Matches only root-level `build/`, not `src/build/`
- **`/*.txt`** → Matches `.txt` files only at root, not in subdirectories
- **`/src/*.go`** → Matches `.go` files only in root-level `src/`

### Escaped Negation (`\!pattern`)
- **`\!important.txt`** → Matches files literally named `!important.txt`
- Full support for gitignore escape sequences

---

## 🧪 Testing Improvements

- **Added 14 new tests** covering edge cases (+29% test coverage)
- **Unicode/Non-ASCII support:** Japanese (日本語), Russian (файл), Emoji (🎉)
- **Deep path testing:** Verified with 100+ directory levels
- **Long pattern testing:** Stress tested with 1000+ character patterns
- **Total tests:** 47 → 61 tests, all passing ✅

**New test coverage:**
- `TestLeadingSlashPatterns` - Root-relative pattern validation (Issue #5)
- `TestRootRelativeWithWildcards` - Root-relative with wildcards
- `TestSubstringMatchingBug` - Ensures correct boundary checking
- `TestEscapedNegation` - Validates `\!` pattern handling
- `TestUnicodePatterns` - Non-ASCII filename support
- `TestVeryDeepPaths` - Deep directory hierarchies
- `TestConsecutiveWildcards` - Complex wildcard patterns
- `TestVeryLongPatterns` - Large pattern handling
- `TestEdgeCasePatterns` - Various edge cases

---

## 📊 Performance

Performance remains excellent with no regressions:
- **Speed:** ~34µs per match operation
- **Memory:** 3,749 bytes/op
- **Allocations:** 148 allocs/op
- **Concurrency:** Thread-safe, no race conditions detected

---

## 🔄 Migration Guide

### For Most Users: Minimal Changes Required

#### If You Used Root-Relative Patterns (Issue #5)

**Scenario 1: Patterns weren't working before**
```go
// Your patterns now work correctly!
patterns := []string{
    "/build/",    // NOW WORKS: Matches only root-level build/
    "/test.txt",  // NOW WORKS: Matches only root-level test.txt
}
// No changes needed - it just works now ✅
```

**Scenario 2: You created workarounds**
```go
// BEFORE (workaround):
patterns := []string{
    "mydir/",
    "!subdir/mydir/",   // Had to manually exclude subdirectories
    "!other/mydir/",
}

// AFTER (proper solution):
patterns := []string{
    "/mydir/",  // Automatically matches only at root
}
```

#### If Affected by Substring Bug Fix

Check if you have patterns like these:
```go
patterns := []string{
    "src/config",      // Did you expect this to match "mysrc/config"?
    "test/data",       // Did you expect this to match "test/data123"?
}
```

If you were **intentionally** relying on substring matching (unlikely), update patterns:
```go
// OLD (relied on bug):
"src/config"  // Incorrectly matched "mysrc/config"

// NEW (correct approach):
"*src/config"  // Use wildcard to explicitly match anywhere
```

#### If You Use Escaped Negation

```go
// Now works correctly:
`\!important.txt`  // Matches files literally named "!important.txt"
```

---

## 📦 Upgrade Instructions

```bash
# Go modules (recommended)
go get github.com/codeglyph/go-dotignore/v2@v2.0.1

# Or update your go.mod
github.com/codeglyph/go-dotignore/v2 v2.0.0
```

Then run:
```bash
go mod tidy
go test ./...  # Verify your patterns still work as expected
```

---

## 🧪 Testing Your Upgrade

After upgrading:

1. **Test root-relative patterns** - Verify `/pattern` behaves correctly
2. **Check for substring assumptions** - Review patterns with `/`
3. **Test with escaped negation** - Verify `\!` patterns work
4. **Validate Unicode filenames** - Ensure non-ASCII files match correctly

### Test Script:
```go
package main

import (
    "fmt"
    "github.com/codeglyph/go-dotignore/v2"
)

func main() {
    patterns := []string{
        "/build/",     // Root-relative
        "logs/",       // Matches anywhere
        "src/test",    // With path separator
        `\!special`,   // Escaped negation
    }

    matcher, _ := dotignore.NewPatternMatcher(patterns)

    tests := map[string]bool{
        "build/":           true,   // /build/ matches at root
        "src/build/":       false,  // /build/ doesn't match in subdirs
        "logs/":            true,   // logs/ matches at root
        "src/logs/":        true,   // logs/ matches in subdirs
        "src/test":         true,   // src/test matches
        "mysrc/test":       false,  // src/test doesn't match mysrc/test
        "!special":         true,   // \!special matches literal !
    }

    for file, expected := range tests {
        result, _ := matcher.Matches(file)
        status := "✅"
        if result != expected {
            status = "❌"
        }
        fmt.Printf("%s %s: expected %v, got %v\n", status, file, expected, result)
    }
}
```

---

## ✅ Compatibility

- **Go Version:** Requires Go 1.20+ (unchanged)
- **API Compatibility:** Same function signatures, enhanced behavior
- **Platform Support:** Linux, macOS, Windows (unchanged)
- **Breaking Changes:** Bug fixes may change behavior for incorrect usage patterns

---

## 🎯 Why v2.0.0?

This is a **major version** because:

1. **New Feature:** Root-relative pattern support significantly changes how patterns work
2. **Behavior Changes:** Three critical bugs fixed that may affect existing workarounds
3. **Gitignore Compliance:** Now fully compliant with gitignore spec (was partially broken before)
4. **User Impact:** Users who worked around bugs may need to update their patterns

The library is now **production-grade** with full gitignore specification support.

---

## 📝 Full Changelog

**Fixed:**
- Root-relative patterns (`/pattern`) now work correctly per gitignore spec (Issue #5)
- Substring matching bug causing false positives with path separator patterns
- Escaped negation (`\!`) now properly handled per gitignore spec
- Path boundary checking now correctly validates directory components

**Added:**
- Support for root-relative patterns with leading `/`
- Support for escaped negation patterns (`\!`)
- Comprehensive Unicode/non-ASCII filename support
- 14 new edge case tests covering root-relative, wildcards, and Unicode

**Removed:**
- Internal unused `NormalizePath()` function (non-breaking, never exported)

**Changed:**
- Improved code documentation and comments
- Optimized string boundary checking
- Test count increased from 47 to 61 tests (+29%)

**Performance:**
- No regressions: maintains ~34µs per match operation
- Memory usage: 3,749 bytes/op (unchanged)
- Allocations: 148 allocs/op (unchanged)

---

## 🙏 Acknowledgments

Thanks to [@shadiramadan](https://github.com/shadiramadan) for reporting Issue #5 and all users who helped improve this library!

---

**Full Diff:** [v1.1.1...v2.0.0](https://github.com/codeglyph/go-dotignore/v2/compare/v1.1.1...v2.0.0)

**Closes:** [#5](https://github.com/linkwithjoydeep/go-dotignore/issues/5)
