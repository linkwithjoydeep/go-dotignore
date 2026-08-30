package dotignore

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Helper function to create a test directory structure with .gitignore files
func createTestRepo(t *testing.T, structure map[string]string) string {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "dotignore-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	for path, content := range structure {
		fullPath := filepath.Join(tmpDir, path)

		// Create parent directories
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}

		// Write file
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("failed to write file %s: %v", fullPath, err)
		}
	}

	return tmpDir
}

func TestNewRepositoryMatcher(t *testing.T) {
	tests := []struct {
		name      string
		structure map[string]string
		wantErr   bool
		wantCount int
	}{
		{
			name: "single root .gitignore",
			structure: map[string]string{
				".gitignore": "*.log\ntemp/\n",
			},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name: "nested .gitignore files",
			structure: map[string]string{
				".gitignore":          "*.log\n",
				"frontend/.gitignore": "node_modules/\ndist/\n",
				"backend/.gitignore":  "target/\n*.class\n",
			},
			wantErr:   false,
			wantCount: 3,
		},
		{
			name: "deeply nested .gitignore",
			structure: map[string]string{
				".gitignore":         "*.log\n",
				"a/.gitignore":       "*.tmp\n",
				"a/b/.gitignore":     "*.cache\n",
				"a/b/c/.gitignore":   "*.test\n",
				"a/b/c/d/.gitignore": "*.debug\n",
			},
			wantErr:   false,
			wantCount: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := createTestRepo(t, tt.structure)
			defer os.RemoveAll(tmpDir)

			matcher, err := NewRepositoryMatcher(tmpDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRepositoryMatcher() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if matcher == nil {
					t.Fatal("expected matcher to be non-nil")
				}

				if got := matcher.IgnoreFileCount(); got != tt.wantCount {
					t.Errorf("IgnoreFileCount() = %v, want %v", got, tt.wantCount)
				}

				if matcher.RootDir() != tmpDir {
					t.Errorf("RootDir() = %v, want %v", matcher.RootDir(), tmpDir)
				}
			}
		})
	}
}

func TestNewRepositoryMatcher_Errors(t *testing.T) {
	tests := []struct {
		name    string
		rootDir string
		wantErr bool
	}{
		{
			name:    "empty root dir",
			rootDir: "",
			wantErr: true,
		},
		{
			name:    "non-existent directory",
			rootDir: "/path/that/does/not/exist",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewRepositoryMatcher(tt.rootDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRepositoryMatcher() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRepositoryMatcher_Matches_SimpleHierarchy(t *testing.T) {
	structure := map[string]string{
		".gitignore":          "*.log\ntemp/\n",
		"frontend/.gitignore": "node_modules/\ndist/\n",
	}

	tmpDir := createTestRepo(t, structure)
	defer os.RemoveAll(tmpDir)

	matcher, err := NewRepositoryMatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewRepositoryMatcher() failed: %v", err)
	}

	tests := []struct {
		path string
		want bool
	}{
		// Root patterns
		{"app.log", true},
		{"debug.log", true},
		{"temp/cache.txt", true},
		{"temp/data.json", true},

		// Frontend patterns
		{"frontend/node_modules/package.json", true},
		{"frontend/dist/bundle.js", true},
		{"frontend/src/app.js", false},

		// Root patterns apply to subdirectories
		{"frontend/debug.log", true},
		{"backend/app.log", true},

		// Not ignored
		{"README.md", false},
		{"frontend/package.json", false},
		{"backend/main.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, err := matcher.Matches(tt.path)
			if err != nil {
				t.Errorf("Matches(%q) error: %v", tt.path, err)
				return
			}
			if got != tt.want {
				t.Errorf("Matches(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestRepositoryMatcher_Matches_Negation(t *testing.T) {
	structure := map[string]string{
		".gitignore":      "*.log\n!important.log\n",
		"logs/.gitignore": "!debug.log\n",
	}

	tmpDir := createTestRepo(t, structure)
	defer os.RemoveAll(tmpDir)

	matcher, err := NewRepositoryMatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewRepositoryMatcher() failed: %v", err)
	}

	tests := []struct {
		path string
		want bool
	}{
		// Root level - negation applies
		{"app.log", true},
		{"important.log", false}, // negated by root .gitignore

		// Logs directory - local negation
		{"logs/app.log", true},
		{"logs/debug.log", false}, // negated by logs/.gitignore
		{"logs/error.log", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, err := matcher.Matches(tt.path)
			if err != nil {
				t.Errorf("Matches(%q) error: %v", tt.path, err)
				return
			}
			if got != tt.want {
				t.Errorf("Matches(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestRepositoryMatcher_Matches_MonorepoScenario(t *testing.T) {
	// Real-world monorepo structure from issue #4
	structure := map[string]string{
		".gitignore": `# Global ignores
*.log
.DS_Store
.env
`,
		"frontend/.gitignore": `# Frontend ignores
node_modules/
dist/
.cache/
*.local.js
`,
		"backend/.gitignore": `# Backend ignores
target/
*.class
logs/
`,
		"docs/.gitignore": `# Docs ignores
_build/
*.pyc
`,
	}

	tmpDir := createTestRepo(t, structure)
	defer os.RemoveAll(tmpDir)

	matcher, err := NewRepositoryMatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewRepositoryMatcher() failed: %v", err)
	}

	tests := []struct {
		path string
		want bool
		desc string
	}{
		// Global patterns apply everywhere
		{"app.log", true, "root .log file"},
		{"frontend/debug.log", true, "frontend .log file"},
		{"backend/app.log", true, "backend .log file (also in backend/.gitignore)"},
		{".DS_Store", true, "root .DS_Store"},
		{"frontend/.DS_Store", true, "frontend .DS_Store"},

		// Frontend-specific
		{"frontend/node_modules/package.json", true, "frontend node_modules"},
		{"frontend/dist/bundle.js", true, "frontend dist"},
		{"frontend/.cache/data.json", true, "frontend cache"},
		{"frontend/config.local.js", true, "frontend local file"},
		{"frontend/src/App.js", false, "frontend source file"},

		// Backend-specific
		{"backend/target/classes/Main.class", true, "backend target dir"},
		{"backend/App.class", true, "backend .class file"},
		{"backend/logs/error.log", true, "backend logs dir"},
		{"backend/src/main.go", false, "backend source file"},

		// Docs-specific
		{"docs/_build/html/index.html", true, "docs build dir"},
		{"docs/config.pyc", true, "docs .pyc file"},
		{"docs/index.rst", false, "docs source file"},

		// Not ignored
		{"README.md", false, "root README"},
		{"frontend/package.json", false, "frontend package.json"},
		{"backend/Cargo.toml", false, "backend config"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := matcher.Matches(tt.path)
			if err != nil {
				t.Errorf("Matches(%q) error: %v", tt.path, err)
				return
			}
			if got != tt.want {
				t.Errorf("Matches(%q) = %v, want %v (%s)", tt.path, got, tt.want, tt.desc)
			}
		})
	}
}

func TestRepositoryMatcher_Matches_OverrideParentPatterns(t *testing.T) {
	// Test that child .gitignore can override parent patterns
	structure := map[string]string{
		".gitignore":         "*.txt\n",
		"special/.gitignore": "!important.txt\n",
	}

	tmpDir := createTestRepo(t, structure)
	defer os.RemoveAll(tmpDir)

	matcher, err := NewRepositoryMatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewRepositoryMatcher() failed: %v", err)
	}

	tests := []struct {
		path string
		want bool
		desc string
	}{
		{"file.txt", true, "root .txt ignored"},
		{"data.txt", true, "root .txt ignored"},
		{"special/file.txt", true, "special/ .txt still ignored"},
		{"special/important.txt", false, "special/important.txt negated"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := matcher.Matches(tt.path)
			if err != nil {
				t.Errorf("Matches(%q) error: %v", tt.path, err)
				return
			}
			if got != tt.want {
				t.Errorf("Matches(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestRepositoryMatcher_Matches_RootRelativePatterns(t *testing.T) {
	// Test root-relative patterns in nested .gitignore files
	structure := map[string]string{
		".gitignore":     "/build/\nconfig/\n",
		"src/.gitignore": "/test/\n",
	}

	tmpDir := createTestRepo(t, structure)
	defer os.RemoveAll(tmpDir)

	matcher, err := NewRepositoryMatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewRepositoryMatcher() failed: %v", err)
	}

	tests := []struct {
		path string
		want bool
		desc string
	}{
		// Root-relative /build/ only matches at repo root
		{"build/output.js", true, "root build dir"},
		{"src/build/test.js", false, "nested build dir not matched by /build/"},

		// Non-root-relative config/ matches anywhere
		{"config/app.json", true, "root config dir"},
		{"src/config/test.json", true, "nested config dir"},

		// src/.gitignore /test/ only matches relative to src/
		{"src/test/unit.js", true, "src/test/ matched by src/.gitignore"},
		{"test/integration.js", false, "root test/ not matched by src/.gitignore /test/"},
		{"src/lib/test/helper.js", false, "src/lib/test/ not matched by /test/ in src/.gitignore"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := matcher.Matches(tt.path)
			if err != nil {
				t.Errorf("Matches(%q) error: %v", tt.path, err)
				return
			}
			if got != tt.want {
				t.Errorf("Matches(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestRepositoryMatcher_Matches_AbsolutePaths(t *testing.T) {
	structure := map[string]string{
		".gitignore": "*.log\n",
	}

	tmpDir := createTestRepo(t, structure)
	defer os.RemoveAll(tmpDir)

	matcher, err := NewRepositoryMatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewRepositoryMatcher() failed: %v", err)
	}

	// Test with absolute path
	absPath := filepath.Join(tmpDir, "app.log")
	got, err := matcher.Matches(absPath)
	if err != nil {
		t.Errorf("Matches() error: %v", err)
	}
	if !got {
		t.Errorf("Matches(%q) = false, want true", absPath)
	}

	// Test path outside repository
	outsidePath := "/tmp/outside.log"
	_, err = matcher.Matches(outsidePath)
	if err == nil {
		t.Error("expected error for path outside repository")
	}
}

func TestRepositoryMatcher_IgnoreFilePaths(t *testing.T) {
	structure := map[string]string{
		".gitignore":          "*.log\n",
		"frontend/.gitignore": "node_modules/\n",
		"backend/.gitignore":  "target/\n",
	}

	tmpDir := createTestRepo(t, structure)
	defer os.RemoveAll(tmpDir)

	matcher, err := NewRepositoryMatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewRepositoryMatcher() failed: %v", err)
	}

	paths := matcher.IgnoreFilePaths()
	if len(paths) != 3 {
		t.Errorf("IgnoreFilePaths() returned %d paths, want 3", len(paths))
	}

	// Check that paths are relative to root
	expectedPaths := map[string]bool{
		".gitignore":          true,
		"frontend/.gitignore": true,
		"backend/.gitignore":  true,
	}

	for _, path := range paths {
		if !expectedPaths[filepath.ToSlash(path)] {
			t.Errorf("unexpected path in IgnoreFilePaths(): %s", path)
		}
	}
}

func TestRepositoryMatcherWithConfig_MaxDepth(t *testing.T) {
	structure := map[string]string{
		".gitignore":       "*.log\n",
		"a/.gitignore":     "*.tmp\n",
		"a/b/.gitignore":   "*.cache\n",
		"a/b/c/.gitignore": "*.test\n",
	}

	tmpDir := createTestRepo(t, structure)
	defer os.RemoveAll(tmpDir)

	config := &RepositoryConfig{
		IgnoreFileName: ".gitignore",
		MaxDepth:       2,
	}

	matcher, err := NewRepositoryMatcherWithConfig(tmpDir, config)
	if err != nil {
		t.Fatalf("NewRepositoryMatcherWithConfig() failed: %v", err)
	}

	// Should only load .gitignore files up to depth 2
	// Root (depth 0), a/ (depth 1), a/b/ (depth 2)
	// Should NOT load a/b/c/.gitignore (depth 3)
	count := matcher.IgnoreFileCount()
	if count != 3 {
		t.Errorf("with MaxDepth=2, got %d ignore files, want 3", count)
	}
}

func TestRepositoryMatcherWithConfig_CustomIgnoreFileName(t *testing.T) {
	structure := map[string]string{
		".ignore":     "*.log\n",
		"src/.ignore": "*.tmp\n",
	}

	tmpDir := createTestRepo(t, structure)
	defer os.RemoveAll(tmpDir)

	config := &RepositoryConfig{
		IgnoreFileName: ".ignore",
	}

	matcher, err := NewRepositoryMatcherWithConfig(tmpDir, config)
	if err != nil {
		t.Fatalf("NewRepositoryMatcherWithConfig() failed: %v", err)
	}

	if count := matcher.IgnoreFileCount(); count != 2 {
		t.Errorf("got %d ignore files, want 2", count)
	}

	// Verify the patterns work
	got, err := matcher.Matches("app.log")
	if err != nil {
		t.Errorf("Matches() error: %v", err)
	}
	if !got {
		t.Error("Matches(app.log) = false, want true")
	}
}

func TestRepositoryMatcher_Matches_WildcardPatterns(t *testing.T) {
	structure := map[string]string{
		".gitignore":     "node_modules/\n**/*.test.js\n",
		"src/.gitignore": "*.tmp\n",
	}

	tmpDir := createTestRepo(t, structure)
	defer os.RemoveAll(tmpDir)

	matcher, err := NewRepositoryMatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewRepositoryMatcher() failed: %v", err)
	}

	tests := []struct {
		path string
		want bool
	}{
		// node_modules/ pattern (matches at any level)
		{"node_modules/pkg/index.js", true},
		{"frontend/node_modules/pkg/index.js", true},

		// **/*.test.js patterns from root
		{"app.test.js", true},
		{"src/components/Button.test.js", true},
		{"tests/integration/api.test.js", true},

		// src/*.tmp from src/.gitignore
		{"src/cache.tmp", true},
		{"src/build/output.tmp", true},

		// Not matched
		{"src/App.js", false},
		{"README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, err := matcher.Matches(tt.path)
			if err != nil {
				t.Errorf("Matches(%q) error: %v", tt.path, err)
				return
			}
			if got != tt.want {
				t.Errorf("Matches(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestRepositoryMatcherWithConfig_SkipFolders(t *testing.T) {
	structure := map[string]string{
		".gitignore":            "*.log\n",
		"vendor/.gitignore":     "*.tmp\n",
		"vendor/pkg/.gitignore": "*.cache\n",
		"src/.gitignore":        "*.test\n",
	}

	tmpDir := createTestRepo(t, structure)
	defer os.RemoveAll(tmpDir)

	config := &RepositoryConfig{
		IgnoreFileName: ".gitignore",
		SkipFolders:    []string{"vendor"},
	}

	matcher, err := NewRepositoryMatcherWithConfig(tmpDir, config)
	if err != nil {
		t.Fatalf("NewRepositoryMatcherWithConfig() failed: %v", err)
	}

	// vendor/ and vendor/pkg/ .gitignore files should not be loaded
	if count := matcher.IgnoreFileCount(); count != 2 {
		t.Errorf("got %d ignore files, want 2 (root + src)", count)
	}

	tests := []struct {
		path string
		want bool
		desc string
	}{
		{"app.log", true, "root pattern still applies"},
		{"vendor/app.log", true, "root pattern applies inside skipped folder"},
		{"vendor/cache.tmp", false, "vendor .gitignore not loaded - *.tmp not matched"},
		{"vendor/pkg/data.cache", false, "vendor/pkg .gitignore not loaded - *.cache not matched"},
		{"src/unit.test", true, "src .gitignore loaded normally"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := matcher.Matches(tt.path)
			if err != nil {
				t.Errorf("Matches(%q) error: %v", tt.path, err)
				return
			}
			if got != tt.want {
				t.Errorf("Matches(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestRepositoryMatcherWithConfig_SkipFolders_Multiple(t *testing.T) {
	structure := map[string]string{
		".gitignore":              "*.log\n",
		"vendor/.gitignore":       "*.tmp\n",
		"node_modules/.gitignore": "*.cache\n",
		"src/.gitignore":          "*.test\n",
	}

	tmpDir := createTestRepo(t, structure)
	defer os.RemoveAll(tmpDir)

	config := &RepositoryConfig{
		IgnoreFileName: ".gitignore",
		SkipFolders:    []string{"vendor", "node_modules"},
	}

	matcher, err := NewRepositoryMatcherWithConfig(tmpDir, config)
	if err != nil {
		t.Fatalf("NewRepositoryMatcherWithConfig() failed: %v", err)
	}

	// Only root and src should be loaded
	if count := matcher.IgnoreFileCount(); count != 2 {
		t.Errorf("got %d ignore files, want 2", count)
	}
}

func TestRepositoryMatcherWithConfig_SkipFolders_Empty(t *testing.T) {
	structure := map[string]string{
		".gitignore":        "*.log\n",
		"vendor/.gitignore": "*.tmp\n",
	}

	tmpDir := createTestRepo(t, structure)
	defer os.RemoveAll(tmpDir)

	config := &RepositoryConfig{
		IgnoreFileName: ".gitignore",
		SkipFolders:    []string{}, // empty - nothing skipped
	}

	matcher, err := NewRepositoryMatcherWithConfig(tmpDir, config)
	if err != nil {
		t.Fatalf("NewRepositoryMatcherWithConfig() failed: %v", err)
	}

	if count := matcher.IgnoreFileCount(); count != 2 {
		t.Errorf("got %d ignore files, want 2", count)
	}
}

func TestRepositoryMatcher_EmptyFile(t *testing.T) {
	structure := map[string]string{
		".gitignore":     "",
		"src/.gitignore": "*.tmp\n",
	}

	tmpDir := createTestRepo(t, structure)
	defer os.RemoveAll(tmpDir)

	matcher, err := NewRepositoryMatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewRepositoryMatcher() failed: %v", err)
	}

	// Empty .gitignore should still be loaded but have no patterns
	// We should have 2 files loaded (root and src)
	if count := matcher.IgnoreFileCount(); count < 1 {
		t.Errorf("got %d ignore files, want at least 1", count)
	}
}

type walkVisit struct {
	path    string
	isDir   bool
	ignored bool
	err     error
}

func collectWalk(t *testing.T, matcher *RepositoryMatcher, fn func(v walkVisit) error) []walkVisit {
	t.Helper()
	var visits []walkVisit
	err := matcher.Walk(func(path string, d fs.DirEntry, ignored bool, walkErr error) error {
		v := walkVisit{path: path, ignored: ignored, err: walkErr}
		if d != nil {
			v.isDir = d.IsDir()
		}
		visits = append(visits, v)
		if fn != nil {
			return fn(v)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Walk() error: %v", err)
	}
	return visits
}

func TestRepositoryMatcher_Walk_RootFirst(t *testing.T) {
	structure := map[string]string{
		".gitignore":  "*.log\n",
		"src/main.go": "package main\n",
	}
	tmpDir := createTestRepo(t, structure)
	defer os.RemoveAll(tmpDir)

	matcher, err := NewRepositoryMatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewRepositoryMatcher() failed: %v", err)
	}

	visits := collectWalk(t, matcher, nil)
	if len(visits) == 0 {
		t.Fatal("expected at least one visit")
	}
	if visits[0].path != tmpDir {
		t.Errorf("first visit = %q, want root %q", visits[0].path, tmpDir)
	}
	if visits[0].ignored {
		t.Errorf("root reported ignored=true, want false")
	}
}

func TestRepositoryMatcher_Walk_VisitsEveryEntryOnce(t *testing.T) {
	structure := map[string]string{
		".gitignore":    "*.log\n",
		"README.md":     "# readme\n",
		"src/main.go":   "package main\n",
		"src/app.log":   "log\n",
		"docs/index.md": "docs\n",
	}
	tmpDir := createTestRepo(t, structure)
	defer os.RemoveAll(tmpDir)

	matcher, err := NewRepositoryMatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewRepositoryMatcher() failed: %v", err)
	}

	visits := collectWalk(t, matcher, nil)

	seen := make(map[string]int)
	for _, v := range visits {
		if !filepath.IsAbs(v.path) {
			t.Errorf("visited path %q is not absolute", v.path)
		}
		seen[v.path]++
	}
	for path, count := range seen {
		if count != 1 {
			t.Errorf("path %q visited %d times, want 1", path, count)
		}
	}

	expected := []string{
		tmpDir,
		filepath.Join(tmpDir, ".gitignore"),
		filepath.Join(tmpDir, "README.md"),
		filepath.Join(tmpDir, "docs"),
		filepath.Join(tmpDir, "docs", "index.md"),
		filepath.Join(tmpDir, "src"),
		filepath.Join(tmpDir, "src", "app.log"),
		filepath.Join(tmpDir, "src", "main.go"),
	}
	for _, path := range expected {
		if _, ok := seen[path]; !ok {
			t.Errorf("expected path %q to be visited, but it wasn't", path)
		}
	}
}

func TestRepositoryMatcher_Walk_GitignoreMatchedDirectoryNotDescended(t *testing.T) {
	structure := map[string]string{
		".gitignore":                "node_modules/\n",
		"node_modules/pkg/index.js": "module.exports = {}\n",
		"src/main.go":               "package main\n",
	}
	tmpDir := createTestRepo(t, structure)
	defer os.RemoveAll(tmpDir)

	matcher, err := NewRepositoryMatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewRepositoryMatcher() failed: %v", err)
	}

	visits := collectWalk(t, matcher, nil)

	nodeModulesPath := filepath.Join(tmpDir, "node_modules")
	found := false
	for _, v := range visits {
		if v.path == nodeModulesPath {
			found = true
			if !v.ignored {
				t.Errorf("node_modules reported ignored=%v, want true", v.ignored)
			}
		}
		if strings.HasPrefix(v.path, nodeModulesPath+string(filepath.Separator)) {
			t.Errorf("entry inside ignored directory was visited: %q", v.path)
		}
	}
	if !found {
		t.Error("node_modules directory itself was never visited")
	}
}

func TestRepositoryMatcher_Walk_SkipFolders(t *testing.T) {
	structure := map[string]string{
		"vendor/lib.go": "package vendor\n",
		"src/main.go":   "package main\n",
	}
	tmpDir := createTestRepo(t, structure)
	defer os.RemoveAll(tmpDir)

	config := &RepositoryConfig{
		IgnoreFileName: ".gitignore",
		SkipFolders:    []string{"vendor"},
	}
	matcher, err := NewRepositoryMatcherWithConfig(tmpDir, config)
	if err != nil {
		t.Fatalf("NewRepositoryMatcherWithConfig() failed: %v", err)
	}

	visits := collectWalk(t, matcher, nil)

	vendorPath := filepath.Join(tmpDir, "vendor")
	found := false
	for _, v := range visits {
		if v.path == vendorPath {
			found = true
			if !v.ignored {
				t.Errorf("vendor reported ignored=%v, want true", v.ignored)
			}
		}
		if strings.HasPrefix(v.path, vendorPath+string(filepath.Separator)) {
			t.Errorf("entry inside skip-folder was visited: %q", v.path)
		}
	}
	if !found {
		t.Error("vendor directory itself was never visited")
	}
}

func TestRepositoryMatcher_Walk_NegationOverride(t *testing.T) {
	structure := map[string]string{
		".gitignore":         "*.log\n",
		"keep/.gitignore":    "!important.log\n",
		"app.log":            "log\n",
		"keep/app.log":       "log\n",
		"keep/important.log": "log\n",
	}
	tmpDir := createTestRepo(t, structure)
	defer os.RemoveAll(tmpDir)

	matcher, err := NewRepositoryMatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewRepositoryMatcher() failed: %v", err)
	}

	visits := collectWalk(t, matcher, nil)

	want := map[string]bool{
		filepath.Join(tmpDir, "app.log"):               true,
		filepath.Join(tmpDir, "keep", "app.log"):       true,
		filepath.Join(tmpDir, "keep", "important.log"): false,
	}
	got := make(map[string]bool)
	for _, v := range visits {
		if _, ok := want[v.path]; ok {
			got[v.path] = v.ignored
		}
	}
	for path, wantIgnored := range want {
		gotIgnored, ok := got[path]
		if !ok {
			t.Errorf("expected path %q to be visited, but it wasn't", path)
			continue
		}
		if gotIgnored != wantIgnored {
			t.Errorf("Walk ignored for %q = %v, want %v", path, gotIgnored, wantIgnored)
		}
	}
}

func TestRepositoryMatcher_Walk_MaxDepth(t *testing.T) {
	structure := map[string]string{
		".gitignore":        "*.log\n",
		"a/.gitignore":      "*.tmp\n",
		"a/file1.txt":       "1\n",
		"a/b/.gitignore":    "*.cache\n",
		"a/b/file2.txt":     "2\n",
		"a/b/c/.gitignore":  "*.test\n",
		"a/b/c/file3.txt":   "3\n",
		"a/b/c/d/file4.txt": "4\n",
	}
	tmpDir := createTestRepo(t, structure)
	defer os.RemoveAll(tmpDir)

	config := &RepositoryConfig{
		IgnoreFileName: ".gitignore",
		MaxDepth:       2,
	}
	matcher, err := NewRepositoryMatcherWithConfig(tmpDir, config)
	if err != nil {
		t.Fatalf("NewRepositoryMatcherWithConfig() failed: %v", err)
	}

	visits := collectWalk(t, matcher, nil)

	seen := make(map[string]bool)
	for _, v := range visits {
		seen[v.path] = true
	}

	// a/b/c is at depth 2 (== MaxDepth), so it and its direct file are still
	// visited; a/b/c/d is at depth 3 (> MaxDepth), so it is visited once but
	// not descended into.
	mustSee := []string{
		filepath.Join(tmpDir, "a", "b", "c"),
		filepath.Join(tmpDir, "a", "b", "c", "file3.txt"),
		filepath.Join(tmpDir, "a", "b", "c", "d"),
	}
	for _, path := range mustSee {
		if !seen[path] {
			t.Errorf("expected path %q to be visited, but it wasn't", path)
		}
	}

	mustNotSee := filepath.Join(tmpDir, "a", "b", "c", "d", "file4.txt")
	if seen[mustNotSee] {
		t.Errorf("path %q beyond MaxDepth was visited, want it skipped", mustNotSee)
	}
}

func TestRepositoryMatcher_Walk_Symlinks(t *testing.T) {
	structure := map[string]string{
		"real/file.txt": "content\n",
	}
	tmpDir := createTestRepo(t, structure)
	defer os.RemoveAll(tmpDir)

	linkPath := filepath.Join(tmpDir, "link")
	if err := os.Symlink(filepath.Join(tmpDir, "real"), linkPath); err != nil {
		t.Skipf("symlinks not supported on this platform: %v", err)
	}

	for _, followSymlinks := range []bool{false, true} {
		t.Run(fmt.Sprintf("FollowSymlinks=%v", followSymlinks), func(t *testing.T) {
			config := &RepositoryConfig{
				IgnoreFileName: ".gitignore",
				FollowSymlinks: followSymlinks,
			}
			matcher, err := NewRepositoryMatcherWithConfig(tmpDir, config)
			if err != nil {
				t.Fatalf("NewRepositoryMatcherWithConfig() failed: %v", err)
			}

			visits := collectWalk(t, matcher, nil)

			linkSeen := false
			for _, v := range visits {
				if v.path == linkPath {
					linkSeen = true
				}
				if v.path == filepath.Join(linkPath, "file.txt") {
					t.Errorf("symlink target was traversed via the link: %q", v.path)
				}
			}
			if !linkSeen {
				t.Error("symlink entry itself was never visited")
			}
		})
	}
}

func TestRepositoryMatcher_Walk_SkipDir(t *testing.T) {
	structure := map[string]string{
		"sub/a.txt": "a\n",
		"sub/b.txt": "b\n",
		"other.txt": "o\n",
	}
	tmpDir := createTestRepo(t, structure)
	defer os.RemoveAll(tmpDir)

	matcher, err := NewRepositoryMatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewRepositoryMatcher() failed: %v", err)
	}

	subPath := filepath.Join(tmpDir, "sub")
	visits := collectWalk(t, matcher, func(v walkVisit) error {
		if v.path == subPath {
			return fs.SkipDir
		}
		return nil
	})

	for _, v := range visits {
		if strings.HasPrefix(v.path, subPath+string(filepath.Separator)) {
			t.Errorf("entry inside SkipDir'd directory was visited: %q", v.path)
		}
	}
}

func TestRepositoryMatcher_Walk_SkipDirOnFileSkipsRemainingSiblings(t *testing.T) {
	structure := map[string]string{
		"a.txt": "a\n",
		"b.txt": "b\n",
		"c.txt": "c\n",
	}
	tmpDir := createTestRepo(t, structure)
	defer os.RemoveAll(tmpDir)

	matcher, err := NewRepositoryMatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewRepositoryMatcher() failed: %v", err)
	}

	bPath := filepath.Join(tmpDir, "b.txt")
	visits := collectWalk(t, matcher, func(v walkVisit) error {
		if v.path == bPath {
			return fs.SkipDir
		}
		return nil
	})

	seen := make(map[string]bool)
	for _, v := range visits {
		seen[v.path] = true
	}
	if !seen[filepath.Join(tmpDir, "a.txt")] {
		t.Error("a.txt should have been visited before b.txt")
	}
	if !seen[bPath] {
		t.Error("b.txt should have been visited")
	}
	if seen[filepath.Join(tmpDir, "c.txt")] {
		t.Error("c.txt should have been skipped as a remaining sibling after b.txt returned SkipDir")
	}
}

func TestRepositoryMatcher_Walk_SkipAll(t *testing.T) {
	structure := map[string]string{
		"a.txt":     "a\n",
		"sub/b.txt": "b\n",
		"c.txt":     "c\n",
	}
	tmpDir := createTestRepo(t, structure)
	defer os.RemoveAll(tmpDir)

	matcher, err := NewRepositoryMatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewRepositoryMatcher() failed: %v", err)
	}

	aPath := filepath.Join(tmpDir, "a.txt")
	var visits []string
	err = matcher.Walk(func(path string, d fs.DirEntry, ignored bool, walkErr error) error {
		visits = append(visits, path)
		if path == aPath {
			return fs.SkipAll
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Walk() error: %v", err)
	}

	for _, path := range visits {
		if path == filepath.Join(tmpDir, "sub") || path == filepath.Join(tmpDir, "c.txt") {
			t.Errorf("path %q visited after SkipAll should not have been", path)
		}
	}
}

func TestRepositoryMatcher_Walk_FilesystemError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses permission checks")
	}

	structure := map[string]string{
		"locked/secret.txt": "secret\n",
		"other.txt":         "o\n",
	}
	tmpDir := createTestRepo(t, structure)
	defer func() {
		os.Chmod(filepath.Join(tmpDir, "locked"), 0755)
		os.RemoveAll(tmpDir)
	}()

	lockedPath := filepath.Join(tmpDir, "locked")
	if err := os.Chmod(lockedPath, 0000); err != nil {
		t.Fatalf("failed to chmod directory: %v", err)
	}

	matcher, err := NewRepositoryMatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewRepositoryMatcher() failed: %v", err)
	}

	var sawErrForLocked bool
	var visitedSecretInsideLocked bool
	walkErr := matcher.Walk(func(path string, d fs.DirEntry, ignored bool, walkErr error) error {
		if path == lockedPath {
			if walkErr == nil {
				t.Error("expected a non-nil err for the unreadable directory")
			}
			sawErrForLocked = true
			return nil
		}
		if path == filepath.Join(lockedPath, "secret.txt") {
			visitedSecretInsideLocked = true
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("Walk() returned error though callback returned nil: %v", walkErr)
	}
	if !sawErrForLocked {
		t.Error("locked directory was never visited with an error")
	}
	if visitedSecretInsideLocked {
		t.Error("contents of unreadable directory should not have been visited")
	}
}

func TestRepositoryMatcher_Walk_FilesystemErrorAborts(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses permission checks")
	}

	structure := map[string]string{
		"locked/secret.txt": "secret\n",
	}
	tmpDir := createTestRepo(t, structure)
	defer func() {
		os.Chmod(filepath.Join(tmpDir, "locked"), 0755)
		os.RemoveAll(tmpDir)
	}()

	lockedPath := filepath.Join(tmpDir, "locked")
	if err := os.Chmod(lockedPath, 0000); err != nil {
		t.Fatalf("failed to chmod directory: %v", err)
	}

	matcher, err := NewRepositoryMatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewRepositoryMatcher() failed: %v", err)
	}

	sentinel := errors.New("abort")
	walkErr := matcher.Walk(func(path string, d fs.DirEntry, ignored bool, walkErr error) error {
		if path == lockedPath {
			return sentinel
		}
		return nil
	})
	if !errors.Is(walkErr, sentinel) {
		t.Errorf("Walk() error = %v, want it to wrap %v", walkErr, sentinel)
	}
}

func BenchmarkRepositoryMatcher_Matches(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "dotignore-bench-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	structure := map[string]string{
		".gitignore":          "*.log\n.DS_Store\n.env\n",
		"frontend/.gitignore": "node_modules/\ndist/\n.cache/\n*.local.js\n",
		"backend/.gitignore":  "target/\n*.class\nlogs/\n",
		"docs/.gitignore":     "_build/\n*.pyc\n",
	}
	for path, content := range structure {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			b.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			b.Fatalf("failed to write file: %v", err)
		}
	}

	matcher, err := NewRepositoryMatcher(tmpDir)
	if err != nil {
		b.Fatalf("NewRepositoryMatcher() failed: %v", err)
	}

	testPaths := []string{
		"app.log",
		"frontend/node_modules/package.json",
		"frontend/src/App.js",
		"backend/target/classes/Main.class",
		"backend/src/main.go",
		"docs/_build/html/index.html",
		"docs/index.rst",
		"README.md",
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, path := range testPaths {
			_, _ = matcher.Matches(path)
		}
	}
}
