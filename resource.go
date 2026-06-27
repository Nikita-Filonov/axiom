package axiom

import (
	"fmt"
	"sync"
)

type Resource func(r *Runner) (any, func(), error)

type ResourceResult struct {
	Value   any
	Cleanup func()
}

type ResourceCleanup func(*Runner)

type Resources struct {
	mu    *sync.Mutex
	onces map[string]*resourceOnce

	Registry map[string]Resource
	Cache    map[string]ResourceResult
	Cleanups []ResourceCleanup
}

type resourceOnce struct {
	once    sync.Once
	value   any
	cleanup func()
	err     error
}

type ResourcesOption func(*Resources)

func NewResources(options ...ResourcesOption) Resources {
	r := Resources{}
	for _, option := range options {
		option(&r)
	}
	return r
}

func WithResource(name string, resource Resource) ResourcesOption {
	return func(r *Resources) {
		if r.Registry == nil {
			r.Registry = map[string]Resource{}
		}
		r.Registry[name] = resource
	}
}

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

func (r *Resources) Copy() Resources {
	result := Resources{mu: &sync.Mutex{}}

	if r.Registry != nil {
		result.Registry = make(map[string]Resource, len(r.Registry))
		for k, v := range r.Registry {
			result.Registry[k] = v
		}
	}
	if r.Cache != nil {
		result.Cache = make(map[string]ResourceResult, len(r.Cache))
		for k, v := range r.Cache {
			result.Cache[k] = v
		}
	}
	if r.Cleanups != nil {
		result.Cleanups = append([]ResourceCleanup{}, r.Cleanups...)
	}

	return result
}

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

	if len(other.Cache) > 0 {
		if result.Cache == nil {
			result.Cache = make(map[string]ResourceResult, len(other.Cache))
		}
		for k, v := range other.Cache {
			result.Cache[k] = v
		}
	}

	if len(other.Cleanups) > 0 {
		result.Cleanups = append(result.Cleanups, other.Cleanups...)
	}

	return result
}

func (r *Resources) Normalize() {
	if r.mu == nil {
		r.mu = &sync.Mutex{}
	}
	if r.Registry == nil {
		r.Registry = map[string]Resource{}
	}
	if r.Cache == nil {
		r.Cache = map[string]ResourceResult{}
	}
	if r.onces == nil {
		r.onces = map[string]*resourceOnce{}
	}
}

func (r *Resources) Teardown(runner *Runner) {
	for i := len(r.Cleanups) - 1; i >= 0; i-- {
		r.Cleanups[i](runner)
	}
	r.Cleanups = nil
}

func GetResource[T any](runner *Runner, name string) (T, error) {
	var zero T

	runner.Resources.Normalize()

	runner.Resources.mu.Lock()
	if res, ok := runner.Resources.Cache[name]; ok {
		runner.Resources.mu.Unlock()
		out, ok := res.Value.(T)
		if !ok {
			return zero, fmt.Errorf("resource %q has unexpected type", name)
		}
		return out, nil
	}

	resource, ok := runner.Resources.Registry[name]
	if !ok {
		runner.Resources.mu.Unlock()
		runner.Runtime.Event(NewEvent(EventTypeResourceSetupFailed, WithEventName(name), WithEventMessage("not found")))
		return zero, fmt.Errorf("resource %q not found", name)
	}

	ro, exists := runner.Resources.onces[name]
	if !exists {
		ro = &resourceOnce{}
		runner.Resources.onces[name] = ro
	}
	runner.Resources.mu.Unlock()

	ro.once.Do(func() {
		runner.Runtime.Event(NewEvent(EventTypeResourceSetupStart, WithEventName(name)))
		val, cleanup, err := resource(runner)
		if err != nil {
			ro.err = err
			runner.Runtime.Event(NewEvent(EventTypeResourceSetupFailed, WithEventName(name), WithEventMessage(err.Error())))
			return
		}

		ro.value = val
		ro.cleanup = cleanup

		runner.Resources.mu.Lock()
		runner.Resources.Cache[name] = ResourceResult{Value: val, Cleanup: cleanup}
		if cleanup != nil {
			runner.Resources.Cleanups = append(runner.Resources.Cleanups, resourceCleanupHook(name, cleanup))
		}
		runner.Resources.mu.Unlock()

		runner.Runtime.Event(NewEvent(EventTypeResourceSetupFinish, WithEventName(name)))
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

func MustResource[T any](runner *Runner, name string) T {
	v, err := GetResource[T](runner, name)
	if err != nil {
		panic(err)
	}
	return v
}

func UseResources(names ...string) func(r *Runner) {
	return func(r *Runner) {
		for _, name := range names {
			MustResource[any](r, name)
		}
	}
}

func resourceCleanupHook(name string, cleanup func()) ResourceCleanup {
	return func(r *Runner) {
		r.Runtime.Event(NewEvent(EventTypeResourceCleanupStart, WithEventName(name)))
		defer func() {
			if v := recover(); v != nil {
				r.Runtime.Event(NewEvent(EventTypeResourceCleanupPanic, WithEventName(name), WithEventMessage(v)))
				panic(v)
			}

			r.Runtime.Event(NewEvent(EventTypeResourceCleanupFinish, WithEventName(name)))
		}()

		cleanup()
	}
}
