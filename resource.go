package axiom

import (
	"fmt"
	"sync"
)

// Resource constructs a runner scoped value on first access through
// [GetResource]. Successful values are shared across cases and retry attempts.
// An optional cleanup runs after the runner's AfterAll hooks.
type Resource func(r *Runner) (any, func(), error)

// Resources stores definitions and values shared by a Runner. Concurrent
// [GetResource] calls for the same name coordinate a single construction.
type Resources struct {
	Registry map[string]Resource

	mu       *sync.Mutex
	onces    map[string]*resourceOnce
	cache    map[string]any
	cleanups []func(*Runner)
}

type resourceOnce struct {
	once  sync.Once
	value any
	err   error
}

// ResourcesOption configures a Resources registry.
type ResourcesOption func(*Resources)

// NewResources creates a resource registry with the supplied options.
func NewResources(options ...ResourcesOption) Resources {
	r := Resources{}
	for _, option := range options {
		option(&r)
	}
	return r
}

// WithResource registers a named resource definition in Resources.
func WithResource(name string, resource Resource) ResourcesOption {
	return func(r *Resources) {
		if r.Registry == nil {
			r.Registry = map[string]Resource{}
		}
		r.Registry[name] = resource
	}
}

// WithResourcesMap copies named definitions into Resources, replacing matching
// names without taking ownership of the input map.
func WithResourcesMap(resources map[string]Resource) ResourcesOption {
	return func(r *Resources) {
		if r.Registry == nil {
			r.Registry = map[string]Resource{}
		}
		for k, v := range resources {
			r.Registry[k] = v
		}
	}
}

// Copy returns Resources with independent maps and cleanup list. Cached values
// and cleanup functions remain shared.
func (r *Resources) Copy() Resources {
	result := Resources{mu: &sync.Mutex{}}

	if r.Registry != nil {
		result.Registry = make(map[string]Resource, len(r.Registry))
		for k, v := range r.Registry {
			result.Registry[k] = v
		}
	}
	if r.cache != nil {
		result.cache = make(map[string]any, len(r.cache))
		for k, v := range r.cache {
			result.cache[k] = v
		}
	}
	if r.cleanups != nil {
		result.cleanups = append([]func(*Runner){}, r.cleanups...)
	}

	return result
}

// Join merges definitions, cached values, and cleanup callbacks from other
// into a new Resources value. This intentionally preserves already constructed
// values by pointer. Each Runner retains its own cleanup stack, so a callback
// copied into a joined Runner runs during that Runner's teardown as well.
func (r *Resources) Join(other Resources) Resources {
	result := r.Copy()

	if len(other.Registry) > 0 {
		if result.Registry == nil {
			result.Registry = make(map[string]Resource, len(other.Registry))
		}
		for k, v := range other.Registry {
			result.Registry[k] = v
		}
	}

	if len(other.cache) > 0 {
		if result.cache == nil {
			result.cache = make(map[string]any, len(other.cache))
		}
		for k, v := range other.cache {
			result.cache[k] = v
		}
	}

	if len(other.cleanups) > 0 {
		result.cleanups = append(result.cleanups, other.cleanups...)
	}

	return result
}

// Normalize initializes internal synchronization and missing maps.
func (r *Resources) Normalize() {
	if r.mu == nil {
		r.mu = &sync.Mutex{}
	}
	if r.Registry == nil {
		r.Registry = map[string]Resource{}
	}
	if r.cache == nil {
		r.cache = map[string]any{}
	}
	if r.onces == nil {
		r.onces = map[string]*resourceOnce{}
	}
}

// teardown runs registered resource cleanups in reverse order.
func (r *Resources) teardown(runner *Runner) {
	for i := len(r.cleanups) - 1; i >= 0; i-- {
		r.cleanups[i](runner)
	}
	r.cleanups = nil
}

// GetResource returns the named resource, constructing it once per runner on
// first use. Concurrent callers share the result, including a construction
// error, which remains cached for that runner. It returns an error for a
// missing resource, construction failure, or type mismatch.
func GetResource[T any](runner *Runner, name string) (T, error) {
	var zero T

	runner.Resources.Normalize()

	runner.Resources.mu.Lock()
	if cached, ok := runner.Resources.cache[name]; ok {
		runner.Resources.mu.Unlock()
		out, ok := cached.(T)
		if !ok {
			return zero, fmt.Errorf("resource %q has unexpected type", name)
		}
		return out, nil
	}

	resource, ok := runner.Resources.Registry[name]
	if !ok {
		runner.Resources.mu.Unlock()
		runner.Runtime.event(NewEvent(EventTypeResourceSetupFailed, WithEventName(name), WithEventMessage("not found")))
		return zero, fmt.Errorf("resource %q not found", name)
	}

	ro, exists := runner.Resources.onces[name]
	if !exists {
		ro = &resourceOnce{}
		runner.Resources.onces[name] = ro
	}
	runner.Resources.mu.Unlock()

	ro.once.Do(func() {
		runner.Runtime.event(NewEvent(EventTypeResourceSetupStart, WithEventName(name)))
		val, cleanup, err := resource(runner)
		if err != nil {
			ro.err = err
			runner.Runtime.event(NewEvent(EventTypeResourceSetupFailed, WithEventName(name), WithEventMessage(err.Error())))
			return
		}

		ro.value = val

		runner.Resources.mu.Lock()
		runner.Resources.cache[name] = val
		if cleanup != nil {
			runner.Resources.cleanups = append(runner.Resources.cleanups, resourceCleanupHook(name, cleanup))
		}
		runner.Resources.mu.Unlock()

		runner.Runtime.event(NewEvent(EventTypeResourceSetupFinish, WithEventName(name)))
	})

	if ro.err != nil {
		return zero, fmt.Errorf("resource %q failed: %w", name, ro.err)
	}

	out, ok := ro.value.(T)
	if !ok {
		return zero, fmt.Errorf("resource %q has unexpected type", name)
	}
	return out, nil
}

// MustResource returns the named resource or panics if [GetResource] fails.
func MustResource[T any](runner *Runner, name string) T {
	v, err := GetResource[T](runner, name)
	if err != nil {
		panic(err)
	}
	return v
}

// UseResources returns a callback that initializes named runner resources.
func UseResources(names ...string) func(r *Runner) {
	return func(r *Runner) {
		for _, name := range names {
			MustResource[any](r, name)
		}
	}
}

func resourceCleanupHook(name string, cleanup func()) func(*Runner) {
	return func(r *Runner) {
		r.Runtime.event(NewEvent(EventTypeResourceCleanupStart, WithEventName(name)))
		defer func() {
			if v := recover(); v != nil {
				r.Runtime.event(NewEvent(EventTypeResourceCleanupPanic, WithEventName(name), WithEventMessage(v)))
				panic(v)
			}

			r.Runtime.event(NewEvent(EventTypeResourceCleanupFinish, WithEventName(name)))
		}()

		cleanup()
	}
}
