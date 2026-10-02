package testleaks

import "time"

type leakReporter interface {
	Helper()
	Errorf(string, ...any)
}

type goroutineFinder func(string, []string) ([]goroutine, error)

func verify(t leakReporter, state *attemptState, id string, c Config, find goroutineFinder) {
	t.Helper()
	deadline := time.Now().Add(c.GracePeriod)
	var goroutines []goroutine
	var resources []trackedResource
	for {
		resources = state.snapshot()
		if c.Goroutines {
			var err error
			goroutines, err = find(id, c.IgnoreFunctions)
			if err != nil {
				t.Errorf("testleaks: goroutine profile: %v", err)
				return
			}
		}
		if len(goroutines) == 0 && len(resources) == 0 {
			return
		}
		if !time.Now().Before(deadline) {
			break
		}
		time.Sleep(min(10*time.Millisecond, time.Until(deadline)))
	}

	t.Errorf("testleaks: %s", formatReport(goroutines, resources))
}
