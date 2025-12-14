package testlogger

import (
	"log/slog"
	"os"

	"github.com/Nikita-Filonov/axiom"
)

func Plugin() axiom.Plugin {
	return func(cfg *axiom.Config) {
		logger := slog.New(
			slog.NewTextHandler(
				os.Stdout,
				&slog.HandlerOptions{
					Level: slog.LevelDebug,
				},
			),
		)

		cfg.Runtime.EmitLogSink(func(l axiom.Log) {
			level := MapLevel(l.Level)
			logger.Log(cfg.Context.Raw, level, l.Text)
		})
	}
}
