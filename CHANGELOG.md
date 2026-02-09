# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0] - 2025-02-09

### Fixed
- **Critical ([#5](https://github.com/linkwithjoydeep/go-dotignore/issues/5)):** Root-relative patterns (starting with `/`) now work correctly per gitignore specification
  - Pattern `/build/` now matches only root-level `build/`, not `src/build/`
  - Pattern `/test.txt` now matches only root-level `test.txt`, not `src/test.txt`
  - Previous behavior: root-relative patterns didn't match anything at all
  - New behavior: matches only at repository root level, as per gitignore spec
- **Critical:** Fixed substring matching bug where patterns with path separators incorrectly matched files using substring logic instead of proper path boundary checking
  - Pattern `src/test` no longer matches `mysrc/test` (incorrect behavior)
  - Pattern `src/test` no longer matches `src/test2` (incorrect behavior)
  - Now correctly validates path component boundaries
- Fixed escaped negation handling: patterns starting with `\!` now correctly match files with literal `!` character per gitignore specification
  - Previous behavior: `\!` was not properly processed as an escape sequence
  - New behavior: `\!important.txt` matches files literally named "!important.txt"

### Added
- **Major Feature:** Full support for root-relative patterns with leading `/`
  - `/pattern` matches only at repository root
  - `pattern` matches at any directory level
  - `/dir/*.txt` matches .txt files only in root-level dir/
  - `/dir/**` matches everything only in root-level dir/
- Support for escaped negation patterns (`\!`)
- Comprehensive test coverage for root-relative patterns with wildcards
- Test coverage for Unicode/non-ASCII filenames (Japanese, Russian, Emoji)
- Test coverage for very deep directory paths (100+ levels)
- Test coverage for very long patterns (1000+ characters)
- Test coverage for consecutive wildcard patterns (`*?*`, `?*?`, etc.)
- New tests: `TestLeadingSlashPatterns`, `TestRootRelativeWithWildcards`, `TestSubstringMatchingBug`, `TestEscapedNegation`, `TestUnicodePatterns`, `TestVeryDeepPaths`, `TestConsecutiveWildcards`, `TestVeryLongPatterns`, `TestEdgeCasePatterns`

### Removed
- Internal unused `NormalizePath()` function (non-breaking change, never exported)

### Changed
- Improved code documentation and comments
- Optimized string boundary checking for better performance
- Test count increased from 47 to 61 tests (+29%)
- Now fully compliant with gitignore specification

### Performance
- No regressions: maintains ~34µs per match operation
- Memory usage: 3,749 bytes/op (unchanged)
- Allocations: 148 allocs/op (unchanged)

### Breaking Changes
This is a major version bump due to:
1. Root-relative pattern support (was completely broken, now works)
2. Substring matching fix (changes matching behavior for edge cases)
3. Users who created workarounds for these bugs may need to update their patterns

## [1.2.0] - 2025-02-09 (SUPERSEDED by v2.0.0)

### Fixed
- **Critical:** Fixed substring matching bug where patterns with path separators incorrectly matched files using substring logic instead of proper path boundary checking
  - Pattern `src/test` no longer matches `mysrc/test` (incorrect behavior)
  - Pattern `src/test` no longer matches `src/test2` (incorrect behavior)
  - Now correctly validates path component boundaries
- Fixed escaped negation handling: patterns starting with `\!` now correctly match files with literal `!` character per gitignore specification
  - Previous behavior: `\!` was not properly processed as an escape sequence
  - New behavior: `\!important.txt` matches files literally named "!important.txt"

### Added
- Support for escaped negation patterns (`\!`)
- Comprehensive test coverage for Unicode/non-ASCII filenames (Japanese, Russian, Emoji)
- Test coverage for very deep directory paths (100+ levels)
- Test coverage for very long patterns (1000+ characters)
- Test coverage for consecutive wildcard patterns (`*?*`, `?*?`, etc.)
- New tests: `TestSubstringMatchingBug`, `TestEscapedNegation`, `TestUnicodePatterns`, `TestVeryDeepPaths`, `TestConsecutiveWildcards`, `TestVeryLongPatterns`, `TestEdgeCasePatterns`

### Removed
- Internal unused `NormalizePath()` function (non-breaking change, never exported)

### Changed
- Improved code documentation and comments
- Optimized string boundary checking for better performance
- Test count increased from 47 to 57 tests (+21%)

### Performance
- No regressions: maintains ~34µs per match operation
- Memory usage: 3,749 bytes/op (unchanged)
- Allocations: 148 allocs/op (unchanged)

## [1.1.1] - [Previous Date]

[Previous release notes...]

---

## Upgrade Guide

### From v1.1.x to v1.2.0

**Most users:** No action required - this is a drop-in replacement.

**If you relied on substring matching (unlikely):**
Review patterns containing `/` to ensure they match your intent. The previous substring behavior was a bug that violated gitignore specification.

Example:
```go
// Before v1.2.0 (buggy behavior):
pattern := "src/test"
Matches("mysrc/test")  // returned true (incorrect)

// After v1.2.0 (correct behavior):
pattern := "src/test"
Matches("mysrc/test")  // returns false (correct)
Matches("src/test")    // returns true
Matches("foo/src/test")  // returns true
```

If you need substring matching, use wildcards explicitly:
```go
pattern := "*src/test"  // Explicitly match anywhere
```
