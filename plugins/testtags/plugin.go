package testtags

import (
	"github.com/Nikita-Filonov/axiom"
)

// Plugin excludes cases that fail the configured tag filter.
func Plugin(options ...ConfigOption) axiom.Plugin {
	cfg := NewConfig(options...)

	return func(e *axiom.Config) {
		testTags := MapList(e.Meta.Tags)

		if len(cfg.Exclude) > 0 && Intersects(testTags, cfg.Exclude) {
			e.Attempt.Exclude("excluded by tag filter")
			return
		}

		if len(cfg.Include) > 0 && !Intersects(testTags, cfg.Include) {
			e.Attempt.Exclude("not included by tag filter")
			return
		}
	}
}
