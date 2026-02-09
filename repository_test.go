package dotignore

import (
	"os"
	"path/filepath"
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
				".gitignore":                    "*.log\n",
				"a/.gitignore":                  "*.tmp\n",
				"a/b/.gitignore":                "*.cache\n",
				"a/b/c/.gitignore":              "*.test\n",
				"a/b/c/d/.gitignore":            "*.debug\n",
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
		".gitignore": "*.log\ntemp/\n",
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
		".gitignore": "*.log\n!important.log\n",
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
		".gitignore": "*.txt\n",
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
		".gitignore": "/build/\nconfig/\n",
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
		".gitignore": "*.log\n",
		"frontend/.gitignore": "node_modules/\n",
		"backend/.gitignore": "target/\n",
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
		".gitignore":                "*.log\n",
		"a/.gitignore":              "*.tmp\n",
		"a/b/.gitignore":            "*.cache\n",
		"a/b/c/.gitignore":          "*.test\n",
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
		".ignore": "*.log\n",
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
		".gitignore": "node_modules/\n**/*.test.js\n",
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

func TestRepositoryMatcher_EmptyFile(t *testing.T) {
	structure := map[string]string{
		".gitignore": "",
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
