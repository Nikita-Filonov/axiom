// Package testlogger forwards Axiom logs to the standard library log/slog package.
// By default, [Plugin] writes text logs to stdout at Debug level and above:
//
//	runner := axiom.NewRunner(
//		axiom.WithRunnerPlugins(testlogger.Plugin()),
//	)
//
// Use [WithLogger] to configure the output, format, filtering, and attributes:
//
//	logger := slog.New(
//		slog.NewJSONHandler(
//			os.Stderr,
//			&slog.HandlerOptions{Level: slog.LevelInfo},
//		),
//	).With("service", "payments")
//
//	runner := axiom.NewRunner(
//		axiom.WithRunnerPlugins(
//			testlogger.Plugin(testlogger.WithLogger(logger)),
//		),
//	)
//
// Each log is forwarded with its message, the level converted by [MapLevel],
// and the current cfg.Context.Raw. The logger is shared across test attempts;
// its handler must support concurrent calls. The caller owns the handler and
// output, including any flushing or closing.
package testlogger
