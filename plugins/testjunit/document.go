package testjunit

import "strconv"

// reportSnapshot holds detached attempt values for XML generation.
type reportSnapshot []attempt

func (s reportSnapshot) document() xmlSuites {
	groups := make(map[string]reportSnapshot)
	var names []string
	for _, a := range s {
		name := a.SuiteName
		if name == "" {
			name = defaultSuiteName
		}
		name = cleanXML(name)
		if _, exists := groups[name]; !exists {
			names = append(names, name)
		}
		groups[name] = append(groups[name], a)
	}

	doc := xmlSuites{Tests: len(s)}
	var duration float64
	for _, name := range names {
		suite, elapsed := groups[name].suite(name)
		doc.Failures += suite.Failures
		doc.Skipped += suite.Skipped
		duration += elapsed
		doc.Suites = append(doc.Suites, suite)
	}
	doc.Time = strconv.FormatFloat(duration, 'f', 6, 64)
	return doc
}
