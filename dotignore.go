// Package dotignore provides gitignore-style pattern matching for file paths.
//
// This package implements the gitignore specification including:
//   - Root-relative patterns with leading / (e.g., /build/ matches only at root)
//   - Wildcard patterns with *, ?, and ** (e.g., *.txt, **/test/*, a?b)
//   - Directory patterns with trailing / (e.g., logs/ matches directories)
//   - Negation patterns with ! (e.g., !important.txt)
//   - Escaped negation with \! (e.g., \!literal matches files starting with !)
//   - Character classes with [] (e.g., [a-z], [0-9])
//   - Pattern anchoring and path boundary matching
//
// IMPORTANT: Versions v1.0.0-v1.1.1 contain critical bugs and are retracted.
// Please upgrade to v2.0.0 or later for full gitignore specification compliance.
//
// Example usage:
//
//	patterns := []string{
//	    "/build/",     // Ignore build directory at root only
//	    "*.log",       // Ignore all .log files
//	    "!debug.log",  // But don't ignore debug.log
//	}
//	matcher, err := dotignore.NewPatternMatcher(patterns)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	shouldIgnore, err := matcher.Matches("build/output.txt")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if shouldIgnore {
//	    // Skip this file
//	}
package dotignore

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/codeglyph/go-dotignore/v2/internal"
)

type ignorePattern struct {
	pattern        string
	regexPattern   *regexp.Regexp
	isDirectory    bool // true if pattern ends with /
	negate         bool
	hasWildcard    bool // true if pattern contains wildcards
	isRootRelative bool // true if pattern starts with / (matches only at root level)
}

// PatternMatcher provides methods to parse, store, and evaluate ignore patterns against file paths.
type PatternMatcher struct {
	ignorePatterns []ignorePattern
}

// NewPatternMatcher initializes a new PatternMatcher instance from a list of string patterns.
func NewPatternMatcher(patterns []string) (*PatternMatcher, error) {
	ignorePatterns, err := buildIgnorePatterns(patterns)
	if err != nil {
		return nil, fmt.Errorf("failed to build ignore patterns: %w", err)
	}
	return &PatternMatcher{
		ignorePatterns: ignorePatterns,
	}, nil
}

// NewPatternMatcherFromReader initializes a new PatternMatcher instance from an io.Reader.
func NewPatternMatcherFromReader(reader io.Reader) (*PatternMatcher, error) {
	if reader == nil {
		return nil, errors.New("reader cannot be nil")
	}

	patterns, err := internal.ReadLines(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse patterns from reader: %w", err)
	}
	return NewPatternMatcher(patterns)
}

// NewPatternMatcherFromFile reads a file containing ignore patterns and returns a PatternMatcher instance.
func NewPatternMatcherFromFile(filePath string) (*PatternMatcher, error) {
	if filePath == "" {
		return nil, errors.New("file path cannot be empty")
	}

	fileReader, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q: %w", filePath, err)
	}
	defer fileReader.Close()

	patterns, err := internal.ReadLines(fileReader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse patterns from file %q: %w", filePath, err)
	}
	return NewPatternMatcher(patterns)
}

// Matches checks if the given file path matches any of the ignore patterns in the PatternMatcher.
// It returns true if the file should be ignored, false otherwise.
func (p *PatternMatcher) Matches(file string) (bool, error) {
	if file == "" {
		return false, nil
	}

	// Clean and normalize the path
	file = filepath.Clean(file)
	if file == "." || file == "./" {
		return false, nil
	}

	// Convert backslashes to forward slashes for consistent matching
	// Use explicit conversion to handle all cases
	file = strings.ReplaceAll(file, "\\", "/")

	return p.matchesInternal(file)
}

// MatchesWithTracking checks if the given file path matches any patterns and also
// returns whether any pattern (including negation patterns) matched at all.
// This is useful for hierarchical matching where we need to know if a .gitignore
// file had any applicable patterns.
//
// Returns: (shouldIgnore bool, anyPatternMatched bool, error)
func (p *PatternMatcher) MatchesWithTracking(file string) (bool, bool, error) {
	if file == "" {
		return false, false, nil
	}

	// Clean and normalize the path
	file = filepath.Clean(file)
	if file == "." || file == "./" {
		return false, false, nil
	}

	// Convert backslashes to forward slashes for consistent matching
	file = strings.ReplaceAll(file, "\\", "/")

	matched := false
	anyPatternMatched := false

	for _, pattern := range p.ignorePatterns {
		isMatch, err := p.matchPattern(file, pattern)
		if err != nil {
			return false, false, fmt.Errorf("error matching pattern %q against file %q: %w", pattern.pattern, file, err)
		}

		if isMatch {
			anyPatternMatched = true
			matched = !pattern.negate
		}
	}

	return matched, anyPatternMatched, nil
}

func buildIgnorePatterns(patterns []string) ([]ignorePattern, error) {
	var ignorePatterns []ignorePattern

	for i, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)

		// Skip empty lines and comments
		if pattern == "" || strings.HasPrefix(pattern, "#") {
			continue
		}

		// Handle escaped negation (\!) before checking for actual negation
		// In gitignore, \! at the start means "match files literally starting with !"
		isNegation := false
		if strings.HasPrefix(pattern, `\!`) {
			// Escaped negation - remove the backslash, keep the !
			pattern = pattern[1:] // Remove the backslash
			isNegation = false
		} else if strings.HasPrefix(pattern, "!") {
			// Actual negation pattern
			if len(pattern) == 1 {
				return nil, fmt.Errorf("invalid pattern at line %d: single '!' is not allowed", i+1)
			}
			pattern = pattern[1:]
			isNegation = true
		}

		// Convert backslashes to forward slashes for consistent handling
		// filepath.ToSlash might not handle all cases, so we'll be explicit
		pattern = strings.ReplaceAll(pattern, "\\", "/")

		// Check if pattern is root-relative (starts with /)
		// In gitignore, leading / means pattern is anchored to root
		isRootRelative := strings.HasPrefix(pattern, "/")
		if isRootRelative {
			pattern = strings.TrimPrefix(pattern, "/")
		}

		// Check if pattern is for directories only (after normalization)
		isDirectory := strings.HasSuffix(pattern, "/")
		if isDirectory {
			pattern = strings.TrimSuffix(pattern, "/")
		}

		// Validate pattern is not empty after processing
		if pattern == "" {
			return nil, fmt.Errorf("invalid pattern at line %d: pattern cannot be empty", i+1)
		}

		// Check if pattern contains wildcards
		hasWildcard := strings.ContainsAny(pattern, "*?")

		// Build regex pattern
		regexPattern, err := internal.BuildRegex(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to build regex for pattern %q at line %d: %w", pattern, i+1, err)
		}

		ignorePatterns = append(ignorePatterns, ignorePattern{
			pattern:        pattern,
			regexPattern:   regexPattern,
			isDirectory:    isDirectory,
			negate:         isNegation,
			hasWildcard:    hasWildcard,
			isRootRelative: isRootRelative,
		})
	}

	return ignorePatterns, nil
}

// matchesInternal performs the actual pattern matching logic
func (p *PatternMatcher) matchesInternal(file string) (bool, error) {
	matched := false

	for _, pattern := range p.ignorePatterns {
		isMatch, err := p.matchPattern(file, pattern)
		if err != nil {
			return false, fmt.Errorf("error matching pattern %q against file %q: %w", pattern.pattern, file, err)
		}

		if isMatch {
			matched = !pattern.negate
		}
	}

	return matched, nil
}

// matchPattern checks if a file matches a specific pattern
func (p *PatternMatcher) matchPattern(file string, pattern ignorePattern) (bool, error) {
	if pattern.isRootRelative {
		return matchRootRelativePattern(file, pattern), nil
	}
	if pattern.regexPattern.MatchString(file) {
		return true, nil
	}
	if pattern.isDirectory && matchDirectoryPattern(file, pattern) {
		return true, nil
	}
	if pattern.hasWildcard && matchWildcardSubpaths(file, pattern) {
		return true, nil
	}
	if strings.Contains(pattern.pattern, "/") {
		return matchPathSeparatorPattern(file, pattern), nil
	}
	return matchSimplePattern(file, pattern), nil
}

// matchRootRelativePattern handles patterns anchored to the root (starting with /).
func matchRootRelativePattern(file string, pattern ignorePattern) bool {
	if pattern.regexPattern.MatchString(file) {
		return true
	}
	if pattern.isDirectory {
		dirName := pattern.pattern
		return file == dirName || file == dirName+"/" || strings.HasPrefix(file, dirName+"/")
	}
	return file == pattern.pattern || strings.HasPrefix(file, pattern.pattern+"/")
}

// matchDirectoryPattern handles directory-only patterns (trailing /).
func matchDirectoryPattern(file string, pattern ignorePattern) bool {
	dirName := pattern.pattern
	if file == dirName {
		return true
	}
	if len(file) > len(dirName) && file[len(dirName)] == '/' && file[:len(dirName)] == dirName {
		return true
	}
	return len(file) == len(dirName)+1 && file[len(file)-1] == '/' && file[:len(dirName)] == dirName
}

// matchWildcardSubpaths tries the pattern against all sub-paths of file.
func matchWildcardSubpaths(file string, pattern ignorePattern) bool {
	parts := strings.Split(file, "/")
	for i := 0; i < len(parts); i++ {
		if pattern.regexPattern.MatchString(strings.Join(parts[i:], "/")) {
			return true
		}
	}
	for i := 1; i < len(parts); i++ {
		combined := strings.Join(parts[:i], "/") + "/" + strings.Join(parts[i:], "/")
		if pattern.regexPattern.MatchString(combined) {
			return true
		}
	}
	return false
}

// matchPathSeparatorPattern handles patterns that contain a path separator.
func matchPathSeparatorPattern(file string, pattern ignorePattern) bool {
	if file == pattern.pattern {
		return true
	}
	patternLen := len(pattern.pattern)
	if len(file) > patternLen && file[patternLen] == '/' && file[:patternLen] == pattern.pattern {
		return true
	}
	if len(file) > patternLen && file[len(file)-patternLen:] == pattern.pattern && file[len(file)-patternLen-1] == '/' {
		return true
	}
	return strings.Contains(file, "/"+pattern.pattern+"/")
}

// matchSimplePattern handles patterns without path separators by checking each path component.
func matchSimplePattern(file string, pattern ignorePattern) bool {
	for _, part := range strings.Split(file, "/") {
		if pattern.regexPattern.MatchString(part) {
			return true
		}
	}
	return false
}
