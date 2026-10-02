package testleaks

import (
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleReleaseAndResourceOrder(t *testing.T) {
	var absent *Handle
	absent.Release()
	state := newAttemptState()
	first := &Handle{state: state, id: state.add("first", "first.go:1")}
	state.add("second", "second.go:2")
	resources := state.snapshot()
	require.Len(t, resources, 2)
	assert.Equal(t, "first", resources[0].name)
	assert.Equal(t, "second", resources[1].name)
	var workers sync.WaitGroup
	for i := 0; i < 8; i++ {
		workers.Go(first.Release)
	}
	workers.Wait()
	first.Release()
	resources = state.snapshot()
	require.Len(t, resources, 1)
	assert.Equal(t, "second", resources[0].name)
}

func TestTrackNilConfig(t *testing.T) {
	assert.Panics(t, func() { Track(nil, "socket") })
}

type failingReadCloser struct{ *strings.Reader }

func (f *failingReadCloser) Close() error { return errors.New("close failed") }

func TestTrackReadCloserReleasesAfterCloseError(t *testing.T) {
	cfg := &axiom.Config{}
	Plugin(WithoutGoroutines())(cfg)
	closer := TrackReadCloser(cfg, "response body", &failingReadCloser{strings.NewReader("body")})
	_, err := io.ReadAll(closer)
	require.NoError(t, err)
	require.EqualError(t, closer.Close(), "close failed")
	state, ok := axiom.GetLocal(cfg, stateKey)
	require.True(t, ok)
	require.NotNil(t, state)
	assert.Empty(t, state.snapshot(), "resource still tracked after Close")
}

func TestTrackReadCloserTypedNil(t *testing.T) {
	var value *failingReadCloser
	assert.Panics(t, func() { TrackReadCloser(&axiom.Config{}, "body", value) })
}
