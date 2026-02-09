# v2.0.0 - Full gitignore Specification Support

## 🎉 Major Update

This release fixes **three critical bugs** and adds full support for root-relative patterns, achieving complete gitignore specification compliance!

**⚠️ Note:** Use v2.0.1+ with the `/v2` import path for proper Go module support.

---

## 🐛 Critical Bug Fixes

### 1. Root-Relative Patterns Now Work! (Issue #5) ⭐
**Finally!** Patterns starting with `/` now correctly match only at the repository root.

```go
// Pattern: /build/
matcher.Matches("build/")          // ✅ true (root level)
matcher.Matches("src/build/")      // ✅ false (not at root)

// Pattern: logs/ (no slash - matches anywhere)
matcher.Matches("logs/")           // ✅ true
matcher.Matches("src/logs/")       // ✅ true
```

**Before:** `/build/` didn't match anything ❌
**After:** `/build/` matches only root-level build/ ✅

### 2. Fixed Substring Matching Bug
Pattern `src/test` no longer incorrectly matches `mysrc/test` or `src/test2`.

### 3. Added Escaped Negation Support
Pattern `\!important.txt` now matches files literally named `!important.txt`.

---

## ✨ What's New

- **Root-relative patterns:** `/pattern` matches only at root
- **Full gitignore spec compliance:** All standard gitignore features now work
- **14 new tests:** Unicode, deep paths, wildcards, and edge cases
- **61 total tests** (was 47) - all passing ✅

---

## 📊 Testing Improvements

- ✅ Unicode support: Japanese, Russian, Emoji filenames
- ✅ Deep paths: 100+ directory levels tested
- ✅ Long patterns: 1000+ characters tested
- ✅ No race conditions detected
- ✅ No performance regressions (~34µs per match)

---

## 🔄 Migration

**Most users:** No changes needed! The bugs are fixed and features now work correctly.

**If you worked around Issue #5:**
```go
// BEFORE (workaround):
patterns := []string{
    "mydir/",
    "!example/mydir/",  // Had to manually exclude
}

// AFTER (proper solution):
patterns := []string{
    "/mydir/",  // Automatically matches only at root
}
```

---

## 📦 Installation

```bash
go get github.com/codeglyph/go-dotignore/v2@v2.0.1
```

---

## 🎯 Why v2.0.0?

- Major new feature: root-relative pattern support
- Three critical bug fixes that change behavior
- Full gitignore specification compliance
- Users with workarounds may need minor updates

---

## ⚡ Performance

No regressions:
- ~34µs per match operation
- Thread-safe, production-ready
- Same memory footprint

---

## 📋 Full Details

See [RELEASE_NOTES_v2.0.0.md](./RELEASE_NOTES_v2.0.0.md) for complete changelog, migration guide, and test script.

**Closes:** #5
