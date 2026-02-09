package dotignore_test

import (
	"fmt"
	"log"

	"github.com/codeglyph/go-dotignore/v2"
)

// ExampleNewRepositoryMatcher demonstrates hierarchical .gitignore matching
// across a repository with nested .gitignore files.
func ExampleNewRepositoryMatcher() {
	// For this example, assume we have a repository structure like:
	// /path/to/repo/
	//   .gitignore          (*.log, .env)
	//   frontend/
	//     .gitignore        (node_modules/, dist/)
	//   backend/
	//     .gitignore        (target/, *.class)

	matcher, err := dotignore.NewRepositoryMatcher("/path/to/repo")
	if err != nil {
		log.Fatal(err)
	}

	// Check files against hierarchical patterns
	files := []string{
		"app.log",                              // Matched by root .gitignore
		"frontend/node_modules/pkg/index.js",   // Matched by frontend/.gitignore
		"backend/target/classes/Main.class",    // Matched by backend/.gitignore
		"frontend/src/App.js",                  // Not matched
	}

	for _, file := range files {
		ignored, _ := matcher.Matches(file)
		fmt.Printf("%-40s ignored: %v\n", file, ignored)
	}
}

// ExampleRepositoryMatcher_Matches demonstrates how nested .gitignore files
// can override parent patterns using negation.
func ExampleRepositoryMatcher_Matches() {
	// Assume repository structure:
	// /path/to/repo/
	//   .gitignore          (*.txt)
	//   important/
	//     .gitignore        (!critical.txt)

	matcher, err := dotignore.NewRepositoryMatcher("/path/to/repo")
	if err != nil {
		log.Fatal(err)
	}

	files := []string{
		"file.txt",                  // Ignored by root
		"important/file.txt",        // Still ignored by root
		"important/critical.txt",    // Un-ignored by important/.gitignore
	}

	for _, file := range files {
		ignored, _ := matcher.Matches(file)
		status := "ignored"
		if !ignored {
			status = "not ignored"
		}
		fmt.Printf("%-30s %s\n", file, status)
	}
}

// ExampleNewRepositoryMatcherWithConfig demonstrates using custom configuration
// for repository matching, such as limiting depth or using custom ignore file names.
func ExampleNewRepositoryMatcherWithConfig() {
	config := &dotignore.RepositoryConfig{
		IgnoreFileName: ".ignore",  // Use .ignore instead of .gitignore
		MaxDepth:       3,           // Only search 3 levels deep
		FollowSymlinks: false,       // Don't follow symbolic links
	}

	matcher, err := dotignore.NewRepositoryMatcherWithConfig("/path/to/repo", config)
	if err != nil {
		log.Fatal(err)
	}

	// Get count of discovered ignore files
	count := matcher.IgnoreFileCount()
	fmt.Printf("Found %d ignore files\n", count)

	// Get paths of all discovered ignore files
	paths := matcher.IgnoreFilePaths()
	for _, path := range paths {
		fmt.Printf("Loaded: %s\n", path)
	}
}

// ExampleRepositoryMatcher_monorepo demonstrates using RepositoryMatcher
// in a monorepo scenario with multiple subprojects.
func ExampleRepositoryMatcher_monorepo() {
	// Typical monorepo structure:
	// project/
	//   .gitignore              (*.log, .DS_Store)
	//   frontend/
	//     .gitignore            (node_modules/, dist/)
	//   backend/
	//     .gitignore            (target/, *.class)
	//   docs/
	//     .gitignore            (_build/, *.pyc)

	matcher, err := dotignore.NewRepositoryMatcher("/path/to/project")
	if err != nil {
		log.Fatal(err)
	}

	// Global patterns apply everywhere
	fmt.Println("Global patterns:")
	globalFiles := []string{
		"app.log",           // Matched by root
		"frontend/app.log",  // Also matched by root
		".DS_Store",         // Matched by root
	}
	for _, file := range globalFiles {
		ignored, _ := matcher.Matches(file)
		fmt.Printf("  %-30s %v\n", file, ignored)
	}

	// Subproject-specific patterns
	fmt.Println("\nSubproject patterns:")
	subprojectFiles := []string{
		"frontend/node_modules/react/index.js",  // Frontend specific
		"backend/target/output.jar",              // Backend specific
		"docs/_build/html/index.html",            // Docs specific
	}
	for _, file := range subprojectFiles {
		ignored, _ := matcher.Matches(file)
		fmt.Printf("  %-40s %v\n", file, ignored)
	}
}
