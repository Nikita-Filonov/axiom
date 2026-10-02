package testleaks_test

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testleaks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			assert.NoError(cfg.T(), body.Close())
		})
		payload, err := io.ReadAll(body)
		require.NoError(cfg.T(), err)
		assert.Equal(cfg.T(), "ok", string(payload))

		// Track any other resource until its actual cleanup completes.
		subscription := testleaks.Track(cfg, "temporary subscription")
		cfg.T().Cleanup(subscription.Release)
	})
}
