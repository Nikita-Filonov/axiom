package testjunit

import (
	"strconv"
	"time"
)

func (s reportSnapshot) suite(name string) (xmlSuite, float64) {
	suite := xmlSuite{Name: name, Tests: len(s)}
	// Sum seconds instead of time.Duration to avoid overflowing large suites.
	var duration float64
	for _, a := range s {
		d := max(a.Duration, 0)
		duration += d.Seconds()
		result := xmlCase{
			ClassName: suite.Name,
			Name:      cleanXML(a.TestName),
			Time:      seconds(d),
		}
		switch a.Status {
		case statusPassed:
			// Successful cases have neither a failure nor a skipped element.
		case statusFailed:
			message := a.Error
			if message == "" {
				message = "test failed"
			}
			message = cleanXML(message)
			result.Failure = &xmlOutcome{Message: message, Text: message}
			suite.Failures++
		case statusSkipped:
			message := a.SkipReason
			if message == "" {
				message = "skipped"
			}
			message = cleanXML(message)
			result.Skipped = &xmlOutcome{Message: message, Text: message}
			suite.Skipped++
		}
		suite.Cases = append(suite.Cases, result)
	}
	suite.Time = strconv.FormatFloat(duration, 'f', 6, 64)
	return suite, duration
}

func seconds(d time.Duration) string {
	return strconv.FormatFloat(d.Seconds(), 'f', 6, 64)
}
