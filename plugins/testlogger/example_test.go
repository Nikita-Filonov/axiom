package testlogger_test

import (
	"bytes"
	"fmt"
	"log/slog"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testlogger"
)

func ExampleWithLogger() {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return attr
		},
	})).With("service", "payments")

	cfg := &axiom.Config{}
	testlogger.Plugin(testlogger.WithLogger(logger))(cfg)
	cfg.Log(axiom.NewDebugLog("filtered"))
	cfg.Log(axiom.NewInfoLog("request sent"))
	fmt.Print(output.String())
	// Output: {"level":"INFO","msg":"request sent","service":"payments"}
}
