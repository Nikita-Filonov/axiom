package testquarantine

import "strings"

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func hasTag(tags []string, tag string) bool {
	tag = normalize(tag)
	if tag == "" {
		return false
	}

	for _, t := range tags {
		if normalize(t) == tag {
			return true
		}
	}

	return false
}

func parseBool(s string) (value bool, ok bool) {
	switch normalize(s) {
	case "1", "true", "yes", "on":
		return true, true
	case "0", "false", "no", "off":
		return false, true
	default:
		return false, false
	}
}
