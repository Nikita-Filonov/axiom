package testtags

import (
	"strings"
)

func NormalizeTag(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

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

func MapList(list []string) map[string]struct{} {
	m := make(map[string]struct{}, len(list))
	for _, v := range list {
		m[NormalizeTag(v)] = struct{}{}
	}
	return m
}

func Intersects(set map[string]struct{}, list []string) bool {
	for _, v := range list {
		if _, ok := set[v]; ok {
			return true
		}
	}
	return false
}
