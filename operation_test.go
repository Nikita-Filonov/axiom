package axiom_test

import (
	"fmt"
	"runtime"
	"sync"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func operationConfig(observe func(axiom.Operation, func())) *axiom.Config {
	cfg := &axiom.Config{}
	plugin := axiom.Plugin(func(c *axiom.Config) {
		c.Runtime.EmitStepWrap(func(o axiom.Operation, next axiom.StepAction) axiom.StepAction {
			return func() { observe(o, next) }
		})
		c.Runtime.EmitSetupWrap(func(o axiom.Operation, next axiom.SetupAction) axiom.SetupAction {
			return func() { observe(o, next) }
		})
		c.Runtime.EmitTeardownWrap(func(o axiom.Operation, next axiom.TeardownAction) axiom.TeardownAction {
			return func() { observe(o, next) }
		})
	})
	plugin(cfg)
	return cfg
}

func TestOperationValues(t *testing.T) {
	key := axiom.NewOperationKey[string]("example.role")
	assert.Equal(t, "example.role", key.Name())
	var operations []axiom.Operation
	cfg := operationConfig(func(o axiom.Operation, next func()) {
		operations = append(operations, o)
		next()
	})
	sameKey := axiom.NewOperationKey[string](key.Name())
	otherType := axiom.NewOperationKey[int](key.Name())
	nilKey := axiom.NewOperationKey[any]("example.nil")
	pointerKey := axiom.NewOperationKey[*int]("example.pointer")
	shared := key.Value("check")
	for _, run := range []func(string, func(), ...axiom.OperationOption){cfg.Step, cfg.Setup, cfg.Teardown} {
		run("human text", func() {}, key.Value("group"), shared, otherType.Value(42), nilKey.Value(nil), pointerKey.Value(nil))
		run("plain", func() {})
	}
	require.Len(t, operations, 6)
	for i := 0; i < len(operations); i += 2 {
		o := operations[i]
		assert.Equal(t, "human text", o.Name())
		assert.Equal(t, "check", sameKey.Get(o))
		assert.Equal(t, 42, otherType.Get(o))
		value, ok := nilKey.TryGet(o)
		assert.True(t, ok)
		assert.Nil(t, value)
		pointer, ok := pointerKey.TryGet(o)
		assert.True(t, ok)
		assert.Nil(t, pointer)
		missing, ok := key.TryGet(operations[i+1])
		assert.False(t, ok)
		assert.Empty(t, missing)
	}
	assert.Empty(t, (axiom.Operation{}).Name())
	assert.PanicsWithValue(t, `operation: missing value for key "example.role"`, func() { key.Get(axiom.Operation{}) })
	assert.PanicsWithValue(t, "operation: key name must not be empty", func() { axiom.NewOperationKey[int]("") })
	var invalid axiom.OperationKey[int]
	assert.PanicsWithValue(t, "operation: key must be created with NewOperationKey", func() { invalid.Value(1) })
	assert.PanicsWithValue(t, "operation: key must be created with NewOperationKey", func() { invalid.TryGet(axiom.Operation{}) })
}

func TestGetAndMustOperationValue(t *testing.T) {
	key := axiom.NewOperationKey[any]("example.value")
	for _, value := range []any{"check", nil, (*int)(nil)} {
		cfg := operationConfig(func(o axiom.Operation, next func()) {
			got, ok := axiom.GetOperationValue(o, key)
			assert.True(t, ok)
			assert.Equal(t, value, got)
			assert.Equal(t, value, axiom.MustOperationValue(o, key))
			next()
		})
		cfg.Step("check", func() {}, axiom.WithOperationValue(key, value))
	}
	value, ok := axiom.GetOperationValue(axiom.Operation{}, key)
	assert.False(t, ok)
	assert.Nil(t, value)
	assert.PanicsWithValue(t, `operation: missing value for key "example.value"`, func() {
		axiom.MustOperationValue(axiom.Operation{}, key)
	})
	var invalid axiom.OperationKey[int]
	assert.PanicsWithValue(t, "operation: key must be created with NewOperationKey", func() {
		axiom.GetOperationValue(axiom.Operation{}, invalid)
	})
	assert.PanicsWithValue(t, "operation: key must be created with NewOperationKey", func() {
		axiom.MustOperationValue(axiom.Operation{}, invalid)
	})
}

func TestOperationZeroValuesArePresent(t *testing.T) {
	t.Run("int", func(t *testing.T) { checkOperationValue(t, 0) })
	t.Run("bool", func(t *testing.T) { checkOperationValue(t, false) })
	t.Run("string", func(t *testing.T) { checkOperationValue(t, "") })
}

func TestOperationNonComparableValues(t *testing.T) {
	t.Run("map", func(t *testing.T) { checkOperationValue(t, map[string]int{"count": 3}) })
	t.Run("slice", func(t *testing.T) { checkOperationValue(t, []string{"first", "second"}) })
	t.Run("struct", func(t *testing.T) {
		checkOperationValue(t, struct{ Items []int }{Items: []int{1, 2}})
	})
	t.Run("interface", func(t *testing.T) { checkOperationValue[any](t, map[string]int{"count": 3}) })
}

func checkOperationValue[T any](t *testing.T, value T) {
	t.Helper()
	key := axiom.NewOperationKey[T]("example.value")
	var observed int
	cfg := operationConfig(func(o axiom.Operation, next func()) {
		observed++
		got, ok := axiom.GetOperationValue(o, key)
		assert.True(t, ok)
		assert.Equal(t, value, got)
		assert.Equal(t, value, axiom.MustOperationValue(o, key))
		next()
	})
	for _, run := range []func(string, func(), ...axiom.OperationOption){cfg.Step, cfg.Setup, cfg.Teardown} {
		run("operation", func() {}, key.Value(value))
	}
	assert.Equal(t, 3, observed)
}

func TestOperationRetainedValuesAreIsolated(t *testing.T) {
	key := axiom.NewOperationKey[string]("example.role")
	var retained []axiom.Operation
	cfg := operationConfig(func(o axiom.Operation, next func()) {
		retained = append(retained, o)
		next()
	})
	shared := key.Value("default")
	for _, run := range []func(string, func(), ...axiom.OperationOption){cfg.Step, cfg.Setup, cfg.Teardown} {
		run("same name", func() {}, shared, key.Value("first"))
		run("same name", func() {}, shared, key.Value("second"))
		run("same name", func() {}, shared)
	}
	require.Len(t, retained, 9)
	for i, o := range retained {
		assert.Equal(t, "same name", o.Name())
		assert.Equal(t, []string{"first", "second", "default"}[i%3], key.Get(o))
	}
}

func TestOperationMissingKeyInPopulatedOperation(t *testing.T) {
	key := axiom.NewOperationKey[string]("example.value")
	otherName := axiom.NewOperationKey[string]("example.other")
	otherType := axiom.NewOperationKey[int](key.Name())
	cfg := operationConfig(func(o axiom.Operation, next func()) {
		value, ok := otherName.TryGet(o)
		assert.False(t, ok)
		assert.Empty(t, value)
		number, ok := otherType.TryGet(o)
		assert.False(t, ok)
		assert.Zero(t, number)
		assert.PanicsWithValue(t, `operation: missing value for key "example.other"`, func() { otherName.Get(o) })
		assert.PanicsWithValue(t, `operation: missing value for key "example.value"`, func() { otherType.Get(o) })
		assert.Equal(t, "present", key.Get(o))
		next()
	})
	for _, run := range []func(string, func(), ...axiom.OperationOption){cfg.Step, cfg.Setup, cfg.Teardown} {
		run("operation", func() {}, key.Value("present"))
	}
}

func TestOperationValuesAcrossWrappers(t *testing.T) {
	key := axiom.NewOperationKey[string]("example.role")
	for _, kind := range []string{"step", "setup", "teardown"} {
		t.Run(kind, func(t *testing.T) {
			cfg := &axiom.Config{}
			var order []string
			for _, label := range []string{"outer", "inner"} {
				wrap := func(o axiom.Operation, next func()) func() {
					return func() {
						assert.Equal(t, "readable", o.Name())
						assert.Equal(t, "check", key.Get(o))
						order = append(order, label+".enter")
						next()
						assert.Equal(t, "check", key.Get(o))
						order = append(order, label+".exit")
					}
				}
				cfg.Runtime.EmitStepWrap(func(o axiom.Operation, next axiom.StepAction) axiom.StepAction { return wrap(o, next) })
				cfg.Runtime.EmitSetupWrap(func(o axiom.Operation, next axiom.SetupAction) axiom.SetupAction { return wrap(o, next) })
				cfg.Runtime.EmitTeardownWrap(func(o axiom.Operation, next axiom.TeardownAction) axiom.TeardownAction { return wrap(o, next) })
			}
			run := map[string]func(string, func(), ...axiom.OperationOption){
				"step": cfg.Step, "setup": cfg.Setup, "teardown": cfg.Teardown,
			}[kind]
			run("readable", func() { order = append(order, "body") }, key.Value("check"))
			assert.Equal(t, []string{"outer.enter", "inner.enter", "body", "inner.exit", "outer.exit"}, order)
		})
	}
}

func TestOperationNestedValuesAreNotInherited(t *testing.T) {
	key := axiom.NewOperationKey[string]("example.role")
	var seen []string
	cfg := operationConfig(func(o axiom.Operation, next func()) {
		before, present := key.TryGet(o)
		seen = append(seen, o.Name()+":"+before)
		next()
		after, stillPresent := key.TryGet(o)
		assert.Equal(t, before, after)
		assert.Equal(t, present, stillPresent)
	})
	cfg.Step("group", func() {
		cfg.Setup("setup", func() {
			cfg.Step("check", func() {}, key.Value("check"))
		})
		cfg.Teardown("cleanup", func() {})
	}, key.Value("group"))
	assert.Equal(t, []string{"group:group", "setup:", "check:check", "cleanup:"}, seen)
}

func TestOperationNilOptionPanicsBeforeLifecycle(t *testing.T) {
	cfg := operationConfig(func(_ axiom.Operation, _ func()) { t.Error("wrapper must not run") })
	cfg.Runtime.EmitEventSink(func(_ axiom.Event) { t.Error("event must not be emitted") })
	for _, run := range []func(string, func(), ...axiom.OperationOption){cfg.Step, cfg.Setup, cfg.Teardown} {
		assert.Panics(t, func() {
			run("invalid", func() { t.Error("body must not run") }, nil)
		})
	}
}

func TestOperationConcurrentInvocations(t *testing.T) {
	key := axiom.NewOperationKey[string]("example.id")
	sharedKey := axiom.NewOperationKey[string]("example.source")
	shared := sharedKey.Value("library")
	const count = 24
	var entered, finished sync.WaitGroup
	entered.Add(count)
	finished.Add(count)
	cfg := operationConfig(func(o axiom.Operation, next func()) {
		entered.Done()
		entered.Wait()
		assert.Equal(t, o.Name(), key.Get(o))
		assert.Equal(t, "library", sharedKey.Get(o))
		next()
		assert.Equal(t, o.Name(), key.Get(o))
	})
	runs := []func(string, func(), ...axiom.OperationOption){cfg.Step, cfg.Setup, cfg.Teardown}
	for i := range count {
		go func() {
			defer finished.Done()
			name := fmt.Sprint(i)
			runs[i%len(runs)](name, func() {}, key.Value(name), shared)
		}()
	}
	finished.Wait()
}

func TestOperationLifecycle(t *testing.T) {
	key := axiom.NewOperationKey[string]("example.role")
	for _, kind := range []string{"step", "setup", "teardown"} {
		for _, outcome := range []string{"return", "panic", "goexit"} {
			t.Run(kind+"/"+outcome, func(t *testing.T) {
				var order []string
				cfg := operationConfig(func(o axiom.Operation, next func()) {
					assert.Equal(t, "check", key.Get(o))
					order = append(order, "wrap.enter")
					defer func() {
						assert.Equal(t, "check", key.Get(o))
						order = append(order, "wrap.exit")
					}()
					next()
				})
				cfg.Runtime.EmitEventSink(func(e axiom.Event) {
					assert.Equal(t, "readable", e.Name)
					order = append(order, string(e.Type))
				})
				cfg.Hooks = axiom.NewHooks(
					axiom.WithBeforeStep(func(_ *axiom.Config, _ string) { order = append(order, "before") }),
					axiom.WithAfterStep(func(_ *axiom.Config, _ string) { order = append(order, "after") }),
				)
				run := map[string]func(string, func(), ...axiom.OperationOption){
					"step": cfg.Step, "setup": cfg.Setup, "teardown": cfg.Teardown,
				}[kind]
				done := make(chan struct{})
				go func() {
					defer close(done)
					run("readable", func() {
						order = append(order, "body")
						switch outcome {
						case "panic":
							panic("failure")
						case "goexit":
							runtime.Goexit()
						}
					}, key.Value("check"))
				}()
				<-done
				want := []string{kind + ".start"}
				if kind == "step" {
					want = append(want, "before")
				}
				want = append(want, "wrap.enter", "body", "wrap.exit")
				if outcome == "panic" {
					want = append(want, kind+".panic")
				}
				if kind == "step" {
					want = append(want, "after")
				}
				want = append(want, kind+".finish")
				assert.Equal(t, want, order)
			})
		}
	}
}

func ExampleOperationKey() {
	role := axiom.NewOperationKey[string]("example/assertions.role")
	cfg := &axiom.Config{}
	plugin := axiom.Plugin(func(c *axiom.Config) {
		c.Runtime.EmitStepWrap(func(step axiom.Operation, next axiom.StepAction) axiom.StepAction {
			return func() {
				if value, ok := role.TryGet(step); ok {
					fmt.Println(step.Name(), value)
				}
				next()
			}
		})
	})
	plugin(cfg)
	cfg.Step("Validate response", func() {}, role.Value("check"))
	// Output: Validate response check
}
