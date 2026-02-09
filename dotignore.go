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
	pattern       string
	regexPattern  *regexp.Regexp
	isDirectory   bool // true if pattern ends with /
	negate        bool
	hasWildcard   bool // true if pattern contains wildcards
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
	// Handle root-relative patterns (patterns starting with /)
	// These should ONLY match at the root level, not in subdirectories
	if pattern.isRootRelative {
		// For root-relative patterns, only match if:
		// 1. File exactly matches the pattern
		// 2. File is inside the pattern directory (for directory patterns)
		// 3. Pattern matches from the start (no parent directories before it)

		// Direct regex match (already anchored to start with ^)
		if pattern.regexPattern.MatchString(file) {
			return true, nil
		}

		// For directory patterns like /build/, match build/ and build/anything
		if pattern.isDirectory {
			dirName := pattern.pattern
			if file == dirName || file == dirName+"/" {
				return true, nil
			}
			// Check if file is inside the directory
			if strings.HasPrefix(file, dirName+"/") {
				return true, nil
			}
		} else {
			// For file patterns like /test.txt, check exact match or with extension
			if file == pattern.pattern {
				return true, nil
			}
			// Also check if pattern is a prefix (for paths like /src matching /src/file.txt)
			if strings.HasPrefix(file, pattern.pattern+"/") {
				return true, nil
			}
		}

		// Root-relative patterns don't do subpath matching
		return false, nil
	}

	// Non-root-relative patterns: try the regex pattern first
	if pattern.regexPattern.MatchString(file) {
		return true, nil
	}

	// Special handling for directory patterns
	if pattern.isDirectory {
		// Pattern like "build/" should match "build/" and anything inside "build/"
		dirName := pattern.pattern
		if file == dirName {
			return true, nil
		}
		// Check if it ends with "/" first before allocating
		if len(file) > len(dirName) && file[len(dirName)] == '/' && file[:len(dirName)] == dirName {
			return true, nil
		}
		// Check if file ends with just "/"
		if len(file) == len(dirName)+1 && file[len(file)-1] == '/' && file[:len(dirName)] == dirName {
			return true, nil
		}
	}

	// For patterns with wildcards, try matching parts of the path
	if pattern.hasWildcard {
		parts := strings.Split(file, "/")

		// For patterns like "src/*.txt", try matching against subpaths
		for i := 0; i < len(parts); i++ {
			subPath := strings.Join(parts[i:], "/")
			if pattern.regexPattern.MatchString(subPath) {
				return true, nil
			}
		}

		// Also try matching the full path from different starting points
		// Skip first iteration since we already tried the full path above
		for i := 1; i < len(parts); i++ {
			prefixPath := strings.Join(parts[:i], "/")
			remainingPath := strings.Join(parts[i:], "/")

			// Check if pattern could match from this point
			if pattern.regexPattern.MatchString(prefixPath + "/" + remainingPath) {
				return true, nil
			}
		}
	}

	// For patterns with path separators, check for matches at proper path boundaries
	if strings.Contains(pattern.pattern, "/") {
		// Exact match (no allocation)
		if file == pattern.pattern {
			return true, nil
		}

		patternLen := len(pattern.pattern)

		// Pattern at the beginning - check boundary without allocation
		if len(file) > patternLen && file[patternLen] == '/' && file[:patternLen] == pattern.pattern {
			return true, nil
		}

		// Pattern at the end with boundary - check without allocation
		if len(file) > patternLen && file[len(file)-patternLen:] == pattern.pattern {
			if file[len(file)-patternLen-1] == '/' {
				return true, nil
			}
		}

		// Pattern in the middle with boundaries
		// This does allocate but only once per call
		if strings.Contains(file, "/"+pattern.pattern+"/") {
			return true, nil
		}
	}

	// For simple patterns (no path separators), check filename components
	if !strings.Contains(pattern.pattern, "/") {
		parts := strings.Split(file, "/")
		for _, part := range parts {
			if pattern.regexPattern.MatchString(part) {
				return true, nil
			}
		}
	}

	return false, nil
}
