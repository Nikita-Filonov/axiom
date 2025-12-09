package testtags_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom/plugins/testtags"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeTag(t *testing.T) {
	tests := []struct {
		in     string
		expect string
	}{
		{" fast ", "fast"},
		{"API", "api"},
		{"  Db ", "db"},
		{"", ""},
		{"   ", ""},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expect, testtags.NormalizeTag(tt.in))
	}
}

func TestParseList(t *testing.T) {
	tests := []struct {
		in     string
		expect []string
	}{
		{"fast,api,DB", []string{"fast", "api", "db"}},
		{" fast , api ,  db ", []string{"fast", "api", "db"}},
		{"one,,two", []string{"one", "two"}},
		{"", nil},
		{"   ", nil},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expect, testtags.ParseList(tt.in))
	}
}

func TestMapList(t *testing.T) {
	list := []string{"API", " fast ", "db"}
	m := testtags.MapList(list)

	// Keys must be normalized
	_, ok1 := m["api"]
	_, ok2 := m["fast"]
	_, ok3 := m["db"]

	assert.True(t, ok1)
	assert.True(t, ok2)
	assert.True(t, ok3)

	// Length must match unique tags
	assert.Equal(t, 3, len(m))
}

func TestIntersects(t *testing.T) {
	set := map[string]struct{}{
		"fast": {},
		"api":  {},
	}

	tests := []struct {
		list   []string
		expect bool
	}{
		{[]string{"fast"}, true},
		{[]string{"api"}, true},
		{[]string{"API"}, false}, // not normalized here!
		{[]string{"slow"}, false},
		{[]string{}, false},
		{nil, false},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expect, testtags.Intersects(set, tt.list))
	}
}
