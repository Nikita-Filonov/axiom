package testleaks

import (
	"sort"
	"sync"
)

type trackedResource struct {
	id   uint64
	name string
	site string
}

type attemptState struct {
	mu   sync.Mutex
	next uint64
	open map[uint64]trackedResource
}

func newAttemptState() *attemptState {
	return &attemptState{open: make(map[uint64]trackedResource)}
}

func (s *attemptState) add(name, site string) uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	s.open[s.next] = trackedResource{id: s.next, name: name, site: site}
	return s.next
}

func (s *attemptState) release(id uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.open, id)
}

func (s *attemptState) snapshot() []trackedResource {
	s.mu.Lock()
	defer s.mu.Unlock()
	resources := make([]trackedResource, 0, len(s.open))
	for _, resource := range s.open {
		resources = append(resources, resource)
	}
	sort.Slice(resources, func(i, j int) bool { return resources[i].id < resources[j].id })
	return resources
}
