# Complete Release Process for v2.0.0

This guide ensures your release properly notifies users about deprecation and critical bug fixes through Go's module system.

---

## ✅ Pre-Release Checklist

- [x] All 61 tests passing
- [x] No race conditions
- [x] Benchmarks successful
- [x] Critical bugs fixed (3 bugs)
- [x] Issue #5 resolved
- [x] go.mod retract directive added
- [x] Package documentation updated
- [x] README updated with warnings
- [x] Migration guide created
- [x] Version check script created

---

## 📦 Go Module Release Best Practices

### 1. Verify go.mod Retract Directive

The `retract` directive in go.mod ensures users see warnings:

```bash
# Verify retract is present
grep -A 1 "retract" go.mod
```

Expected output:
```
retract (
    [v1.0.0, v1.1.1] // Critical bugs: substring matching, root-relative patterns broken, no escaped negation support. Fixed in v2.0.0
)
```

### 2. Test Retraction Warning

```bash
# Test that retraction warning appears
go list -m -retracted github.com/codeglyph/go-dotignore@v1.1.1

# Should show retraction message
```

### 3. Verify Module Documentation

```bash
# Check package documentation
go doc github.com/codeglyph/go-dotignore

# Should show deprecation warning in package comment
```

---

## 🚀 Release Steps

### Step 1: Final Code Review

```bash
# Run all tests
go test ./... -race -count=1

# Run benchmarks
go test -bench=. -benchmem

# Check for issues
go vet ./...
golangci-lint run  # if available

# Format code
go fmt ./...
```

### Step 2: Update Version References

Make sure all documentation references v2.0.0:
- [x] README.md
- [x] MIGRATION.md
- [x] doc.go
- [x] dotignore.go package comment
- [x] RELEASE_NOTES_v2.0.0.md
- [x] CHANGELOG.md

### Step 3: Commit All Changes

```bash
# Add all release files
git add .

# Create release commit
git commit -m "feat!: release v2.0.0 with critical bug fixes

BREAKING CHANGE: Root-relative patterns now work correctly

Fixed three critical bugs:
- Root-relative patterns (/pattern) now match only at repository root (Issue #5)
- Fixed substring matching bug causing false positives
- Added support for escaped negation (\\!)

Retracted versions v1.0.0-v1.1.1 due to critical bugs.
All users should upgrade to v2.0.0 immediately.

Closes #5"
```

### Step 4: Create Git Tag with Detailed Message

```bash
# Create annotated tag with full details
git tag -a v2.0.0 -m "Release v2.0.0 - Full gitignore Specification Support

This major release fixes three critical bugs and achieves full
gitignore specification compliance.

CRITICAL BUG FIXES:
==================

1. Root-Relative Pattern Support (Issue #5)
   - Patterns starting with / now correctly match only at repository root
   - /build/ matches build/ but NOT src/build/
   - BEFORE: Root-relative patterns didn't work at all
   - AFTER: Full compliance with gitignore specification

2. Substring Matching Bug Fixed
   - Pattern src/test no longer matches mysrc/test or src/test2
   - Now uses proper path boundary checking
   - BEFORE: False positive matches with substring logic
   - AFTER: Correct boundary-aware matching

3. Escaped Negation Support Added
   - Pattern \\!important.txt now matches files named !important.txt
   - Full support for escaping special characters
   - BEFORE: No support for escaped negation
   - AFTER: Complete gitignore escape sequence support

NEW FEATURES:
=============
- Root-relative pattern support with leading /
- Escaped negation patterns with \\!
- 14 new comprehensive tests (+29% coverage)
- Full gitignore specification compliance

TESTING:
========
- 61 total tests (was 47), all passing
- No race conditions detected
- Performance: ~34µs per match (no regressions)
- Cross-platform: Linux, macOS, Windows

BREAKING CHANGES:
=================
This is a major version bump (v2.0.0) because:
1. Root-relative patterns now work (were completely broken)
2. Substring matching behavior changed (was a critical bug)
3. Users with workarounds may need minor updates

MIGRATION:
==========
For most users: No code changes required.
Patterns that were correct now just work properly.

If you worked around bugs in v1.x, remove workarounds:
  BEFORE: [\"mydir/\", \"!subdir/mydir/\", \"!other/mydir/\"]
  AFTER:  [\"/mydir/\"]  # Now works correctly

See MIGRATION.md for full details.

RETRACTION:
===========
Versions v1.0.0-v1.1.1 are retracted due to critical bugs.
All users should upgrade to v2.0.0 immediately.

go get github.com/codeglyph/go-dotignore/v2@latest

CLOSES:
=======
Closes #5

Full changelog: https://github.com/codeglyph/go-dotignore/blob/v2.0.0/CHANGELOG.md"
```

### Step 5: Push Everything

```bash
# Push main branch
git push origin main

# Push the tag
git push origin v2.0.0

# Verify tag is pushed
git ls-remote --tags origin | grep v2.0.0
```

### Step 6: Create GitHub Release

**Option A: Using GitHub CLI (Recommended)**

```bash
gh release create v2.0.0 \
  --title "v2.0.0 - Full gitignore Specification Support" \
  --notes-file GITHUB_RELEASE_v2.0.0.md \
  --latest
```

**Option B: Web UI**

1. Go to https://github.com/codeglyph/go-dotignore/releases/new
2. Choose tag: `v2.0.0`
3. Title: `v2.0.0 - Full gitignore Specification Support`
4. Description: Copy from `GITHUB_RELEASE_v2.0.0.md`
5. Check "Set as the latest release"
6. Click "Publish release"

### Step 7: Close Issue #5

```bash
gh issue comment 5 --body "🎉 **FIXED in v2.0.0!**

Root-relative patterns now work correctly per gitignore specification!

## What Changed

✅ \`/build/\` now matches **only** root-level build/, not src/build/
✅ \`/test.txt\` now matches **only** root-level test.txt
✅ Full gitignore specification compliance

## No More Workarounds Needed

**Before (workaround):**
\`\`\`go
patterns := []string{
    \"mydir/\",
    \"!example/mydir/\",
    \"!other/mydir/\",
}
\`\`\`

**After (proper solution):**
\`\`\`go
patterns := []string{
    \"/mydir/\",  // Automatically matches only at root
}
\`\`\`

## Upgrade

\`\`\`bash
go get github.com/codeglyph/go-dotignore/v2@latest
\`\`\`

## Release Notes

📖 **Full details:** https://github.com/codeglyph/go-dotignore/releases/tag/v2.0.0
📖 **Migration guide:** https://github.com/codeglyph/go-dotignore/blob/main/MIGRATION.md

Thanks for reporting this issue! 🙏"

# Close the issue
gh issue close 5
```

### Step 8: Verify Go Module System

```bash
# Check that pkg.go.dev picks up the new version (may take 15-30 min)
# Visit: https://pkg.go.dev/github.com/codeglyph/go-dotignore@v2.0.0

# Verify retraction is visible
go list -m -retracted -versions github.com/codeglyph/go-dotignore

# Should show retracted versions with reason
```

### Step 9: Test User Experience

Create a test project to verify the upgrade experience:

```bash
# Create test directory
mkdir -p /tmp/test-upgrade
cd /tmp/test-upgrade

# Initialize module
go mod init test

# Try to use retracted version (should show warning)
go get github.com/codeglyph/go-dotignore@v1.1.1

# Should display retraction warning:
# "module github.com/codeglyph/go-dotignore@v1.1.1: retracted by module author:
#  Critical bugs: substring matching, root-relative patterns broken, no escaped negation support. Fixed in v2.0.0"

# Upgrade to v2.0.0 (should work smoothly)
go get github.com/codeglyph/go-dotignore/v2@latest

# Cleanup
cd -
rm -rf /tmp/test-upgrade
```

---

## 📢 Post-Release Announcements

### 1. Update README Badge

Add to README.md:
```markdown
![GitHub release](https://img.shields.io/github/v/release/codeglyph/go-dotignore)
```

### 2. Social Media / Reddit

```
🎉 go-dotignore v2.0.0 released!

Major update fixing 3 critical bugs including the long-awaited root-relative pattern support (Issue #5)!

✅ /pattern now matches only at repository root
✅ Fixed substring matching bug
✅ Added escaped negation support (\!)

Versions v1.0.0-v1.1.1 are retracted. Upgrade now:
go get github.com/codeglyph/go-dotignore/v2@latest

Full details: https://github.com/codeglyph/go-dotignore/releases/tag/v2.0.0

#golang #opensource
```

### 3. Go Community

Post on:
- [ ] r/golang subreddit
- [ ] Gophers Slack (#libraries channel)
- [ ] Go Forum (https://forum.golangbridge.org/)
- [ ] Twitter/X with #golang hashtag

### 4. Dependent Projects

If you know of projects using go-dotignore:
- Open issues or PRs to notify them
- Mention the critical bug fixes
- Link to migration guide

---

## 🔍 Post-Release Monitoring

### Week 1:
- [ ] Monitor GitHub Issues for bug reports
- [ ] Check pkg.go.dev for correct documentation
- [ ] Verify retraction warnings appear correctly
- [ ] Watch for Dependabot/Renovate PRs in other projects

### Week 2-4:
- [ ] Track download statistics
- [ ] Monitor for edge cases not covered by tests
- [ ] Update documentation based on user feedback

---

## 📊 Success Metrics

Track these to measure release success:

1. **Adoption Rate**
   ```bash
   # Check download stats on pkg.go.dev
   # Monitor GitHub star/fork trends
   ```

2. **Issue Reports**
   - No critical bugs in first week = success
   - Quick resolution of any minor issues

3. **Retraction Effectiveness**
   - Users should see warnings when using v1.x
   - Migration should be smooth (check feedback)

4. **Community Response**
   - Positive feedback on bug fixes
   - Issue #5 closure celebrated

---

## 🆘 Rollback Plan (Emergency Only)

If critical issues are found in v2.0.0:

```bash
# 1. DO NOT delete v2.0.0 tag (breaks users)

# 2. Create hotfix v2.0.1 instead:
git checkout -b hotfix/v2.0.1

# Fix the issue
# ... make changes ...

# Commit and tag
git commit -m "fix: critical hotfix for v2.0.0"
git tag -a v2.0.1 -m "Hotfix for v2.0.0"
git push origin hotfix/v2.0.1
git push origin v2.0.1

# 3. Create GitHub release for v2.0.1

# 4. Optionally retract v2.0.0 if severe:
# Add to go.mod:
# retract v2.0.0 // Critical issue fixed in v2.0.1
```

**Never delete or force-push tags in production!**

---

## ✅ Final Checklist

Before announcing:
- [ ] All tests pass
- [ ] Tag pushed to GitHub
- [ ] GitHub release created
- [ ] Issue #5 closed
- [ ] pkg.go.dev shows v2.0.0
- [ ] Retraction warning works for v1.x
- [ ] Documentation is correct
- [ ] Migration guide is clear

---

## 🎉 You're Done!

Your v2.0.0 release is complete with proper Go module deprecation warnings!

**What users will see:**

1. **Using v1.x**: Retraction warning prompting upgrade
2. **Upgrading to v2.0.0**: Clear migration guide and release notes
3. **In their editor**: Package doc shows version warning
4. **On pkg.go.dev**: Full documentation with deprecation notice

**Commands for quick reference:**

```bash
# One-liner release (if all files committed):
git push origin main && git push origin v2.0.0 && \
gh release create v2.0.0 --title "v2.0.0 - Full gitignore Specification Support" --notes-file GITHUB_RELEASE_v2.0.0.md --latest && \
gh issue close 5 --comment "Fixed in v2.0.0! 🎉"
```
