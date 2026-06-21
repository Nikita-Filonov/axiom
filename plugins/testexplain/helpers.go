package testexplain

import (
	"fmt"
	"reflect"
	goruntime "runtime"
	"sort"

	"github.com/Nikita-Filonov/axiom"
)

func explainSkip(s axiom.Skip) SkipExplanation {
	return SkipExplanation{
		Enabled: s.Enabled,
		Reason:  s.Reason,
	}
}

func explainRetry(r axiom.Retry) RetryExplanation {
	return RetryExplanation{
		Times:            r.Times,
		Delay:            r.Delay.String(),
		DelayNanoseconds: int64(r.Delay),
	}
}

func explainParallel(p axiom.Parallel) ParallelExplanation {
	return ParallelExplanation{Enabled: p.Enabled}
}

func explainContext(c axiom.Context) ContextExplanation {
	return ContextExplanation{
		Raw:      c.Raw != nil,
		DB:       c.DB != nil,
		MQ:       c.MQ != nil,
		RPC:      c.RPC != nil,
		DataKeys: sortedMapKeys(c.Data),
	}
}

func explainHooks(h axiom.Hooks) HooksExplanation {
	return HooksExplanation{
		BeforeAll:  explainCallables(h.BeforeAll),
		AfterAll:   explainCallables(h.AfterAll),
		BeforeTest: explainCallables(h.BeforeTest),
		AfterTest:  explainCallables(h.AfterTest),
		BeforeStep: explainCallables(h.BeforeStep),
		AfterStep:  explainCallables(h.AfterStep),
	}
}

func explainRuntime(r axiom.Runtime) RuntimeExplanation {
	return RuntimeExplanation{
		TestWraps:     explainCallables(r.TestWraps),
		StepWraps:     explainCallables(r.StepWraps),
		SetupWraps:    explainCallables(r.SetupWraps),
		TeardownWraps: explainCallables(r.TeardownWraps),
		LogSinks:      explainCallables(r.LogSinks),
		AssertSinks:   explainCallables(r.AssertSinks),
		ArtefactSinks: explainCallables(r.ArtefactSinks),
		EventSinks:    explainCallables(r.EventSinks),
	}
}

func explainCallables[T any](items []T) CallableExplanation {
	names := make([]string, 0, len(items))
	for _, item := range items {
		if name := callableName(item); name != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	return CallableExplanation{
		Count: len(items),
		Names: names,
	}
}

func callableName(fn any) string {
	v := reflect.ValueOf(fn)
	if !v.IsValid() || v.Kind() != reflect.Func || v.IsNil() {
		return ""
	}

	pc := v.Pointer()
	if pc == 0 {
		return ""
	}

	f := goruntime.FuncForPC(pc)
	if f == nil {
		return ""
	}

	return f.Name()
}

func paramsType(params any) string {
	if params == nil {
		return ""
	}

	return fmt.Sprintf("%T", params)
}

func sortedMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
