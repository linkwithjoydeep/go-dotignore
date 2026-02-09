# Release v1.2.0 Checklist

## 📋 Pre-Release Checklist

- [x] All tests passing (57/57 tests ✅)
- [x] No race conditions detected
- [x] Benchmarks run successfully
- [x] Critical bugs fixed
- [x] Documentation updated
- [x] Release notes prepared

## 📝 Release Documents Created

1. **CHANGELOG.md** - Permanent changelog following Keep a Changelog format
2. **RELEASE_NOTES_v1.2.0.md** - Detailed release notes with migration guide
3. **GITHUB_RELEASE_v1.2.0.md** - Concise GitHub release description

## 🚀 Release Steps

### 1. Review Changes

```bash
# View all changes since last release
git diff v1.1.1 HEAD

# Check current status
git status
```

### 2. Commit Release Documents (if not already committed)

```bash
git add CHANGELOG.md RELEASE_NOTES_v1.2.0.md GITHUB_RELEASE_v1.2.0.md
git commit -m "docs: add release notes for v1.2.0"
```

### 3. Create and Push Git Tag

```bash
# Create annotated tag
git tag -a v1.2.0 -m "Release v1.2.0 - Bug fixes and enhanced testing"

# Push tag to remote
git push origin v1.2.0

# Push commits (if any)
git push origin main
```

### 4. Create GitHub Release

**Option A: Using GitHub CLI (gh)**
```bash
gh release create v1.2.0 \
  --title "v1.2.0 - Bug Fixes & Enhanced Testing" \
  --notes-file GITHUB_RELEASE_v1.2.0.md
```

**Option B: Using GitHub Web UI**
1. Go to https://github.com/codeglyph/go-dotignore/releases/new
2. Select tag: `v1.2.0`
3. Title: `v1.2.0 - Bug Fixes & Enhanced Testing`
4. Copy content from `GITHUB_RELEASE_v1.2.0.md` into description
5. Check "Set as the latest release"
6. Click "Publish release"

### 5. Verify Release

```bash
# Check that Go can fetch the new version
go list -m github.com/codeglyph/go-dotignore@v1.2.0

# Verify on pkg.go.dev (may take a few minutes)
# Visit: https://pkg.go.dev/github.com/codeglyph/go-dotignore@v1.2.0
```

### 6. Announce (Optional)

Consider announcing on:
- [ ] Project README (add badge for latest version)
- [ ] GitHub Discussions (if enabled)
- [ ] Social media (Twitter, Reddit r/golang, etc.)
- [ ] Go community Slack/Discord

## 📊 Quick Stats for Announcement

- **Critical bugs fixed:** 2
- **New tests added:** 10 (+21% coverage)
- **Total tests:** 57 (all passing ✅)
- **Performance:** No regressions
- **Breaking changes:** None for correct usage

## 🔍 Post-Release Verification

After releasing, monitor for:
- [ ] GitHub Issues for reports of problems
- [ ] pkg.go.dev shows new version (can take 15-30 minutes)
- [ ] Dependabot/Renovate PRs in dependent projects
- [ ] Any unexpected behavior reports

## 🆘 Rollback Plan (If Needed)

If critical issues are discovered:

```bash
# Delete the tag from remote
git push --delete origin v1.2.0

# Delete the tag locally
git tag -d v1.2.0

# Delete GitHub release (via web UI or gh CLI)
gh release delete v1.2.0

# Release hotfix as v1.2.1
```

## 📞 Communication Template

**For dependent projects:**

```
go-dotignore v1.2.0 has been released with important bug fixes:

✅ Fixed critical substring matching bug
✅ Added escaped negation support (\!)
✅ 21% more test coverage
✅ No breaking changes for correct usage

Upgrade: go get github.com/codeglyph/go-dotignore@v1.2.0

Full details: https://github.com/codeglyph/go-dotignore/releases/tag/v1.2.0
```

---

## ✅ All Set!

Your library is production-ready for v1.2.0 release. The release documents are prepared and ready to publish.

**Recommended release command:**
```bash
git tag -a v1.2.0 -m "Release v1.2.0 - Bug fixes and enhanced testing" && \
git push origin v1.2.0 && \
gh release create v1.2.0 --title "v1.2.0 - Bug Fixes & Enhanced Testing" --notes-file GITHUB_RELEASE_v1.2.0.md
```
