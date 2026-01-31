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

type Resources struct {
	mu *sync.Mutex

	Registry map[string]Resource
	Cache    map[string]ResourceResult
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

func (r *Resources) Join(other Resources) Resources {
	result := Resources{
		mu:       &sync.Mutex{},
		Registry: map[string]Resource{},
		Cache:    map[string]ResourceResult{},
	}

	for k, v := range r.Registry {
		result.Registry[k] = v
	}
	for k, v := range other.Registry {
		result.Registry[k] = v
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
	runner.Resources.mu.Unlock()
	if !ok {
		return zero, fmt.Errorf("resource %q not found", name)
	}

	val, cleanup, err := resource(runner)
	if err != nil {
		return zero, fmt.Errorf("resource %q failed: %w", name, err)
	}

	runner.Resources.mu.Lock()
	if existing, ok := runner.Resources.Cache[name]; ok {
		runner.Resources.mu.Unlock()
		out, ok := existing.Value.(T)
		if !ok {
			return zero, fmt.Errorf("resource %q has unexpected type", name)
		}
		return out, nil
	}

	runner.Resources.Cache[name] = ResourceResult{Value: val, Cleanup: cleanup}
	runner.Resources.mu.Unlock()

	if cleanup != nil {
		runner.Hooks.AfterAll = append(runner.Hooks.AfterAll, func(_ *Runner) { cleanup() })
	}

	out, ok := val.(T)
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
