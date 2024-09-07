package environments

import (
	"context"
	"sync"

	acceptancetesting "github.com/redpanda-data/redpanda-operator/acceptance/testing"
)

type scenarioHookTracker struct {
	scenarios map[string]*acceptancetesting.TestingT
	mutex     sync.Mutex
}

func newScenarioHookTracker() *scenarioHookTracker {
	return &scenarioHookTracker{
		scenarios: make(map[string]*acceptancetesting.TestingT),
	}
}

func (s *scenarioHookTracker) add(ctx context.Context, id, namespace string, opts *acceptancetesting.TestingOptions) context.Context {
	opts = opts.Clone()
	opts.KubectlOptions.WithNamespace(namespace)

	s.mutex.Lock()
	defer s.mutex.Unlock()

	ctx = acceptancetesting.TestingContext(ctx, opts)
	s.scenarios[id] = acceptancetesting.T(ctx)

	return ctx
}

func (s *scenarioHookTracker) cleanup(ctx context.Context, id string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	t := s.scenarios[id]
	t.DoCleanup(ctx)
	delete(s.scenarios, id)
}
