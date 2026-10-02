package testleaks_test

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testleaks"
)

func TestLeakChecksExample(t *testing.T) {
	// Install the checker for every attempt executed by this Runner.
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(
			testleaks.Plugin(
				testleaks.WithGracePeriod(300 * time.Millisecond),
			),
		),
	)

	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("read response")), func(cfg *axiom.Config) {
		// Workers started inside the attempt inherit its pprof label.
		stop := make(chan struct{})
		stopped := make(chan struct{})
		go func() {
			defer close(stopped)
			<-stop
		}()
		cfg.T().Cleanup(func() {
			close(stop)
			<-stopped
		})

		// Closing a tracked ReadCloser releases its resource handle.
		body := testleaks.TrackReadCloser(cfg, "response body", io.NopCloser(strings.NewReader("ok")))
		cfg.T().Cleanup(func() {
			if err := body.Close(); err != nil {
				cfg.T().Error(err)
			}
		})
		payload, err := io.ReadAll(body)
		if err != nil || string(payload) != "ok" {
			cfg.T().Fatalf("read response: payload=%q, err=%v", payload, err)
		}

		// Track any other resource until its actual cleanup completes.
		subscription := testleaks.Track(cfg, "temporary subscription")
		cfg.T().Cleanup(subscription.Release)
	})
}
