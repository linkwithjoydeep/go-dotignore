# v1.2.0 - Bug Fixes & Enhanced Testing

## 🐛 Critical Bug Fixes

### Fixed Substring Matching Bug (HIGH PRIORITY)
Pattern matching with path separators was incorrectly using substring matching, causing false positives.

**Fixed:**
- Pattern `src/test` no longer incorrectly matches `mysrc/test` or `src/test2`
- Now properly checks path boundaries following gitignore specification

### Added Escaped Negation Support
Patterns starting with `\!` now correctly match files with literal `!` in the name.

```go
`\!important.txt`  // Now matches files literally named "!important.txt"
```

## 🧪 Testing Improvements

- **+21% more tests:** 47 → 57 tests (all passing ✅)
- **Unicode support verified:** Japanese, Russian, Emoji filenames
- **Deep paths tested:** 100+ directory levels
- **Long patterns tested:** 1000+ characters
- **No race conditions detected**

## 📊 Performance

No regressions - maintains excellent performance:
- ~34µs per match operation
- Thread-safe, production-ready

## 🔄 Migration

**For most users:** Drop-in replacement, no changes needed ✅

**If affected by substring bug:** Review patterns with `/` to ensure they match your intent. See full release notes for details.

## 📦 Installation

```bash
go get github.com/codeglyph/go-dotignore@v1.2.0
```

## Full Details

See [RELEASE_NOTES_v1.2.0.md](./RELEASE_NOTES_v1.2.0.md) for complete changelog and migration guide.
