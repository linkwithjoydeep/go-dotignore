package dotignore

import (
	"os"
	"strings"
	"testing"
)

func TestNewPatternMatcherFromFile(t *testing.T) {
	// Create a temporary .ignore file
	ignoreContent := `
# Comments should be ignored
ignore.exe
6.out
ignore.so
!ok.go
!ok.py
!ok.exe
!ok.a
# Another comment
`
	tempFile, err := os.CreateTemp("", "test.ignore")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.WriteString(ignoreContent); err != nil {
		t.Fatalf("Failed to write to temporary file: %v", err)
	}
	tempFile.Close()

	matcher, err := NewPatternMatcherFromFile(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to parse ignore file: %v", err)
	}

	// Test files that should be ignored
	ignoredFiles := []string{"ignore.exe", "6.out", "ignore.so"}
	for _, filename := range ignoredFiles {
		isIgnored, err := matcher.Matches(filename)
		if err != nil {
			t.Errorf("Error matching file %s: %v", filename, err)
			continue
		}
		if !isIgnored {
			t.Errorf("Expected file %s to be ignored, but it wasn't", filename)
		}
	}

	// Test files that should be included (negated patterns)
	includedFiles := []string{"ok.go", "ok.py", "ok.exe", "ok.a"}
	for _, filename := range includedFiles {
		isIgnored, err := matcher.Matches(filename)
		if err != nil {
			t.Errorf("Error matching file %s: %v", filename, err)
			continue
		}
		if isIgnored {
			t.Errorf("Expected file %s to be included, but it was ignored", filename)
		}
	}
}

func TestMatches(t *testing.T) {
	patterns := []string{
		"**",          // Match everything
		"**/",         // Match all directories
		"dir/**",      // Match everything in dir
		"**/dir2/*",   // Match files directly in any dir2
		"**/dir2/**",  // Match everything in any dir2
		"**/file",     // Match any file named "file"
		"**/**/*.txt", // Match .txt files anywhere (redundant but valid)
		"a**/*.txt",   // Match .txt files in paths starting with 'a'
		"a/*.txt",     // Match .txt files directly in 'a' directory
		"a[b-d]e",     // Character class: abe, ace, ade
		"abc.def",     // Exact match
		"abc?def",     // Single character wildcard
		"a\\*b",       // Escaped asterisk (literal *)
		"**/foo/bar",  // Match foo/bar anywhere
		"abc/**",      // Match everything in abc directory
	}

	matcher, err := NewPatternMatcher(patterns)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	tests := []struct {
		file     string
		expected bool
		reason   string
	}{
		{"file", true, "matches **/file pattern"},
		{"file/", true, "matches ** pattern"},
		{"dir/file", true, "matches dir/** pattern"},
		{"dir/file/", true, "matches dir/** pattern"},
		{"something/dir2/file", true, "matches **/dir2/* pattern"},
		{"something/dir2/sub/file", true, "matches **/dir2/** pattern"},
		{"a/file.txt", true, "matches a/*.txt pattern"},
		{"atest/file.txt", true, "matches a**/*.txt pattern"},
		{"abc.def", true, "exact match"},
		{"abcXdef", true, "matches abc?def pattern"},
		{"a*b", true, "matches a\\*b pattern (literal asterisk)"},
		{"deep/foo/bar", true, "matches **/foo/bar pattern"},
		{"abc/anything", true, "matches abc/** pattern"},
		{"abe", true, "matches a[b-d]e character class"},
		{"ace", true, "matches a[b-d]e character class"},
		{"ade", true, "matches a[b-d]e character class"},

		// These should not match specific patterns but might match **
		{"aae", true, "doesn't match a[b-d]e but matches **"},
		{"afe", true, "doesn't match a[b-d]e but matches **"},
	}

	for _, test := range tests {
		t.Run(test.file, func(t *testing.T) {
			result, err := matcher.Matches(test.file)
			if err != nil {
				t.Errorf("Error matching file %s: %v", test.file, err)
				return
			}
			if result != test.expected {
				t.Errorf("File %s: expected %v, got %v (%s)", test.file, test.expected, result, test.reason)
			}
		})
	}
}

func TestBuildIgnorePatterns(t *testing.T) {
	patterns := []string{"docs", "config", "", "# comment"}
	ignorePatterns, err := buildIgnorePatterns(patterns)
	if err != nil {
		t.Fatalf("buildIgnorePatterns failed: %v", err)
	}

	// Should filter out empty patterns and comments
	expectedCount := 2
	if len(ignorePatterns) != expectedCount {
		t.Errorf("Expected %d patterns, got %d", expectedCount, len(ignorePatterns))
	}

	// Check that we got the right patterns
	if len(ignorePatterns) >= 1 && ignorePatterns[0].pattern != "docs" {
		t.Errorf("Expected first pattern to be 'docs', got '%s'", ignorePatterns[0].pattern)
	}
	if len(ignorePatterns) >= 2 && ignorePatterns[1].pattern != "config" {
		t.Errorf("Expected second pattern to be 'config', got '%s'", ignorePatterns[1].pattern)
	}
}

func TestBuildIgnorePatternsStripEmptyPatterns(t *testing.T) {
	patterns := []string{"docs", "config", "", "   ", "# comment"}
	ignorePatterns, err := buildIgnorePatterns(patterns)
	if err != nil {
		t.Fatalf("buildIgnorePatterns failed: %v", err)
	}

	expectedCount := 2
	if len(ignorePatterns) != expectedCount {
		t.Errorf("Expected %d patterns after filtering, got %d", expectedCount, len(ignorePatterns))
	}
}

func TestBuildIgnorePatternsExceptionFlag(t *testing.T) {
	patterns := []string{"docs", "!docs/README.md"}
	ignorePatterns, err := buildIgnorePatterns(patterns)
	if err != nil {
		t.Fatalf("buildIgnorePatterns failed: %v", err)
	}

	if len(ignorePatterns) < 2 {
		t.Fatalf("Expected at least 2 patterns, got %d", len(ignorePatterns))
	}

	if !ignorePatterns[1].negate {
		t.Errorf("Expected second pattern to have negate=true, got %v", ignorePatterns[1].negate)
	}

	if ignorePatterns[1].pattern != "docs/README.md" {
		t.Errorf("Expected pattern to be 'docs/README.md', got '%s'", ignorePatterns[1].pattern)
	}
}

func TestBuildIgnorePatternsLeadingSpaceTrimmed(t *testing.T) {
	patterns := []string{"docs", "  !docs/README.md"}
	ignorePatterns, err := buildIgnorePatterns(patterns)
	if err != nil {
		t.Fatalf("buildIgnorePatterns failed: %v", err)
	}

	if len(ignorePatterns) < 2 {
		t.Fatalf("Expected at least 2 patterns, got %d", len(ignorePatterns))
	}

	if !ignorePatterns[1].negate {
		t.Errorf("Expected negate to be true after trimming leading space, got %v", ignorePatterns[1].negate)
	}
}

func TestBuildIgnorePatternsTrailingSpaceTrimmed(t *testing.T) {
	patterns := []string{"docs", "!docs/README.md  "}
	ignorePatterns, err := buildIgnorePatterns(patterns)
	if err != nil {
		t.Fatalf("buildIgnorePatterns failed: %v", err)
	}

	if len(ignorePatterns) < 2 {
		t.Fatalf("Expected at least 2 patterns, got %d", len(ignorePatterns))
	}

	if !ignorePatterns[1].negate {
		t.Errorf("Expected negate to be true after trimming trailing space, got %v", ignorePatterns[1].negate)
	}
}

func TestBuildIgnorePatternsErrorSingleException(t *testing.T) {
	patterns := []string{"!"}
	_, err := buildIgnorePatterns(patterns)
	if err == nil {
		t.Error("Expected error for single exclamation point pattern")
	}

	expectedErr := "single '!' is not allowed"
	if err != nil && !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Expected error message to contain '%s', got: %v", expectedErr, err)
	}
}

func TestBuildIgnorePatternsFolderSplit(t *testing.T) {
	patterns := []string{"docs/config/CONFIG.md"}
	ignorePatterns, err := buildIgnorePatterns(patterns)
	if err != nil {
		t.Fatalf("buildIgnorePatterns failed: %v", err)
	}

	if len(ignorePatterns) == 0 {
		t.Fatal("Expected at least one pattern")
	}

	pattern := ignorePatterns[0]
	expectedPattern := "docs/config/CONFIG.md"
	if pattern.pattern != expectedPattern {
		t.Errorf("Expected pattern to be '%s', got '%s'", expectedPattern, pattern.pattern)
	}
}

func TestDirectoryPatterns(t *testing.T) {
	patterns := []string{
		"build/",    // Directory only
		"*.tmp",     // File pattern
		"src/test/", // Nested directory
	}

	matcher, err := NewPatternMatcher(patterns)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	tests := []struct {
		file     string
		expected bool
		reason   string
	}{
		{"build/", true, "build/ directory should match build/ pattern"},
		{"build/file.txt", true, "files in build directory should be matched"},
		{"file.tmp", true, "should match *.tmp pattern"},
		{"src/test/", true, "should match src/test/ directory pattern"},
		{"src/test/file.js", true, "files in src/test should be matched"},
		{"other/", false, "other/ directory should not match"},
		{"file.txt", false, "regular .txt files should not match"},
	}

	for _, test := range tests {
		t.Run(test.file, func(t *testing.T) {
			result, err := matcher.Matches(test.file)
			if err != nil {
				t.Errorf("Error matching file %s: %v", test.file, err)
				return
			}
			if result != test.expected {
				t.Errorf("File %s: expected %v, got %v (%s)", test.file, test.expected, result, test.reason)
			}
		})
	}
}

func TestNegationPatterns(t *testing.T) {
	patterns := []string{
		"*.log",          // Ignore all .log files
		"!important.log", // But keep important.log
		"temp/",          // Ignore temp directory
		"!temp/keep.txt", // But keep temp/keep.txt
	}

	matcher, err := NewPatternMatcher(patterns)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	tests := []struct {
		file     string
		expected bool
		reason   string
	}{
		{"app.log", true, "should match *.log pattern"},
		{"important.log", false, "should be negated by !important.log"},
		{"temp/cache.txt", true, "should match temp/ directory pattern"},
		{"temp/keep.txt", false, "should be negated by !temp/keep.txt"},
		{"other.txt", false, "should not match any pattern"},
	}

	for _, test := range tests {
		t.Run(test.file, func(t *testing.T) {
			result, err := matcher.Matches(test.file)
			if err != nil {
				t.Errorf("Error matching file %s: %v", test.file, err)
				return
			}
			if result != test.expected {
				t.Errorf("File %s: expected %v, got %v (%s)", test.file, test.expected, result, test.reason)
			}
		})
	}
}

func TestEmptyAndCommentPatterns(t *testing.T) {
	patterns := []string{
		"", // Empty line
		"# This is a comment",
		"*.txt", // Valid pattern
		"   ",   // Whitespace only
		"# Another comment",
		"!important.txt", // Valid negation
		"",               // Another empty line
	}

	matcher, err := NewPatternMatcher(patterns)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	// Should only have 2 active patterns: *.txt and !important.txt
	tests := []struct {
		file     string
		expected bool
	}{
		{"test.txt", true},       // Matches *.txt
		{"important.txt", false}, // Negated by !important.txt
		{"test.log", false},      // No matching pattern
	}

	for _, test := range tests {
		t.Run(test.file, func(t *testing.T) {
			result, err := matcher.Matches(test.file)
			if err != nil {
				t.Errorf("Error matching file %s: %v", test.file, err)
				return
			}
			if result != test.expected {
				t.Errorf("File %s: expected %v, got %v", test.file, test.expected, result)
			}
		})
	}
}

func TestNewPatternMatcherErrors(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		wantErr  bool
	}{
		{
			name:     "Single exclamation",
			patterns: []string{"!"},
			wantErr:  true,
		},
		{
			name:     "Empty pattern after negation",
			patterns: []string{"!", "valid.txt"},
			wantErr:  true,
		},
		{
			name:     "Valid patterns",
			patterns: []string{"*.txt", "!important.txt"},
			wantErr:  false,
		},
		{
			name:     "Empty slice",
			patterns: []string{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPatternMatcher(tt.patterns)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPatternMatcher() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewPatternMatcherFromReaderErrors(t *testing.T) {
	t.Run("Nil reader", func(t *testing.T) {
		_, err := NewPatternMatcherFromReader(nil)
		if err == nil {
			t.Error("Expected error for nil reader")
		}
	})
}

func TestNewPatternMatcherFromFileErrors(t *testing.T) {
	t.Run("Empty filepath", func(t *testing.T) {
		_, err := NewPatternMatcherFromFile("")
		if err == nil {
			t.Error("Expected error for empty filepath")
		}
	})

	t.Run("Non-existent file", func(t *testing.T) {
		_, err := NewPatternMatcherFromFile("non-existent-file.txt")
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
	})
}

func TestMatchesEdgeCases(t *testing.T) {
	patterns := []string{"*.txt", "!important.txt", "temp/"}
	matcher, err := NewPatternMatcher(patterns)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	tests := []struct {
		name     string
		file     string
		expected bool
	}{
		{
			name:     "Empty string",
			file:     "",
			expected: false,
		},
		{
			name:     "Current directory",
			file:     ".",
			expected: false,
		},
		{
			name:     "Current directory with slash",
			file:     "./",
			expected: false,
		},
		{
			name:     "Text file should be ignored",
			file:     "test.txt",
			expected: true,
		},
		{
			name:     "Important file should not be ignored",
			file:     "important.txt",
			expected: false,
		},
		{
			name:     "File in temp directory",
			file:     "temp/file.log",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := matcher.Matches(tt.file)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for file %q", tt.expected, result, tt.file)
			}
		})
	}
}

func TestSubstringMatchingBug(t *testing.T) {
	// Test for the substring matching bug where patterns with "/" incorrectly
	// match files that contain the pattern as a substring without proper path boundaries
	patterns := []string{
		"src/test",
	}

	matcher, err := NewPatternMatcher(patterns)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	tests := []struct {
		file     string
		expected bool
		reason   string
	}{
		{"src/test", true, "exact match should work"},
		{"src/test/file.js", true, "pattern should match directory and its contents"},
		{"foo/src/test", true, "pattern should match at path boundaries"},
		{"foo/src/test/bar.js", true, "pattern should match at path boundaries with nested files"},
		{"mysrc/test", false, "should NOT match - 'src' is not at path boundary"},
		{"src/test2", false, "should NOT match - 'test' is not at path boundary"},
		{"foo/mysrc/test", false, "should NOT match - 'src' is not at path boundary in subdirectory"},
	}

	for _, tt := range tests {
		result, err := matcher.Matches(tt.file)
		if err != nil {
			t.Errorf("Error matching file %s: %v", tt.file, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("File %q: expected %v, got %v - %s", tt.file, tt.expected, result, tt.reason)
		}
	}
}

func TestEscapedNegation(t *testing.T) {
	// Test escaped negation pattern \! which should match files starting with literal "!"
	// According to gitignore spec, \! at the start means "match literal !", not a negation
	patterns := []string{
		"*.log",           // Ignore all .log files
		"!important.log",  // Negate: don't ignore important.log
		`\!special.log`,   // Escaped: match files literally named "!special.log"
	}

	matcher, err := NewPatternMatcher(patterns)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	tests := []struct {
		file     string
		expected bool
		reason   string
	}{
		{"test.log", true, "*.log should match"},
		{"important.log", false, "!important.log negates the match"},
		{"!special.log", true, `\!special.log should match files literally named "!special.log"`},
		{"special.log", true, "*.log should match"},
		{"test.txt", false, "should not match - different extension"},
	}

	for _, tt := range tests {
		result, err := matcher.Matches(tt.file)
		if err != nil {
			t.Errorf("Error matching file %s: %v", tt.file, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("File %q: expected %v, got %v - %s", tt.file, tt.expected, result, tt.reason)
		}
	}
}

func TestEscapedNegationWithoutOtherPatterns(t *testing.T) {
	// Test escaped negation pattern in isolation
	patterns := []string{
		`\!important.txt`,
	}

	matcher, err := NewPatternMatcher(patterns)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	tests := []struct {
		file     string
		expected bool
		reason   string
	}{
		{"!important.txt", true, "should match file literally named '!important.txt'"},
		{"important.txt", false, "should not match - no leading '!'"},
		{"test.txt", false, "should not match - different file"},
	}

	for _, tt := range tests {
		result, err := matcher.Matches(tt.file)
		if err != nil {
			t.Errorf("Error matching file %s: %v", tt.file, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("File %q: expected %v, got %v - %s", tt.file, tt.expected, result, tt.reason)
		}
	}
}

func TestUnicodePatterns(t *testing.T) {
	// Test Unicode and non-ASCII patterns
	patterns := []string{
		"日本語.txt",       // Japanese
		"файл.log",        // Russian
		"🎉celebration.md", // Emoji
		"café/*.txt",      // Accented characters
	}

	matcher, err := NewPatternMatcher(patterns)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	tests := []struct {
		file     string
		expected bool
		reason   string
	}{
		{"日本語.txt", true, "should match Japanese filename"},
		{"другой日本語.txt", false, "should not match different file"},
		{"файл.log", true, "should match Russian filename"},
		{"🎉celebration.md", true, "should match emoji filename"},
		{"café/test.txt", true, "should match files in café directory"},
		{"cafe/test.txt", false, "café and cafe are different (no accent)"},
		{"normal.txt", false, "should not match ASCII-only files"},
	}

	for _, tt := range tests {
		result, err := matcher.Matches(tt.file)
		if err != nil {
			t.Errorf("Error matching file %s: %v", tt.file, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("File %q: expected %v, got %v - %s", tt.file, tt.expected, result, tt.reason)
		}
	}
}

func TestVeryDeepPaths(t *testing.T) {
	// Test very deep directory hierarchies (100+ levels)
	patterns := []string{
		"**/target.txt", // Should match at any depth
		"deep/**/file",  // Should match in deep subdirectories
	}

	matcher, err := NewPatternMatcher(patterns)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	// Build a very deep path (100 levels)
	deepPath := ""
	for i := 0; i < 100; i++ {
		if i > 0 {
			deepPath += "/"
		}
		deepPath += "level"
	}
	deepPath += "/target.txt"

	// Build another deep path with "deep" at the start
	deepPath2 := "deep"
	for i := 0; i < 50; i++ {
		deepPath2 += "/subdir"
	}
	deepPath2 += "/file"

	tests := []struct {
		file     string
		expected bool
		reason   string
	}{
		{deepPath, true, "should match target.txt at 100 levels deep"},
		{deepPath2, true, "should match file in deep subdirectories"},
		{"level/level/target.txt", true, "should match at shallower depths too"},
	}

	for _, tt := range tests {
		result, err := matcher.Matches(tt.file)
		if err != nil {
			t.Errorf("Error matching file %s: %v", tt.file, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("File %q: expected %v, got %v - %s", tt.file, tt.expected, result, tt.reason)
		}
	}
}

func TestConsecutiveWildcards(t *testing.T) {
	// Test patterns with consecutive wildcards
	patterns := []string{
		"*?*",         // Multiple wildcards: * (0+) + ? (1) + * (0+) = min 1 char
		"?*?",         // Question mark with asterisk: ? (1) + * (0+) + ? (1) = min 2 chars
		"a*?*c.txt",   // Complex pattern
	}

	matcher, err := NewPatternMatcher(patterns)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	tests := []struct {
		file     string
		expected bool
		reason   string
	}{
		{"ab", true, "*?* matches: *(0) + ?(a) + *(b) or *(a) + ?(b) + *(0)"},
		{"abc", true, "*?* should match longer filenames"},
		{"a", true, "*?* matches: *(0) + ?(a) + *(0)"},
		{"axc.txt", true, "a*?*c.txt should match with chars in between"},
		{"abc.txt", true, "a*?*c.txt matches with b in between"},
		{"ac.txt", true, "a*?*c.txt: a + *(0) + ?(c) + *(.txt) works via subpath matching"},
	}

	for _, tt := range tests {
		result, err := matcher.Matches(tt.file)
		if err != nil {
			t.Errorf("Error matching file %s: %v", tt.file, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("File %q: expected %v, got %v - %s", tt.file, tt.expected, result, tt.reason)
		}
	}
}

func TestVeryLongPatterns(t *testing.T) {
	// Test very long patterns (1000+ characters)
	longDirName := ""
	for i := 0; i < 1000; i++ {
		longDirName += "a"
	}

	patterns := []string{
		longDirName + ".txt",
		"*/" + longDirName,
	}

	matcher, err := NewPatternMatcher(patterns)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	tests := []struct {
		file     string
		expected bool
		reason   string
	}{
		{longDirName + ".txt", true, "should match very long filename"},
		{"dir/" + longDirName, true, "should match long filename in subdirectory"},
		{longDirName + "b.txt", false, "should not match different long filename"},
	}

	for _, tt := range tests {
		result, err := matcher.Matches(tt.file)
		if err != nil {
			t.Errorf("Error matching file %s (length %d): %v", "LONG_PATH", len(tt.file), err)
			continue
		}
		if result != tt.expected {
			t.Errorf("File length %d: expected %v, got %v - %s", len(tt.file), tt.expected, result, tt.reason)
		}
	}
}

func TestEdgeCasePatterns(t *testing.T) {
	// Test edge case patterns that might cause issues
	tests := []struct {
		name     string
		patterns []string
		file     string
		expected bool
		reason   string
	}{
		{
			name:     "Multiple consecutive slashes in file path",
			patterns: []string{"src/test.txt"},
			file:     "src//test.txt",
			expected: true,
			reason:   "filepath.Clean normalizes // to /, so this should match",
		},
		{
			name:     "Pattern with wildcard at boundaries",
			patterns: []string{"*test*"},
			file:     "test",
			expected: true,
			reason:   "should match when wildcard matches zero characters",
		},
		{
			name:     "Empty directory name components",
			patterns: []string{"a/b"},
			file:     "a//b",
			expected: true,
			reason:   "filepath.Clean normalizes a//b to a/b, so this should match",
		},
		{
			name:     "Character class edge cases",
			patterns: []string{"[a-z][0-9]"},
			file:     "a0",
			expected: true,
			reason:   "should match character class at boundaries",
		},
		{
			name:     "Multiple wildcards in directory",
			patterns: []string{"**/test/**/*.txt"},
			file:     "a/b/test/c/d/file.txt",
			expected: true,
			reason:   "should match with multiple ** wildcards",
		},
		{
			name:     "Pattern with spaces",
			patterns: []string{"my file.txt"},
			file:     "my file.txt",
			expected: true,
			reason:   "should match filenames with spaces",
		},
		{
			name:     "Very short pattern",
			patterns: []string{"a"},
			file:     "a",
			expected: true,
			reason:   "should match single character patterns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher, err := NewPatternMatcher(tt.patterns)
			if err != nil {
				t.Fatalf("Failed to create matcher: %v", err)
			}

			result, err := matcher.Matches(tt.file)
			if err != nil {
				t.Errorf("Error matching file %s: %v", tt.file, err)
				return
			}
			if result != tt.expected {
				t.Errorf("File %q: expected %v, got %v - %s", tt.file, tt.expected, result, tt.reason)
			}
		})
	}
}

func TestLeadingSlashPatterns(t *testing.T) {
	// Test leading slash patterns (gitignore spec: / prefix means root-relative)
	// Issue: https://github.com/linkwithjoydeep/go-dotignore/issues/5
	patterns := []string{
		"/build/",   // Should match ONLY root-level build/, not nested
		"/test.txt", // Should match ONLY root-level test.txt
		"logs/",     // Should match logs/ anywhere (no leading slash)
	}

	matcher, err := NewPatternMatcher(patterns)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	tests := []struct {
		file     string
		expected bool
		reason   string
	}{
		// Pattern: /build/ (root-relative)
		{"build/", true, "/build/ should match root-level build/"},
		{"build/file.txt", true, "/build/ should match files in root build/"},
		{"src/build/", false, "/build/ should NOT match nested build/ (Issue #5)"},
		{"src/build/file.txt", false, "/build/ should NOT match files in nested build/"},
		{"build", true, "/build/ should match build without trailing slash"},

		// Pattern: /test.txt (root-relative file)
		{"test.txt", true, "/test.txt should match root-level file"},
		{"src/test.txt", false, "/test.txt should NOT match nested file"},
		{"test/test.txt", false, "/test.txt should NOT match in subdirectory"},

		// Pattern: logs/ (non-root-relative, should match anywhere)
		{"logs/", true, "logs/ should match at root"},
		{"logs/app.log", true, "logs/ should match files in root logs/"},
		{"src/logs/", true, "logs/ should match nested logs/ (no leading slash)"},
		{"src/logs/app.log", true, "logs/ should match files in nested logs/"},
	}

	for _, tt := range tests {
		result, err := matcher.Matches(tt.file)
		if err != nil {
			t.Errorf("Error matching file %s: %v", tt.file, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("File %q: expected %v, got %v - %s", tt.file, tt.expected, result, tt.reason)
		}
	}
}

func TestRootRelativeWithWildcards(t *testing.T) {
	// Test root-relative patterns with wildcards
	patterns := []string{
		"/*.txt",      // Only .txt files at root
		"/src/*.go",   // Only .go files in root-level src/
		"/test/**",    // Everything in root-level test/
	}

	matcher, err := NewPatternMatcher(patterns)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	tests := []struct {
		file     string
		expected bool
		reason   string
	}{
		// Pattern: /*.txt
		{"file.txt", true, "/*.txt should match root-level .txt"},
		{"dir/file.txt", false, "/*.txt should NOT match nested .txt"},

		// Pattern: /src/*.go
		{"src/main.go", true, "/src/*.go should match .go in root src/"},
		{"src/util/helper.go", false, "/src/*.go should NOT match nested .go in src/"},
		{"lib/src/main.go", false, "/src/*.go should NOT match in nested src/ directory"},

		// Pattern: /test/**
		{"test/file.txt", true, "/test/** should match files in root test/"},
		{"test/sub/file.txt", true, "/test/** should match deeply nested in root test/"},
		{"src/test/file.txt", false, "/test/** should NOT match in nested test/ directory"},
	}

	for _, tt := range tests {
		result, err := matcher.Matches(tt.file)
		if err != nil {
			t.Errorf("Error matching file %s: %v", tt.file, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("File %q: expected %v, got %v - %s", tt.file, tt.expected, result, tt.reason)
		}
	}
}

func TestComplexPatterns(t *testing.T) {
	patterns := []string{
		"**/*.log",        // All .log files anywhere
		"!debug/**/*.log", // Except .log files in debug directory
		"temp/**",         // Everything in temp directory
		"*.tmp",           // All .tmp files
		"docs/",           // docs directory
		"!docs/README.md", // Except README.md in docs
	}

	matcher, err := NewPatternMatcher(patterns)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	tests := []struct {
		file     string
		expected bool
	}{
		{"app.log", true},
		{"src/app.log", true},
		{"debug/app.log", false}, // Negated by !debug/**/*.log
		{"debug/sub/app.log", false},
		{"temp/file.txt", true},
		{"temp/sub/file.txt", true},
		{"file.tmp", true},
		{"docs/guide.md", true},
		{"docs/README.md", false}, // Negated
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			result, err := matcher.Matches(tt.file)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("File %q: expected %v, got %v", tt.file, tt.expected, result)
			}
		})
	}
}

func TestWindowsPaths(t *testing.T) {
	patterns := []string{"src\\*.txt", "build\\"}
	matcher, err := NewPatternMatcher(patterns)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	// Test that both forward and backward slashes work
	tests := []struct {
		file     string
		expected bool
	}{
		{"src/test.txt", true},
		{"src\\test.txt", true},
		{"build/", true},
		{"build\\", true},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			result, err := matcher.Matches(tt.file)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("File %q: expected %v, got %v", tt.file, tt.expected, result)
			}
		})
	}
}

func TestPatternOrderMatters(t *testing.T) {
	// Test that pattern order affects the final result
	patterns1 := []string{"*.txt", "!important.txt"}
	patterns2 := []string{"!important.txt", "*.txt"}

	matcher1, _ := NewPatternMatcher(patterns1)
	matcher2, _ := NewPatternMatcher(patterns2)

	file := "important.txt"

	result1, _ := matcher1.Matches(file)
	result2, _ := matcher2.Matches(file)

	// With patterns1, important.txt should not be ignored (false)
	// With patterns2, important.txt should be ignored (true)
	if result1 != false {
		t.Errorf("With patterns1, expected false, got %v", result1)
	}
	if result2 != true {
		t.Errorf("With patterns2, expected true, got %v", result2)
	}
}

func BenchmarkMatches(b *testing.B) {
	patterns := []string{
		"*.log", "*.tmp", "*.cache",
		"build/", "dist/", "node_modules/",
		"**/*.test.js", "**/*.spec.js",
		"!important.log", "!src/**/*.test.js",
	}

	matcher, err := NewPatternMatcher(patterns)
	if err != nil {
		b.Fatalf("Failed to create matcher: %v", err)
	}

	testFiles := []string{
		"app.log",
		"src/component.js",
		"src/component.test.js",
		"build/app.js",
		"node_modules/package/index.js",
		"important.log",
		"cache.tmp",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, file := range testFiles {
			_, _ = matcher.Matches(file)
		}
	}
}
