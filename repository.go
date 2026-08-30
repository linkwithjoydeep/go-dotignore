// Package dotignore provides gitignore-style pattern matching for file paths.
package dotignore

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/codeglyph/go-dotignore/v2/internal"
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
	config   *RepositoryConfig
}

// RepositoryConfig configures the behavior of RepositoryMatcher.
type RepositoryConfig struct {
	// IgnoreFileName is the name of ignore files to process (default: ".gitignore")
	IgnoreFileName string

	// MaxDepth limits how deep to search for ignore files (0 = unlimited)
	MaxDepth int

	// FollowSymlinks determines whether to follow symbolic links when discovering ignore files
	FollowSymlinks bool

	// SkipFolders list of foldernames to not search for IgnoreFileName-files.
	SkipFolders []string
}

// DefaultRepositoryConfig returns a RepositoryConfig with sensible defaults.
func DefaultRepositoryConfig() *RepositoryConfig {
	return &RepositoryConfig{
		IgnoreFileName: ".gitignore",
		MaxDepth:       0, // unlimited
		FollowSymlinks: false,
		SkipFolders:    make([]string, 0),
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
		config:   config,
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

		if d.IsDir() && internal.Contains(config.SkipFolders, d.Name()) {
			return fs.SkipDir
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

	// Walk directories from root to leaf, applying matchers as we go.
	// currentDir tracks the directory path incrementally (no rebuilding from
	// scratch), and matchPath is sliced directly off relPath rather than
	// recomputed via filepath.Rel at each level.
	// Later matchers can override earlier ones via negation.
	matched := false
	currentDir := rm.rootDir
	matchPath := relPath

	if matcher, exists := rm.matchers[currentDir]; exists {
		isMatch, anyPatternMatched, err := matcher.MatchesWithTracking(matchPath)
		if err != nil {
			return false, fmt.Errorf("error matching against %s: %w", currentDir, err)
		}
		if anyPatternMatched {
			matched = isMatch
		}
	}

	for {
		idx := strings.IndexByte(matchPath, '/')
		if idx == -1 {
			break
		}
		currentDir = filepath.Join(currentDir, matchPath[:idx])
		matchPath = matchPath[idx+1:]

		matcher, exists := rm.matchers[currentDir]
		if !exists {
			continue
		}

		// Check if this matcher has a pattern that applies
		// Use MatchesWithTracking to know if any pattern actually matched
		isMatch, anyPatternMatched, err := matcher.MatchesWithTracking(matchPath)
		if err != nil {
			return false, fmt.Errorf("error matching against %s: %w", currentDir, err)
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

// WalkFunc is the type of the function called by Walk for every entry visited.
// It mirrors fs.WalkDirFunc, with an additional ignored parameter reporting
// whether the entry is excluded by the repository's .gitignore rules or
// RepositoryConfig.SkipFolders.
//
// Control flow matches filepath.WalkDir: returning fs.SkipDir for a directory
// skips its contents, for a file skips the rest of its siblings; returning
// fs.SkipAll stops the walk entirely; any other non-nil error aborts the walk
// and is returned by Walk.
type WalkFunc func(path string, d fs.DirEntry, ignored bool, err error) error

// repoWalkFrame is one entry in the incremental stack of applicable
// PatternMatchers maintained while walking. offset is the byte offset into a
// descendant's root-relative path at which the path relative to this frame's
// directory begins.
type repoWalkFrame struct {
	matcher *PatternMatcher
	offset  int
}

// Walk traverses the repository tree rooted at rm.RootDir(), calling fn for
// the root first (with ignored=false), then for every other entry exactly
// once, in lexical order, with absolute paths.
//
// A directory Walk will not descend into - because it matched a .gitignore
// rule or because its name is in RepositoryConfig.SkipFolders - is still
// reported once with ignored=true before Walk moves past it without visiting
// its contents. Walk honors the same MaxDepth and FollowSymlinks settings the
// RepositoryMatcher was constructed with.
func (rm *RepositoryMatcher) Walk(fn WalkFunc) error {
	rootInfo, statErr := os.Lstat(rm.rootDir)

	var rootEntry fs.DirEntry
	var rootEntries []fs.DirEntry
	readErr := statErr
	if statErr == nil {
		rootEntry = fs.FileInfoToDirEntry(rootInfo)
		rootEntries, readErr = os.ReadDir(rm.rootDir)
	}

	err := fn(rm.rootDir, rootEntry, false, readErr)
	if err == fs.SkipDir || err == fs.SkipAll {
		return nil
	}
	if err != nil {
		return err
	}
	if readErr != nil {
		return nil
	}

	var stack []repoWalkFrame
	if m, ok := rm.matchers[rm.rootDir]; ok {
		stack = append(stack, repoWalkFrame{matcher: m, offset: 0})
	}

	_, err = rm.walkEntries(rootEntries, rm.rootDir, "", 0, stack, fn)
	return err
}

// walkEntries visits the already-listed entries of a directory, recursing
// into subdirectories as needed. It returns (stopAll, err): stopAll is true
// once fn has returned fs.SkipAll, at which point every caller up the
// recursion unwinds without further callback invocations.
func (rm *RepositoryMatcher) walkEntries(entries []fs.DirEntry, parentAbs, parentRel string, depth int, stack []repoWalkFrame, fn WalkFunc) (bool, error) {
	for _, entry := range entries {
		name := entry.Name()
		childRel := name
		if parentRel != "" {
			childRel = parentRel + "/" + name
		}
		childAbs := filepath.Join(parentAbs, name)
		isDir := entry.IsDir()

		ignored, matchErr := rm.matchedAgainstStack(childRel, stack)

		skipFolder := isDir && internal.Contains(rm.config.SkipFolders, name)
		if skipFolder {
			ignored = true
		}

		exceedsDepth := isDir && rm.config.MaxDepth > 0 && strings.Count(childRel, "/") > rm.config.MaxDepth
		skipSymlink := isDir && entry.Type()&fs.ModeSymlink != 0 && !rm.config.FollowSymlinks

		wantDescend := isDir && !skipFolder && !ignored && matchErr == nil && !exceedsDepth && !skipSymlink

		var childEntries []fs.DirEntry
		var readErr error
		if wantDescend {
			childEntries, readErr = os.ReadDir(childAbs)
		}

		cbErr := matchErr
		if cbErr == nil {
			cbErr = readErr
		}
		err := fn(childAbs, entry, ignored, cbErr)
		if err == fs.SkipAll {
			return true, nil
		}
		if err == fs.SkipDir {
			if isDir {
				continue
			}
			return false, nil
		}
		if err != nil {
			return false, err
		}

		if !wantDescend || readErr != nil {
			continue
		}

		childStack := stack
		if m, ok := rm.matchers[childAbs]; ok {
			childStack = append(append([]repoWalkFrame(nil), stack...), repoWalkFrame{matcher: m, offset: len(childRel) + 1})
		}

		stopAll, walkErr := rm.walkEntries(childEntries, childAbs, childRel, depth+1, childStack, fn)
		if stopAll {
			return true, nil
		}
		if walkErr != nil {
			return false, walkErr
		}
	}
	return false, nil
}

// matchedAgainstStack applies the incremental matcher stack (root to leaf) to
// relPath, following the same root-to-leaf, negation-overrides-parent
// semantics as Matches.
func (rm *RepositoryMatcher) matchedAgainstStack(relPath string, stack []repoWalkFrame) (bool, error) {
	matched := false
	for _, frame := range stack {
		matchPath := relPath[frame.offset:]
		isMatch, anyPatternMatched, err := frame.matcher.MatchesWithTracking(matchPath)
		if err != nil {
			return false, fmt.Errorf("error matching against pattern matcher: %w", err)
		}
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
