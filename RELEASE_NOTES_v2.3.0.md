## v2.3.0 - 2026-08-30

### Added
- Feature: `RepositoryMatcher.Walk(fn WalkFunc) error`, an ignore-aware directory traversal.
  - Mirrors `filepath.WalkDir`'s callback shape (`fs.SkipDir`/`fs.SkipAll` control flow, filesystem errors delivered via the `err` parameter).
  - Adds an `ignored bool` reporting whether each entry is excluded by `.gitignore` rules or `RepositoryConfig.SkipFolders`.
  - A directory excluded either way is reported once with `ignored=true` and never descended into.
  - Honors the same `MaxDepth` and `FollowSymlinks` settings the `RepositoryMatcher` was constructed with.

### Performance
- Removed all hot-path allocations from `PatternMatcher` matching: 34184 ns/op, 3762 B/op, 148 allocs/op → 25317 ns/op, 0 B/op, 0 allocs/op.
- Removed hot-path allocations from `RepositoryMatcher.Matches`: 48 allocs/op, 3065 B/op → 21 allocs/op, 2044 B/op.

### Changed
- Corrected `doc.go`'s Performance section, which claimed "no allocations during regex matching" before that was actually true.

### Source
- Commits: `618c474`..`7493de0`
