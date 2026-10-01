package teststats

// Counts tallies outcomes by status.
type Counts struct {
	Total   int
	Passed  int
	Failed  int
	Skipped int
	Flaky   int
}

// Summary counts runs by derived status and attempts by outcome. Attempts are
// never flaky, so Attempts.Flaky is zero for results recorded by [Plugin].
type Summary struct {
	Runs     Counts
	Attempts Counts
}

// Summary tallies the collector's runs and attempts.
func (s *Stats) Summary() Summary {
	var summary Summary
	for _, run := range s.Runs() {
		summary.Runs.add(run.Status)
		for _, a := range run.Attempts {
			summary.Attempts.add(a.Status)
		}
	}

	return summary
}

func (c *Counts) add(status Status) {
	c.Total++
	switch status {
	case StatusPassed:
		c.Passed++
	case StatusFailed:
		c.Failed++
	case StatusSkipped:
		c.Skipped++
	case StatusFlaky:
		c.Flaky++
	}
}
