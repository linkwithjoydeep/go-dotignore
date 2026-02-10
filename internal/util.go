package internal

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// ReadLines reads lines from an io.Reader and strips UTF-8 BOM characters.
func ReadLines(reader io.Reader) ([]string, error) {
	if reader == nil {
		return nil, fmt.Errorf("reader cannot be nil")
	}

	scanner := bufio.NewScanner(reader)
	var lines []string
	utf8BOM := []byte{0xEF, 0xBB, 0xBF}

	for lineNumber := 0; scanner.Scan(); lineNumber++ {
		line := scanner.Bytes()
		if lineNumber == 0 {
			line = bytes.TrimPrefix(line, utf8BOM)
		}
		lines = append(lines, string(line))
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading lines: %w", err)
	}

	return lines, nil
}

// BuildRegex converts a gitignore-style pattern to a regular expression.
// It properly handles wildcards, escaping, and gitignore-specific rules.
func BuildRegex(pattern string) (*regexp.Regexp, error) {
	if pattern == "" {
		return nil, fmt.Errorf("pattern cannot be empty")
	}

	var sb strings.Builder
	sb.WriteString("^")

	for i := 0; i < len(pattern); i++ {
		switch char := pattern[i]; char {
		case '*':
			i = writeWildcard(pattern, i, &sb)
		case '?':
			sb.WriteString("[^/]")
		case '[':
			i = writeCharClass(pattern, i, &sb)
		case '.', '+', '^', '$', '(', ')', '{', '}', '|':
			sb.WriteByte('\\')
			sb.WriteByte(char)
		case '\\':
			i = writeEscaped(pattern, i, &sb)
		default:
			sb.WriteByte(char)
		}
	}

	sb.WriteString("$")

	regex, err := regexp.Compile(sb.String())
	if err != nil {
		return nil, fmt.Errorf("failed to compile regex %q: %w", sb.String(), err)
	}
	return regex, nil
}

// writeWildcard writes the regex equivalent of * or ** at position i and returns the new index.
func writeWildcard(pattern string, i int, sb *strings.Builder) int {
	if i+1 < len(pattern) && pattern[i+1] == '*' {
		i++ // consume second '*'
		if i+1 < len(pattern) && pattern[i+1] == '/' {
			i++ // consume '/'
			sb.WriteString("(.*?/)?")
		} else {
			sb.WriteString(".*")
		}
	} else {
		sb.WriteString("[^/]*")
	}
	return i
}

// writeCharClass writes a character class [...] and returns the new index.
func writeCharClass(pattern string, i int, sb *strings.Builder) int {
	j := i + 1
	for j < len(pattern) && pattern[j] != ']' {
		j++
	}
	if j < len(pattern) {
		sb.WriteString(pattern[i : j+1])
		return j
	}
	sb.WriteString("\\[")
	return i
}

// writeEscaped writes an escaped character and returns the new index.
func writeEscaped(pattern string, i int, sb *strings.Builder) int {
	if i+1 < len(pattern) {
		i++
		if isRegexMetaChar(pattern[i]) {
			sb.WriteByte('\\')
		}
		sb.WriteByte(pattern[i])
	} else {
		sb.WriteString("\\\\")
	}
	return i
}

// isRegexMetaChar checks if a character has special meaning in regex
func isRegexMetaChar(char byte) bool {
	switch char {
	case '.', '+', '*', '?', '^', '$', '(', ')', '[', ']', '{', '}', '|', '\\':
		return true
	default:
		return false
	}
}
