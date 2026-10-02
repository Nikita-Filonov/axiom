package testjunit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigAndNilReporter(t *testing.T) {
	assert.Equal(t, "axiom", newConfig().SuiteName)
	assert.Equal(t, "package", newConfig(WithSuiteName("package")).SuiteName)
	assert.Equal(t, "last", newConfig(WithSuiteName("first"), WithSuiteName("last")).SuiteName)
	for _, test := range []struct {
		name string
		fn   func()
	}{
		{"nil option", func() { Plugin(NewReporter(), nil) }},
		{"empty suite name", func() { WithSuiteName("") }},
		{"nil reporter plugin", func() { Plugin(nil) }},
		{"nil config", func() { Plugin(NewReporter())(nil) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			assert.Panics(t, test.fn)
		})
	}
}
