package testtags

import (
	"strings"
)

// NormalizeTag trims whitespace and folds a tag to lowercase.
func NormalizeTag(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// ParseList splits a comma-separated tag list and removes empty entries.
func ParseList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		if v := NormalizeTag(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// MapList returns a set of normalized tags.
func MapList(list []string) map[string]struct{} {
	m := make(map[string]struct{}, len(list))
	for _, v := range list {
		m[NormalizeTag(v)] = struct{}{}
	}
	return m
}

// Intersects reports whether list contains an element of set.
func Intersects(set map[string]struct{}, list []string) bool {
	for _, v := range list {
		if _, ok := set[v]; ok {
			return true
		}
	}
	return false
}
