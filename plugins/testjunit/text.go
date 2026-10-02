package testjunit

import "strings"

// cleanXML replaces code points excluded by XML 1.0. Invalid UTF-8 is
// converted to RuneError by Go's range decoder.
func cleanXML(value string) string {
	return strings.Map(func(r rune) rune {
		if r == '\t' || r == '\n' || r == '\r' ||
			(r >= 0x20 && r <= 0xd7ff) ||
			(r >= 0xe000 && r <= 0xfffd) ||
			(r >= 0x10000 && r <= 0x10ffff) {
			return r
		}
		return '\ufffd'
	}, value)
}
