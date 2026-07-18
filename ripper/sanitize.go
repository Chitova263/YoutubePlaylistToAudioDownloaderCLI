package ripper

import (
	"strings"
	"unicode"
)

// illegalChars are characters forbidden in Windows file/directory names.
const illegalChars = `<>:"/\|?*`

// SanitizePath cleans a string for use as a file or directory name.
// It removes characters illegal on Windows, collapses whitespace,
// trims trailing dots/spaces (Windows restriction), and caps length at 200 chars.
func SanitizePath(name string) string {
	var b strings.Builder
	b.Grow(len(name))

	for _, r := range name {
		if strings.ContainsRune(illegalChars, r) {
			continue
		}
		// Replace control characters with nothing
		if unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}

	// Collapse multiple spaces into one
	result := collapseSpaces(b.String())

	// Trim leading/trailing whitespace and trailing dots (Windows restriction)
	result = strings.TrimSpace(result)
	result = strings.TrimRight(result, ".")

	// Cap length to avoid MAX_PATH issues
	if len(result) > 200 {
		result = result[:200]
		result = strings.TrimSpace(result)
	}

	// If the name is completely empty after sanitization, use a fallback
	if result == "" {
		result = "_unnamed"
	}

	return result
}

func collapseSpaces(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' {
			if !prevSpace {
				b.WriteRune(' ')
			}
			prevSpace = true
		} else {
			b.WriteRune(r)
			prevSpace = false
		}
	}
	return b.String()
}
