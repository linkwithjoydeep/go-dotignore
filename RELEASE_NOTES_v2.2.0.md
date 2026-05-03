## v2.2.0 - 2026-05-03

### Added
- Feature ([#6](https://github.com/linkwithjoydeep/go-dotignore/issues/6)): `RepositoryConfig.SkipFolders` to skip selected directories while discovering nested ignore files.
- Internal `Contains()` helper used by repository traversal skip checks.
- Godoc coverage for `Contains()`.

### Changed
- Dropped `slices` dependency usage for skip-folder checks in repository scanning path.

### Credits
- Thanks to [@Marcel2603](https://github.com/Marcel2603) for PR [#6](https://github.com/linkwithjoydeep/go-dotignore/pull/6).

### Source
- Commit: `fe7feef51cd11cd936088946265aeee8d7230c49`
- Title: `feat(repository-matcher): Add skipfolders, to skip scanning these folders (#6)`
