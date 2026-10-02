package testleaks

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/Nikita-Filonov/axiom"
)

// Handle marks one explicitly tracked resource as released. Release is safe
// to call more than once or concurrently.
type Handle struct {
	once  sync.Once
	state *attemptState
	id    uint64
}

// Release removes the resource from the attempt's open-resource set.
func (h *Handle) Release() {
	if h == nil {
		return
	}
	h.once.Do(func() { h.state.release(h.id) })
}

// Track registers a resource owned by this attempt. Call Release when its
// actual cleanup completes. It panics if Plugin was not applied to cfg.
func Track(cfg *axiom.Config, name string) *Handle {
	return track(cfg, name)
}

func track(cfg *axiom.Config, name string) *Handle {
	if cfg == nil {
		panic("testleaks: nil config")
	}
	if name == "" {
		panic("testleaks: empty resource name")
	}
	state, ok := axiom.GetLocal(cfg, stateKey)
	if !ok {
		panic("testleaks: plugin not installed")
	}
	_, file, line, _ := runtime.Caller(2)
	id := state.add(name, fmt.Sprintf("%s:%d", file, line))
	return &Handle{state: state, id: id}
}
