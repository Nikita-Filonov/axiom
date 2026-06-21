package testtracing

import (
	"sync"

	"github.com/Nikita-Filonov/axiom"
)

type Trace struct {
	mu      sync.Mutex
	records []TraceRecord
}

type TraceRecord struct {
	Case   axiom.Case
	Meta   axiom.Meta
	Events []axiom.Event
}

func NewTrace() *Trace {
	return &Trace{}
}

func (t *Trace) startRecord(cfg *axiom.Config) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	record := TraceRecord{}
	if cfg != nil {
		record.Meta = cfg.Meta.Copy()
		if cfg.Case != nil {
			record.Case = cfg.Case.Copy()
		}
	}

	t.records = append(t.records, record)
	return len(t.records) - 1
}

func (t *Trace) AppendToRecord(index int, event axiom.Event) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.records[index].Events = append(t.records[index].Events, event)
}

func (t *Trace) Snapshot() []TraceRecord {
	t.mu.Lock()
	defer t.mu.Unlock()

	records := make([]TraceRecord, len(t.records))
	for i, record := range t.records {
		records[i] = TraceRecord{
			Case:   record.Case.Copy(),
			Meta:   record.Meta.Copy(),
			Events: append([]axiom.Event{}, record.Events...),
		}
	}
	return records
}
