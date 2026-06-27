package testtags

import (
	"github.com/Nikita-Filonov/axiom"
)

func Plugin(options ...ConfigOption) axiom.Plugin {
	cfg := NewConfig(options...)

	return func(e *axiom.Config) {
		testTags := MapList(e.Meta.Tags)

		if len(cfg.Exclude) > 0 && Intersects(testTags, cfg.Exclude) {
			e.Skip = axiom.NewSkip(axiom.SkipBecause("excluded by tag filter"))
			return
		}

		if len(cfg.Include) > 0 && !Intersects(testTags, cfg.Include) {
			e.Skip = axiom.NewSkip(axiom.SkipBecause("not included by tag filter"))
			return
		}
	}
}
