// Package dotignore provides gitignore-style pattern matching for file paths.
//
// # Overview
//
// This package implements the full gitignore specification for pattern matching,
// allowing you to determine whether files should be ignored based on gitignore-style
// patterns. It's useful for building tools that need to respect .gitignore files,
// or for implementing custom file filtering logic.
//
// # Features
//
//   - Root-relative patterns with leading / (e.g., /build/ matches only at root)
//   - Wildcard patterns with *, ?, and ** (e.g., *.txt, **/test/*, a?b)
//   - Directory patterns with trailing / (e.g., logs/ matches directories)
//   - Negation patterns with ! (e.g., !important.txt)
//   - Escaped negation with \! (e.g., \!literal matches files starting with !)
//   - Character classes with [] (e.g., [a-z], [0-9])
//   - Cross-platform path handling (Windows and Unix)
//   - Thread-safe pattern matching
//
// # Version Notice
//
// ⚠️ IMPORTANT: Versions v1.0.0-v1.1.1 contain critical bugs and are retracted.
// Always use v2.0.0 or later for production code.
//
// Critical bugs in v1.x:
//   - Root-relative patterns (/pattern) don't work at all
//   - Substring matching causes false positive matches
//   - No support for escaped negation (\!)
//
// # Quick Start
//
// Create a pattern matcher from a list of patterns:
//
//	patterns := []string{
//	    "/build/",      // Root-level build directory only
//	    "*.log",        // All .log files
//	    "!debug.log",   // Except debug.log
//	    "**/temp/**",   // temp directories at any level
//	}
//
//	matcher, err := dotignore.NewPatternMatcher(patterns)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Check if a file should be ignored
//	ignore, err := matcher.Matches("build/output.txt")
//	if ignore {
//	    // Skip this file
//	}
//
// # Pattern Syntax
//
// Root-Relative Patterns:
//
//	/build/         Matches "build/" at root only, not "src/build/"
//	/*.txt          Matches .txt files at root only
//	/src/*.go       Matches .go files in root-level src/ only
//
// Wildcards:
//
//	*.txt           Matches .txt files at any level
//	temp/*          Matches files directly in any temp/ directory
//	**/logs/**      Matches anything in logs/ directories at any level
//	file?.txt       Matches file1.txt, fileA.txt, etc.
//
// Directory Patterns:
//
//	logs/           Matches logs/ directory and all its contents
//	/cache/         Matches root-level cache/ and its contents
//
// Negation:
//
//	!important.log  Don't ignore important.log (overrides previous patterns)
//	\!literal.txt   Match files literally named "!literal.txt"
//
// Character Classes:
//
//	[a-z]*.txt      Matches a.txt, b.txt, ... z.txt
//	test[0-9].log   Matches test0.log, test1.log, ... test9.log
//
// # Reading from .gitignore Files
//
// Read patterns from a .gitignore file:
//
//	matcher, err := dotignore.NewPatternMatcherFromFile(".gitignore")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Or from any io.Reader
//	file, _ := os.Open(".gitignore")
//	matcher, err := dotignore.NewPatternMatcherFromReader(file)
//
// # Performance
//
// The package is optimized for performance:
//   - Regex compilation happens once during initialization
//   - Pattern matching is ~34µs per operation
//   - Thread-safe for concurrent use
//   - No allocations during regex matching
//
// # Compatibility
//
// Fully compatible with Git's gitignore specification as documented at:
// https://git-scm.com/docs/gitignore
//
// Tested on:
//   - Linux
//   - macOS
//   - Windows
//
// Requires Go 1.20 or later.
//
// # Examples
//
// See the package examples for common use cases:
//   - ExampleNewPatternMatcher - Basic usage
//   - ExamplePatternMatcher_Matches - File matching
//   - ExampleNewPatternMatcherFromFile - Reading from files
//   - ExamplePatternMatcher_Matches_directories - Directory patterns
//   - ExamplePatternMatcher_Matches_wildcards - Wildcard patterns
//   - ExamplePatternMatcher_Matches_negation - Negation patterns
package dotignore
