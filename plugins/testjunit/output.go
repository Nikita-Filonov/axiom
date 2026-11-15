package testjunit

import "fmt"

type outputWriter struct {
	path    string
	onError func(error)
}

func newOutputWriter(settings Config) *outputWriter {
	if settings.OutputFile == "" {
		return nil
	}
	return &outputWriter{
		path:    settings.OutputFile,
		onError: settings.ErrorHandler,
	}
}

func (r *Reporter) addOutput(output *outputWriter) {
	if output == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.outputs {
		if existing.path == output.path {
			return
		}
	}
	r.outputs = append(r.outputs, output)
}

func (r *Reporter) publish() {
	r.mu.Lock()
	outputs := append([]*outputWriter(nil), r.outputs...)
	r.mu.Unlock()

	for _, output := range outputs {
		output.publish(r)
	}
}

func (w *outputWriter) publish(reporter *Reporter) {
	if err := reporter.WriteFile(w.path); err != nil {
		w.onError(fmt.Errorf("testjunit: export %q: %w", w.path, err))
	}
}
