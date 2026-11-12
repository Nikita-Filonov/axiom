package teststats

import "time"

// Elapsed returns the time from the earliest start to the latest end among
// completed attempts. It includes gaps between attempts and runs without
// double-counting overlapping tests. An empty collector returns zero.
//
// Elapsed is safe for concurrent use. Like the other Stats queries, it excludes
// running attempts. Loaded partial results describe only the retained completed
// attempts; check LoadResult.Incomplete separately. This is not the duration of
// the whole go test command, which also includes compilation and startup.
func (s *Stats) Elapsed() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.attempts) == 0 {
		return 0
	}
	start := s.attempts[0].Start
	end := s.attempts[0].End
	for _, attempt := range s.attempts[1:] {
		if attempt.Start.Before(start) {
			start = attempt.Start
		}
		if attempt.End.After(end) {
			end = attempt.End
		}
	}
	return end.Sub(start)
}
