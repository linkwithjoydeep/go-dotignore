module github.com/codeglyph/go-dotignore

go 1.20

// Retract versions with critical bugs - users should upgrade to v2.0.0+
retract [v1.0.0, v1.1.1] // Critical bugs: substring matching, root-relative patterns broken, no escaped negation support. Fixed in v2.0.0
