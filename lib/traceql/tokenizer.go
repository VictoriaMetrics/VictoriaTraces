package traceql

import (
	"unicode"
	"unicode/utf8"
)

type tokenizer struct {
	m map[string]struct{}
}

func (t *tokenizer) reset() {
	clear(t.m)
}

func isTokenChar(c byte) bool {
	return tokenCharTable[c] != 0
}

var tokenCharTable = func() *[256]byte {
	var a [256]byte
	for c := uint(0); c < 256; c++ {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '.' {
			a[c] = 1
		}
	}
	return &a
}()

func isTokenRune(c rune) bool {
	if c < utf8.RuneSelf {
		// Fast path - the char is ASCII
		return isTokenChar(byte(c))
	}
	return unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_' || c == '.'
}
