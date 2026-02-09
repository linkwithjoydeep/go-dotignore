# Migration Guide: v1.x to v2.0+

## Why Upgrade?

Versions v1.0.0-v1.1.1 contain **three critical bugs** and are officially retracted. All users should upgrade to v2.0.1+ immediately.

**Note:** v2.0.0 was tagged with an incorrect module path. Use v2.0.1+ which includes the proper `/v2` suffix in the import path.

## Critical Bugs Fixed in v2.0.0

### 1. Root-Relative Patterns Didn't Work (Issue #5)
**Severity:** CRITICAL

**Problem:** Patterns starting with `/` didn't match anything at all.

```go
// v1.x (BROKEN):
patterns := []string{"/build/"}
matcher.Matches("build/")  // false ❌ (should be true)

// v2.0.0 (FIXED):
patterns := []string{"/build/"}
matcher.Matches("build/")          // true ✅
matcher.Matches("src/build/")      // false ✅
```

### 2. Substring Matching Bug
**Severity:** CRITICAL

**Problem:** Pattern `src/test` incorrectly matched `mysrc/test` and `src/test2`.

```go
// v1.x (BROKEN):
patterns := []string{"src/test"}
matcher.Matches("mysrc/test")  // true ❌ (should be false)
matcher.Matches("src/test2")   // true ❌ (should be false)

// v2.0.0 (FIXED):
patterns := []string{"src/test"}
matcher.Matches("mysrc/test")  // false ✅
matcher.Matches("src/test2")   // false ✅
matcher.Matches("src/test")    // true ✅
```

### 3. No Escaped Negation Support
**Severity:** MEDIUM

**Problem:** Couldn't match files starting with literal `!`.

```go
// v1.x (BROKEN):
patterns := []string{`\!important.txt`}
matcher.Matches("!important.txt")  // false ❌ (should be true)

// v2.0.0 (FIXED):
patterns := []string{`\!important.txt`}
matcher.Matches("!important.txt")  // true ✅
```

---

## How to Upgrade

### Step 1: Update Dependency

```bash
go get github.com/codeglyph/go-dotignore/v2@v2.0.1
go mod tidy
```

### Step 2: Remove Workarounds (If Any)

If you created workarounds for the bugs, remove them:

#### Root-Relative Pattern Workarounds

```go
// OLD WORKAROUND (remove this):
patterns := []string{
    "build/",
    "!src/build/",
    "!lib/build/",
    // ... manually excluding all subdirectories
}

// NEW (proper solution):
patterns := []string{
    "/build/",  // Automatically matches only at root
}
```

#### Substring Matching Workarounds

```go
// OLD WORKAROUND (remove this):
patterns := []string{
    "*/src/test",    // Had to be more specific
    "src/test/*",    // Split into multiple patterns
}

// NEW (works correctly now):
patterns := []string{
    "src/test",  // Now has proper boundary checking
}
```

### Step 3: Test Your Patterns

Run this test to verify your patterns work correctly:

```go
package main

import (
    "fmt"
    "log"
    "github.com/codeglyph/go-dotignore/v2"
)

func main() {
    // Test your actual patterns
    patterns := []string{
        "/build/",
        "*.log",
        "!important.log",
        "temp/",
    }

    matcher, err := dotignore.NewPatternMatcher(patterns)
    if err != nil {
        log.Fatal(err)
    }

    // Test cases specific to your use case
    testCases := map[string]bool{
        "build/":          true,   // Root-level build
        "src/build/":      false,  // Nested build (should NOT match /build/)
        "app.log":         true,   // Matches *.log
        "important.log":   false,  // Negated
        "temp/":           true,   // Directory pattern
    }

    for file, expected := range testCases {
        result, err := matcher.Matches(file)
        if err != nil {
            log.Fatal(err)
        }
        status := "✅"
        if result != expected {
            status = "❌ FAILED"
        }
        fmt.Printf("%s %s: expected %v, got %v\n", status, file, expected, result)
    }
}
```

### Step 4: Update CI/CD (If Needed)

If you pin versions in CI/CD, update to v2.0.1+:

```yaml
# GitHub Actions
- name: Install dependencies
  run: go get github.com/codeglyph/go-dotignore/v2@v2.0.1
```

```dockerfile
# Dockerfile
RUN go get github.com/codeglyph/go-dotignore/v2@v2.0.1
```

---

## Breaking Changes

### None for Correct Usage

If your patterns were using the library correctly (as intended per gitignore spec), **no code changes are required**. The bugs are fixed, and correct usage just works now.

### Only If You Relied on Bugs

If you were **relying on the buggy behavior** (unlikely), you'll need to update:

1. **Root-relative patterns now work** - If you expected `/build/` to not match anything, this behavior has changed (but it was a bug).

2. **Substring matching fixed** - If you expected `src/test` to match `mysrc/test`, this no longer happens (but it was a bug).

---

## New Features in v2.0.0

### Root-Relative Pattern Support

```go
patterns := []string{
    "/build/",       // Only root-level build/
    "/test.txt",     // Only root-level test.txt
    "/*.go",         // Only .go files at root
    "/src/*.js",     // Only .js files in root-level src/
}
```

### Escaped Negation

```go
patterns := []string{
    `\!important.txt`,  // Matches files literally named "!important.txt"
    `\!special`,        // Matches files starting with literal "!"
}
```

---

## Common Migration Scenarios

### Scenario 1: No Changes Needed

**Your patterns were correct, just buggy behavior is fixed:**

```go
// Your code (unchanged):
patterns := []string{
    "*.log",
    "!important.log",
    "temp/",
}
// Just upgrade and it works better now! ✅
```

### Scenario 2: Remove Root-Relative Workarounds

**You worked around broken root-relative patterns:**

```go
// BEFORE v2.0.0:
patterns := []string{
    "build/",           // Matched everywhere
    "!src/build/",      // Had to manually exclude
    "!lib/build/",
    "!test/build/",
}

// AFTER v2.0.0:
patterns := []string{
    "/build/",  // Now works correctly - only matches at root
}
```

### Scenario 3: Simplify Specific Path Patterns

**You made patterns overly specific due to substring bug:**

```go
// BEFORE v2.0.0:
patterns := []string{
    "^src/test$",       // Used regex-like specificity
    "**/src/test",      // Or wildcards to be more specific
}

// AFTER v2.0.0:
patterns := []string{
    "src/test",  // Simple pattern now works correctly
}
```

---

## Compatibility Matrix

| Feature | v1.0.0-v1.1.1 | v2.0.0+ |
|---------|---------------|---------|
| Root-relative patterns (`/pattern`) | ❌ Broken | ✅ Works |
| Substring matching bug | ❌ Broken | ✅ Fixed |
| Escaped negation (`\!`) | ❌ Not supported | ✅ Supported |
| Standard wildcards (`*`, `?`, `**`) | ✅ Works | ✅ Works |
| Negation (`!pattern`) | ✅ Works | ✅ Works |
| Directory patterns (`dir/`) | ✅ Works | ✅ Works |
| Character classes (`[a-z]`) | ✅ Works | ✅ Works |
| Cross-platform paths | ✅ Works | ✅ Works |
| Thread-safety | ✅ Safe | ✅ Safe |

---

## Rollback (Not Recommended)

If you absolutely must rollback (not recommended due to critical bugs):

```bash
go get github.com/codeglyph/go-dotignore/v2@v1.1.1
```

**However, this is strongly discouraged** as v1.x contains critical bugs that can cause:
- False positive matches (files ignored when they shouldn't be)
- False negative matches (files not ignored when they should be)
- Inability to use root-relative patterns

---

## Getting Help

If you encounter issues during migration:

1. Check the [Release Notes](https://github.com/codeglyph/go-dotignore/v2/releases/tag/v2.0.0)
2. Review [Examples](https://pkg.go.dev/github.com/codeglyph/go-dotignore/v2#pkg-examples)
3. Open an [Issue](https://github.com/codeglyph/go-dotignore/v2/issues)

---

## Summary

- ✅ **Upgrade to v2.0.0** - Critical bugs fixed
- ✅ **Remove workarounds** - Features now work correctly
- ✅ **Test your patterns** - Verify behavior is as expected
- ✅ **No API changes** - Same function signatures
- ✅ **Better compliance** - Full gitignore specification support

**Upgrade now:** `go get github.com/codeglyph/go-dotignore/v2@latest`
