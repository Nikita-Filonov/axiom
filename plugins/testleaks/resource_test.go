package testleaks

import (
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/Nikita-Filonov/axiom"
)

func TestHandleReleaseAndResourceOrder(t *testing.T) {
	var absent *Handle
	absent.Release()
	state := newAttemptState()
	first := &Handle{state: state, id: state.add("first", "first.go:1")}
	state.add("second", "second.go:2")
	resources := state.snapshot()
	if len(resources) != 2 || resources[0].name != "first" || resources[1].name != "second" {
		t.Fatalf("resource order = %+v", resources)
	}
	var workers sync.WaitGroup
	for i := 0; i < 8; i++ {
		workers.Go(first.Release)
	}
	workers.Wait()
	first.Release()
	resources = state.snapshot()
	if len(resources) != 1 || resources[0].name != "second" {
		t.Fatalf("resources after release = %+v", resources)
	}
}

func TestTrackNilConfig(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("Track(nil) did not panic")
		}
	}()
	Track(nil, "socket")
}

type failingReadCloser struct{ *strings.Reader }

func (f *failingReadCloser) Close() error { return errors.New("close failed") }

func TestTrackReadCloserReleasesAfterCloseError(t *testing.T) {
	cfg := &axiom.Config{}
	Plugin(WithoutGoroutines())(cfg)
	closer := TrackReadCloser(cfg, "response body", &failingReadCloser{strings.NewReader("body")})
	if _, err := io.ReadAll(closer); err != nil {
		t.Fatal(err)
	}
	if err := closer.Close(); err == nil || err.Error() != "close failed" {
		t.Fatalf("Close error = %v", err)
	}
	state, ok := axiom.GetLocal(cfg, stateKey)
	if !ok || len(state.snapshot()) != 0 {
		t.Fatalf("resource still tracked after Close: %+v", state)
	}
}

func TestTrackReadCloserTypedNil(t *testing.T) {
	var value *failingReadCloser
	defer func() {
		if recover() == nil {
			t.Error("typed nil ReadCloser did not panic")
		}
	}()
	TrackReadCloser(&axiom.Config{}, "body", value)
}
