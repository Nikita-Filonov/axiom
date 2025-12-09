package testtags

import (
	"github.com/Nikita-Filonov/axiom"
)

func Plugin(options ...ConfigOption) axiom.Plugin {
	cfg := NewConfig(options...)

	return func(e *axiom.Config) {
		testTags := MapList(e.Meta.Tags)

		if len(cfg.Exclude) > 0 && Intersects(testTags, cfg.Exclude) {
			e.Skip = axiom.Skip{Enabled: true, Reason: "excluded by tag filter"}
			return
		}

		if len(cfg.Include) > 0 && !Intersects(testTags, cfg.Include) {
			e.Skip = axiom.Skip{Enabled: true, Reason: "not included by tag filter"}
			return
		}
	}
}
