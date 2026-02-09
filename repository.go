// Package dotignore provides gitignore-style pattern matching for file paths.
package dotignore

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// RepositoryMatcher provides hierarchical .gitignore pattern matching that mirrors
// Git's native behavior with nested .gitignore files in subdirectories.
//
// Unlike PatternMatcher which processes patterns from a single source, RepositoryMatcher
// automatically discovers and processes .gitignore files throughout a directory tree,
// applying Git's precedence rules where patterns in deeper directories override those
// in parent directories.
//
// Example usage:
//
//	matcher, err := dotignore.NewRepositoryMatcher("/path/to/repo")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Check if a file should be ignored
//	shouldIgnore, err := matcher.Matches("frontend/node_modules/package.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
type RepositoryMatcher struct {
	rootDir  string
	matchers map[string]*PatternMatcher // Map of directory path -> matcher
}

// RepositoryConfig configures the behavior of RepositoryMatcher.
type RepositoryConfig struct {
	// IgnoreFileName is the name of ignore files to process (default: ".gitignore")
	IgnoreFileName string

	// MaxDepth limits how deep to search for ignore files (0 = unlimited)
	MaxDepth int

	// FollowSymlinks determines whether to follow symbolic links when discovering ignore files
	FollowSymlinks bool
}

// DefaultRepositoryConfig returns a RepositoryConfig with sensible defaults.
func DefaultRepositoryConfig() *RepositoryConfig {
	return &RepositoryConfig{
		IgnoreFileName: ".gitignore",
		MaxDepth:       0, // unlimited
		FollowSymlinks: false,
	}
}

// NewRepositoryMatcher creates a new RepositoryMatcher for the given root directory.
// It automatically discovers and loads all .gitignore files in the directory tree.
//
// The root directory should be an absolute path. Relative paths will be converted
// to absolute paths relative to the current working directory.
func NewRepositoryMatcher(rootDir string) (*RepositoryMatcher, error) {
	return NewRepositoryMatcherWithConfig(rootDir, DefaultRepositoryConfig())
}

// NewRepositoryMatcherWithConfig creates a new RepositoryMatcher with custom configuration.
func NewRepositoryMatcherWithConfig(rootDir string, config *RepositoryConfig) (*RepositoryMatcher, error) {
	if rootDir == "" {
		return nil, errors.New("root directory cannot be empty")
	}

	if config == nil {
		config = DefaultRepositoryConfig()
	}

	if config.IgnoreFileName == "" {
		config.IgnoreFileName = ".gitignore"
	}

	// Convert to absolute path
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path for %q: %w", rootDir, err)
	}

	// Verify directory exists
	info, err := os.Stat(absRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to access directory %q: %w", absRoot, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%q is not a directory", absRoot)
	}

	rm := &RepositoryMatcher{
		rootDir:  absRoot,
		matchers: make(map[string]*PatternMatcher),
	}

	// Discover and load all .gitignore files
	if err := rm.discoverIgnoreFiles(config); err != nil {
		return nil, fmt.Errorf("failed to discover ignore files: %w", err)
	}

	return rm, nil
}

// discoverIgnoreFiles walks the directory tree and loads all .gitignore files.
func (rm *RepositoryMatcher) discoverIgnoreFiles(config *RepositoryConfig) error {
	return filepath.WalkDir(rm.rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// If we can't read a directory, skip it but don't fail
			if os.IsPermission(err) || os.IsNotExist(err) {
				return fs.SkipDir
			}
			return err
		}

		// Check depth limit
		if config.MaxDepth > 0 {
			relPath, err := filepath.Rel(rm.rootDir, path)
			if err != nil {
				return err
			}
			depth := strings.Count(relPath, string(filepath.Separator))
			if depth > config.MaxDepth {
				return fs.SkipDir
			}
		}

		// Handle symlinks
		if d.Type()&fs.ModeSymlink != 0 && !config.FollowSymlinks {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		// Check if this is an ignore file
		if !d.IsDir() && d.Name() == config.IgnoreFileName {
			dir := filepath.Dir(path)

			// Load the .gitignore file
			matcher, err := NewPatternMatcherFromFile(path)
			if err != nil {
				// If we can't parse the file, skip it but log the error
				// Don't fail the entire operation
				return nil
			}

			rm.matchers[dir] = matcher
		}

		return nil
	})
}

// Matches checks if the given file path should be ignored according to the
// hierarchical .gitignore rules. The path should be relative to the repository root
// or an absolute path within the repository.
//
// The matching follows Git's behavior:
//  1. Patterns from .gitignore files in parent directories apply to subdirectories
//  2. Patterns in deeper .gitignore files can override parent patterns using negation
//  3. Patterns are evaluated from root to the file's directory, with later patterns
//     taking precedence
func (rm *RepositoryMatcher) Matches(path string) (bool, error) {
	if path == "" {
		return false, nil
	}

	// Convert to absolute path if needed
	var absPath string
	if filepath.IsAbs(path) {
		absPath = filepath.Clean(path)
	} else {
		absPath = filepath.Clean(filepath.Join(rm.rootDir, path))
	}

	// Ensure the path is within the repository
	if !strings.HasPrefix(absPath, rm.rootDir) {
		return false, fmt.Errorf("path %q is outside repository root %q", path, rm.rootDir)
	}

	// Get relative path from root
	relPath, err := filepath.Rel(rm.rootDir, absPath)
	if err != nil {
		return false, fmt.Errorf("failed to compute relative path: %w", err)
	}

	// Normalize to forward slashes for consistent matching
	relPath = filepath.ToSlash(relPath)

	// Build list of directories from root to the file's directory
	// We need to check .gitignore files in order from root to leaf
	var dirsToCheck []string
	currentDir := rm.rootDir
	dirsToCheck = append(dirsToCheck, currentDir)

	// Split the relative path and build up directory path
	parts := strings.Split(relPath, "/")
	for i := 0; i < len(parts)-1; i++ {
		currentDir = filepath.Join(currentDir, parts[i])
		dirsToCheck = append(dirsToCheck, currentDir)
	}

	// Apply matchers in order from root to leaf
	// Later matchers can override earlier ones via negation
	matched := false

	for _, dir := range dirsToCheck {
		matcher, exists := rm.matchers[dir]
		if !exists {
			continue
		}

		// Compute path relative to this matcher's directory
		var matchPath string
		if dir == rm.rootDir {
			matchPath = relPath
		} else {
			relToDir, err := filepath.Rel(dir, absPath)
			if err != nil {
				continue
			}
			matchPath = filepath.ToSlash(relToDir)
		}

		// Check if this matcher has a pattern that applies
		// Use MatchesWithTracking to know if any pattern actually matched
		isMatch, anyPatternMatched, err := matcher.MatchesWithTracking(matchPath)
		if err != nil {
			return false, fmt.Errorf("error matching against %s: %w", dir, err)
		}

		// Only update matched status if a pattern actually matched
		// This allows deeper .gitignore files to override parent patterns
		// through negation (e.g., parent has "*.log", child has "!debug.log")
		// but doesn't override if the child .gitignore has no applicable patterns
		if anyPatternMatched {
			matched = isMatch
		}
	}

	return matched, nil
}

// RootDir returns the absolute path to the repository root directory.
func (rm *RepositoryMatcher) RootDir() string {
	return rm.rootDir
}

// IgnoreFileCount returns the number of .gitignore files discovered and loaded.
func (rm *RepositoryMatcher) IgnoreFileCount() int {
	return len(rm.matchers)
}

// IgnoreFilePaths returns a list of all .gitignore file paths that were loaded,
// relative to the repository root.
func (rm *RepositoryMatcher) IgnoreFilePaths() []string {
	var paths []string
	for dir := range rm.matchers {
		relDir, err := filepath.Rel(rm.rootDir, dir)
		if err != nil {
			continue
		}
		if relDir == "." {
			paths = append(paths, ".gitignore")
		} else {
			paths = append(paths, filepath.Join(relDir, ".gitignore"))
		}
	}
	return paths
}
