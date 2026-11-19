package testlogger

import "github.com/Nikita-Filonov/axiom"

// Plugin forwards Axiom logs to a slog logger using the current cfg.Context.Raw.
// By default it writes text to stdout at Debug level and above. Options apply
// once, in order, when Plugin is called; the last logger wins. Nil options or a
// nil configured logger panic. The logger is shared across installations and
// attempts. Each installation appends one log sink.
func Plugin(options ...Option) axiom.Plugin {
	logger := newConfig(options...).Logger
	return func(cfg *axiom.Config) {
		cfg.Runtime.EmitLogSink(func(l axiom.Log) {
			level := MapLevel(l.Level)
			logger.Log(cfg.Context.Raw, level, l.Text)
		})
	}
}
