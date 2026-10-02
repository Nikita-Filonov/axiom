package testleaks

import "fmt"

func formatReport(goroutines []goroutine, resources []trackedResource) string {
	var report []byte
	if len(goroutines) > 0 {
		report = fmt.Appendf(report, "%d goroutine(s) still running", goroutineCount(goroutines))
		for i, g := range goroutines {
			if i >= 8 {
				report = fmt.Appendf(report, "\n... %d more stack group(s)", len(goroutines)-i)
				break
			}
			report = fmt.Appendf(report, "\n\n%d goroutine(s):\n%s", g.count, g.stack)
		}
	}
	if len(resources) > 0 {
		if len(report) > 0 {
			report = append(report, '\n')
		}
		report = fmt.Appendf(report, "%d tracked resource(s) not released", len(resources))
		for _, resource := range resources {
			report = fmt.Appendf(report, "\n- %s (registered at %s)", resource.name, resource.site)
		}
	}
	return string(report)
}
