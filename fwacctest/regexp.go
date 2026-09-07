package fwacctest

import (
	"regexp"
	"strings"
)

func ExpectLiteralError(text string) *regexp.Regexp {
	words := strings.Split(text, " ")
	for i, word := range words {
		words[i] = regexp.QuoteMeta(word)
	}
	return regexp.MustCompile(strings.Join(words, `\s+`))
}
