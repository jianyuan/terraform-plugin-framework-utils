// Package fwacctest provides helpers for writing acceptance tests for
// Terraform Plugin Framework providers.
package fwacctest

import (
	"regexp"
	"strings"
)

// ExpectLiteralError returns a regular expression that matches text literally,
// allowing any whitespace between words. Terraform wraps long error messages
// across lines, so this is useful with resource.TestStep.ExpectError.
func ExpectLiteralError(text string) *regexp.Regexp {
	words := strings.Split(text, " ")
	for i, word := range words {
		words[i] = regexp.QuoteMeta(word)
	}
	return regexp.MustCompile(strings.Join(words, `\s+`))
}
