package utils

import (
	"strings"
	"unicode/utf8"
)

// CleanUTF8 removes or replaces invalid UTF8 characters from a string
// Returns the cleaned string and a boolean indicating if cleaning was needed
func CleanUTF8(input string) (string, bool) {
	needsCleaning := strings.Contains(input, "\x00") || !utf8.ValidString(input)

	if !needsCleaning {
		return input, false
	}

	cleaned := strings.ToValidUTF8(input, "")
	cleaned = strings.ReplaceAll(cleaned, "\x00", "")

	return cleaned, true
}

