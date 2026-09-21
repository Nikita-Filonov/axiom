package axiom_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type cacheCallResult[T any] struct {
	value T
	err   error
}

func startCacheCall[T any](ctx context.Context, cache *axiom.Cache, key axiom.CacheKey[T], create func() (T, error)) <-chan cacheCallResult[T] {
	result := make(chan cacheCallResult[T], 1)
	go func() {
		value, err := key.GetOrCreate(ctx, cache, create)
		result <- cacheCallResult[T]{value: value, err: err}
	}()
	return result
}

func TestCache_GetOrCreate_ConcurrentCallersShareValue(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cache := axiom.NewCache()
		key := axiom.NewCacheKey[*int]("value")
		value := new(int)
		*value = 42
		release := make(chan struct{})
		var calls atomic.Int32
		create := func() (*int, error) {
			calls.Add(1)
			<-release
			return value, nil
		}

		const workers = 30
		results := make([]<-chan cacheCallResult[*int], workers)
		for i := range results {
			results[i] = startCacheCall(t.Context(), cache, key, create)
		}
		synctest.Wait()
		assert.Equal(t, int32(1), calls.Load())
		_, ok := key.Get(cache)
		assert.False(t, ok, "pending must not appear as a cached zero value")
		close(release)
		for _, result := range results {
			got := <-result
			require.NoError(t, got.err)
			assert.Same(t, value, got.value)
		}
		got, ok := key.Get(cache)
		require.True(t, ok)
		assert.Same(t, value, got)
		assert.Equal(t, int32(1), calls.Load())
	})
}

func TestCache_GetOrCreate_ConcurrentErrorIsSharedWithoutAutomaticRetry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cache := axiom.NewCache()
		key := axiom.NewCacheKey[int]("value")
		release := make(chan struct{})
		failure := errors.New("failed")
		var calls atomic.Int32
		create := func() (int, error) {
			calls.Add(1)
			<-release
			return 42, failure
		}
		results := make([]<-chan cacheCallResult[int], 20)
		for i := range results {
			results[i] = startCacheCall(t.Context(), cache, key, create)
		}
		synctest.Wait()
		assert.Equal(t, int32(1), calls.Load())
		close(release)
		for _, result := range results {
			got := <-result
			assert.Zero(t, got.value)
			assert.Same(t, failure, got.err)
		}
		assert.Equal(t, int32(1), calls.Load())
		_, ok := key.Get(cache)
		assert.False(t, ok)
		value, err := key.GetOrCreate(t.Context(), cache, func() (int, error) { return 7, nil })
		require.NoError(t, err)
		assert.Equal(t, 7, value)
	})
}

func TestCache_GetOrCreate_IndependentAndNestedKeysDoNotBlock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cache := axiom.NewCache()
		blocked := axiom.NewCacheKey[int]("blocked")
		release := make(chan struct{})
		result := startCacheCall(t.Context(), cache, blocked, func() (int, error) {
			<-release
			return 1, nil
		})
		synctest.Wait()

		outer := axiom.NewCacheKey[int]("outer")
		inner := axiom.NewCacheKey[int]("inner")
		value, err := outer.GetOrCreate(t.Context(), cache, func() (int, error) {
			return inner.GetOrCreate(t.Context(), cache, func() (int, error) { return 2, nil })
		})
		require.NoError(t, err)
		assert.Equal(t, 2, value)
		close(release)
		require.NoError(t, (<-result).err)
	})
}

func TestCache_GetOrCreate_WaiterCancellationLeavesCreationAndOtherWaitersAlone(t *testing.T) {
	for _, reason := range []string{"cancel", "deadline"} {
		t.Run(reason, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				cache := axiom.NewCache()
				key := axiom.NewCacheKey[int]("value")
				release := make(chan struct{})
				var calls atomic.Int32
				create := func() (int, error) {
					calls.Add(1)
					<-release
					return 42, nil
				}
				leader := startCacheCall(t.Context(), cache, key, create)
				synctest.Wait()

				ctx, cancel := context.WithTimeout(t.Context(), time.Second)
				defer cancel()
				cancelled := startCacheCall(ctx, cache, key, create)
				other := startCacheCall(t.Context(), cache, key, create)
				synctest.Wait()
				wantErr := context.DeadlineExceeded
				if reason == "cancel" {
					cancel()
					wantErr = context.Canceled
				}
				// For the deadline case, synctest advances its clock while this waits.
				got := <-cancelled
				assert.Zero(t, got.value)
				assert.ErrorIs(t, got.err, wantErr)

				late := startCacheCall(t.Context(), cache, key, create)
				synctest.Wait()
				assert.Equal(t, int32(1), calls.Load(), "a cancelled waiter must not evict pending work")
				close(release)
				for _, result := range []<-chan cacheCallResult[int]{leader, other, late} {
					got := <-result
					require.NoError(t, got.err)
					assert.Equal(t, 42, got.value)
				}
			})
		})
	}
}

func TestCache_GetOrCreate_CreatorContextDoesNotInterruptOrDiscardSuccess(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cache := axiom.NewCache()
		key := axiom.NewCacheKey[int]("value")
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		release := make(chan struct{})
		leader := startCacheCall(ctx, cache, key, func() (int, error) {
			<-release
			return 42, nil
		})
		synctest.Wait()
		cancel()
		synctest.Wait()
		select {
		case <-leader:
			t.Fatal("GetOrCreate must not return while its synchronous constructor is still running")
		default:
		}
		close(release)
		got := <-leader
		require.NoError(t, got.err)
		assert.Equal(t, 42, got.value)
		value, ok := key.Get(cache)
		require.True(t, ok)
		assert.Equal(t, 42, value)
	})
}

func TestCache_GetOrCreate_AbortedCreatorUnblocksWaitersAndPreservesControlFlow(t *testing.T) {
	for _, termination := range []string{"panic", "panic nil", "goexit"} {
		t.Run(termination, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				cache := axiom.NewCache()
				key := axiom.NewCacheKey[int]("aborted-key")
				release := make(chan struct{})
				finished := make(chan struct{})
				panicValue := errors.New("constructor panic")
				var recovered any
				returned := false
				go func() {
					defer close(finished)
					defer func() { recovered = recover() }()
					_, _ = key.GetOrCreate(t.Context(), cache, func() (int, error) {
						<-release
						switch termination {
						case "panic":
							panic(panicValue)
						case "panic nil":
							panic(nil)
						default:
							runtime.Goexit()
						}
						return 0, nil
					})
					returned = true
				}()
				synctest.Wait()

				var calls atomic.Int32
				waiters := make([]<-chan cacheCallResult[int], 5)
				for i := range waiters {
					waiters[i] = startCacheCall(t.Context(), cache, key, func() (int, error) {
						calls.Add(1)
						return 99, nil
					})
				}
				synctest.Wait()
				close(release)
				<-finished
				assert.False(t, returned, "panic/Goexit must propagate in the creator's calling goroutine")
				switch termination {
				case "panic":
					assert.Same(t, panicValue, recovered)
				case "panic nil":
					assert.IsType(t, &runtime.PanicNilError{}, recovered)
				case "goexit":
					assert.Nil(t, recovered)
				}
				for _, waiter := range waiters {
					got := <-waiter
					assert.Zero(t, got.value)
					assert.Error(t, got.err)
					assert.ErrorContains(t, got.err, "aborted-key")
				}
				assert.Zero(t, calls.Load(), "waiters must not silently retry aborted creation")
				_, ok := key.Get(cache)
				assert.False(t, ok)
				value, err := key.GetOrCreate(t.Context(), cache, func() (int, error) { return 7, nil })
				require.NoError(t, err)
				assert.Equal(t, 7, value)
			})
		})
	}
}

func TestCache_SetDelete_DetachPendingWorkWithoutChangingItsWaiters(t *testing.T) {
	for _, operation := range []string{"set", "delete", "delete and recreate"} {
		for _, outcome := range []string{"success", "error", "panic", "goexit"} {
			t.Run(operation+"/"+outcome, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					cache := axiom.NewCache()
					key := axiom.NewCacheKey[int]("value")
					release := make(chan struct{})
					finished := make(chan struct{})
					failure := errors.New("failed")
					go func() {
						defer close(finished)
						defer func() { _ = recover() }()
						_, _ = key.GetOrCreate(t.Context(), cache, func() (int, error) {
							<-release
							switch outcome {
							case "error":
								return 11, failure
							case "panic":
								panic(failure)
							case "goexit":
								runtime.Goexit()
							}
							return 11, nil
						})
					}()
					synctest.Wait()
					waiter := startCacheCall(t.Context(), cache, key, func() (int, error) { return 99, nil })
					synctest.Wait()

					var replacement <-chan cacheCallResult[int]
					releaseReplacement := make(chan struct{})
					switch operation {
					case "set":
						key.Set(cache, 22)
						value, err := key.GetOrCreate(t.Context(), cache, func() (int, error) { return 99, nil })
						require.NoError(t, err)
						assert.Equal(t, 22, value)
					case "delete":
						key.Delete(cache)
					case "delete and recreate":
						key.Delete(cache)
						replacement = startCacheCall(t.Context(), cache, key, func() (int, error) {
							<-releaseReplacement
							return 22, nil
						})
						synctest.Wait()
					}
					close(release)
					<-finished
					got := <-waiter
					switch outcome {
					case "success":
						require.NoError(t, got.err)
						assert.Equal(t, 11, got.value)
					case "error":
						assert.Same(t, failure, got.err)
						assert.Zero(t, got.value)
					default:
						assert.Error(t, got.err)
						assert.ErrorContains(t, got.err, "value")
						assert.Zero(t, got.value)
					}

					value, ok := key.Get(cache)
					if operation == "set" {
						require.True(t, ok)
						assert.Equal(t, 22, value)
					} else {
						assert.False(t, ok, "old work must not resurrect or replace the current entry")
					}
					if replacement != nil {
						close(releaseReplacement)
						got := <-replacement
						require.NoError(t, got.err)
						assert.Equal(t, 22, got.value)
						value, ok := key.Get(cache)
						require.True(t, ok, "old failures must not remove a new pending entry")
						assert.Equal(t, 22, value)
					}
				})
			})
		}
	}
}

func TestCache_ConcurrentMixedOperations(t *testing.T) {
	var cache axiom.Cache
	keys := []axiom.CacheKey[int]{
		axiom.NewCacheKey[int]("first"),
		axiom.NewCacheKey[int]("second"),
		axiom.NewCacheKey[int]("third"),
	}
	start := make(chan struct{})
	var workers sync.WaitGroup
	for worker := range 16 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			for iteration := range 200 {
				slot := (worker + iteration) % len(keys)
				key := keys[slot]
				want := slot + 1
				switch iteration % 4 {
				case 0:
					key.Set(&cache, want)
				case 1:
					value, ok := key.Get(&cache)
					if ok {
						assert.Equal(t, want, value)
					}
				case 2:
					key.Delete(&cache)
				case 3:
					value, err := key.GetOrCreate(t.Context(), &cache, func() (int, error) { return want, nil })
					assert.NoError(t, err)
					assert.Equal(t, want, value)
				}
			}
		}()
	}
	close(start)
	workers.Wait()
}

func TestCache_ResourceAndOrdinaryFixture_PreserveSuiteLifecycle(t *testing.T) {
	var cacheBuilds, fixtureBuilds, factoryBuilds, factoryCleanups int
	var setups []string
	cacheResource := axiom.DefineResource("shared-cache", func(_ *axiom.Runner) (*axiom.Cache, func(), error) {
		cacheBuilds++
		return axiom.NewCache(), nil, nil
	})
	key := axiom.NewCacheKey[int]("readonly-entity")
	factory := axiom.DefineFixture("factory", func(cfg *axiom.Config) (func() (int, error), func(), error) {
		factoryBuilds++
		closed := false
		return func() (int, error) {
				require.False(cfg.T(), closed)
				cfg.Setup("create", func() {})
				return 42, nil
			}, func() {
				closed = true
				factoryCleanups++
			}, nil
	})
	fixture := axiom.DefineFixture("entity", func(cfg *axiom.Config) (int, func(), error) {
		fixtureBuilds++
		cache := cacheResource.Get(cfg.Runner)
		value, err := key.GetOrCreate(cfg.Context.Raw, cache, func() (int, error) {
			return factory.Get(cfg)()
		})
		return value, nil, err
	})
	runner := axiom.NewRunner(
		axiom.WithRunnerResources(cacheResource),
		axiom.WithRunnerFixtures(factory, fixture),
		axiom.WithRunnerPlugins(func(cfg *axiom.Config) {
			cfg.Runtime.EmitSetupWrap(func(name string, next axiom.SetupAction) axiom.SetupAction {
				return func() {
					setups = append(setups, cfg.Case.Name+":"+name)
					next()
				}
			})
		}),
	)

	t.Run("suite", func(t *testing.T) {
		suite := axiom.NewSuite(t, &axiom.Suite{}, axiom.WithSuiteConfigRunner(runner))
		for _, name := range []string{"first", "second"} {
			suite.Test(name, func(s *axiom.Suite) {
				s.RunCase(axiom.NewCase(axiom.WithCaseName(name)), func(cfg *axiom.Config) {
					if name == "second" {
						assert.Equal(cfg.T(), 1, factoryCleanups, "the first test's dependency is already closed")
					}
					assert.Equal(cfg.T(), 42, fixture.Get(cfg))
					assert.Equal(cfg.T(), 42, fixture.Get(cfg))
				})
			})
		}
		suite.Run()
	})

	assert.Equal(t, 1, cacheBuilds)
	assert.Equal(t, 2, fixtureBuilds, "each case still owns its ordinary fixture lifecycle")
	assert.Equal(t, 1, factoryBuilds, "cache hits must not resolve the test-bound dependency")
	assert.Equal(t, 1, factoryCleanups)
	assert.Equal(t, []string{"first:create"}, setups, "only the creator reports actual setup")
}

func TestCache_ScopeFollowsExistingResourceJoinSemantics(t *testing.T) {
	resource := axiom.DefineResource("cache", func(_ *axiom.Runner) (*axiom.Cache, func(), error) {
		return axiom.NewCache(), nil, nil
	})
	base := axiom.NewRunner(axiom.WithRunnerResources(resource))
	first := base.Join(axiom.NewRunner())
	second := base.Join(axiom.NewRunner())
	firstCache := resource.Get(first)
	secondCache := resource.Get(second)
	assert.NotSame(t, firstCache, secondCache, "cold definitions create independent caches")

	baseCache := resource.Get(base)
	warmed := base.Join(axiom.NewRunner())
	assert.Same(t, baseCache, resource.Get(warmed), "warmed joins inherit the existing pointer")
	assert.NotSame(t, baseCache, firstCache)
	assert.NotSame(t, baseCache, secondCache)
}

func TestCache_ScopeFollowsExplicitContextSharingAndCaseOverrides(t *testing.T) {
	contextKey := axiom.NewContextKey[*axiom.Cache]("shared-cache")
	key := axiom.NewCacheKey[int]("value")
	shared := axiom.NewCache()
	base := axiom.NewRunner(axiom.WithRunnerContext(contextKey.Value(shared)))
	joined := base.Join(axiom.NewRunner())
	testCase := axiom.NewCase()
	first := base.BuildConfig(t, &testCase)
	second := joined.BuildConfig(t, &testCase)
	key.Set(contextKey.Get(&first.Context), 42)
	value, ok := key.Get(contextKey.Get(&second.Context))
	require.True(t, ok)
	assert.Equal(t, 42, value)
	assert.Same(t, shared, contextKey.Get(&second.Context))

	caseCache := axiom.NewCache()
	override := axiom.NewCase(axiom.WithCaseContext(contextKey.Value(caseCache)))
	for range 2 {
		cfg := joined.BuildConfig(t, &override)
		assert.Same(t, caseCache, contextKey.Get(&cfg.Context), "case copies do not reset pointed-to caches")
		_, ok := key.Get(contextKey.Get(&cfg.Context))
		assert.False(t, ok)
	}
}

const cacheRetryHelperEnv = "AXIOM_CACHE_RETRY_HELPER"

func TestCache_ResourceAndLocal_Retry(t *testing.T) {
	if mode := os.Getenv(cacheRetryHelperEnv); mode != "" {
		runCacheRetryHelper(t, mode)
		return
	}

	for _, mode := range []string{"scenario-error", "constructor-error", "dependency-fatal"} {
		t.Run(mode, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestCache_ResourceAndLocal_Retry$", "-test.v", "-test.timeout=15s")
			cmd.Env = append(os.Environ(), cacheRetryHelperEnv+"="+mode)
			output, err := cmd.CombinedOutput()
			require.Error(t, err, "the first deliberately failed attempt keeps the helper process failed")
			creates := 2
			if mode == "scenario-error" {
				creates = 1
			}
			assert.Contains(t, string(output), fmt.Sprintf(
				"cache-retry mode=%s attempts=2 cache-builds=1 creates=%d local-caches=2 successful-attempts=1",
				mode, creates,
			))
		})
	}
}

func runCacheRetryHelper(t *testing.T, mode string) {
	var attempts, cacheBuilds, creates, localCaches, successes int
	resource := axiom.DefineResource("cache", func(_ *axiom.Runner) (*axiom.Cache, func(), error) {
		cacheBuilds++
		return axiom.NewCache(), nil, nil
	})
	key := axiom.NewCacheKey[int]("value")
	localKey := axiom.NewLocalKey[*axiom.Cache]("attempt-cache")
	var previousConfig *axiom.Config
	var previousLocal *axiom.Cache
	fixture := axiom.DefineFixture("entity", func(cfg *axiom.Config) (int, func(), error) {
		value, err := key.GetOrCreate(cfg.Context.Raw, resource.Get(cfg.Runner), func() (int, error) {
			creates++
			if attempts == 1 {
				switch mode {
				case "constructor-error":
					return 0, errors.New("deliberate constructor error")
				case "dependency-fatal":
					cfg.T().Fatal("deliberate dependency failure")
				}
			}
			return 42, nil
		})
		return value, nil, err
	})
	runner := axiom.NewRunner(
		axiom.WithRunnerResources(resource),
		axiom.WithRunnerFixtures(fixture),
		axiom.WithRunnerRetry(axiom.WithRetryTimes(2)),
		axiom.WithRunnerHooks(axiom.WithBeforeTest(func(cfg *axiom.Config) {
			attempts++
			_, exists := axiom.GetLocal(cfg, localKey)
			require.False(cfg.T(), exists)
			require.NotSame(cfg.T(), previousConfig, cfg)
			local := axiom.NewCache()
			axiom.SetLocal(cfg, localKey, local)
			require.NotSame(cfg.T(), previousLocal, local)
			previousConfig, previousLocal = cfg, local
			localCaches++
		})),
	)
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("retry")), func(cfg *axiom.Config) {
		require.Equal(cfg.T(), 42, fixture.Get(cfg))
		if attempts == 1 && mode == "scenario-error" {
			cfg.T().Error("deliberate scenario failure")
			return
		}
		successes++
	})
	t.Logf("cache-retry mode=%s attempts=%d cache-builds=%d creates=%d local-caches=%d successful-attempts=%d",
		mode, attempts, cacheBuilds, creates, localCaches, successes)
}

func TestCache_SkipNowDoesNotLeavePendingEntry(t *testing.T) {
	cache := axiom.NewCache()
	key := axiom.NewCacheKey[int]("value")
	t.Run("skipped creator", func(t *testing.T) {
		_, _ = key.GetOrCreate(t.Context(), cache, func() (int, error) {
			t.SkipNow()
			return 0, nil
		})
		t.Fatal("SkipNow must exit the calling test")
	})
	_, ok := key.Get(cache)
	assert.False(t, ok)
	value, err := key.GetOrCreate(t.Context(), cache, func() (int, error) { return 42, nil })
	require.NoError(t, err)
	assert.Equal(t, 42, value)
}

func TestCache_BasicUsage(t *testing.T) {
	cache := axiom.NewCache()
	key := axiom.NewCacheKey[string]("catalog.version")

	key.Set(cache, "v1")
	value, ok := key.Get(cache)
	require.True(t, ok)
	assert.Equal(t, "v1", value)

	key.Delete(cache)
	_, ok = key.Get(cache)
	assert.False(t, ok)
}

func TestCache_GetOrCreate_ReusesSuccessfulValue(t *testing.T) {
	cache := axiom.NewCache()
	key := axiom.NewCacheKey[string]("catalog.version")
	calls := 0

	for range 2 {
		value, err := key.GetOrCreate(t.Context(), cache, func() (string, error) {
			calls++
			return "v1", nil
		})
		require.NoError(t, err)
		assert.Equal(t, "v1", value)
	}
	assert.Equal(t, 1, calls)
}

func TestCache_ResourceIsAccessibleOutsideCase(t *testing.T) {
	sharedCache := axiom.DefineResource("shared-data-cache",
		func(_ *axiom.Runner) (*axiom.Cache, func(), error) {
			return axiom.NewCache(), nil, nil
		},
	)
	runner := axiom.NewRunner(axiom.WithRunnerResources(sharedCache))
	key := axiom.NewCacheKey[int]("catalog.shared")

	value, err := key.GetOrCreate(t.Context(), sharedCache.Get(runner), func() (int, error) {
		return 42, nil
	})
	require.NoError(t, err)
	assert.Equal(t, 42, value)
	assert.Same(t, sharedCache.Get(runner), sharedCache.Get(runner))
}

func TestCacheExample(t *testing.T) {
	type Settings struct {
		Environment string
	}

	// This instance defines the sharing scope.
	cache := axiom.NewCache()
	settingsKey := axiom.NewCacheKey[Settings]("service.settings")

	// The constructor is called only on the first cache miss.
	loads := 0
	loadSettings := func() (Settings, error) {
		loads++
		return Settings{Environment: "staging"}, nil
	}

	first, err := settingsKey.GetOrCreate(t.Context(), cache, loadSettings)
	require.NoError(t, err)
	second, err := settingsKey.GetOrCreate(t.Context(), cache, loadSettings)
	require.NoError(t, err)

	// Both calls observe the value produced by the first call.
	require.Equal(t, first, second)
	require.Equal(t, 1, loads)
}
