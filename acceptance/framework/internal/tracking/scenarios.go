package tracking

import (
	"context"
	"sync"

	"github.com/cucumber/godog"
	"github.com/redpanda-data/redpanda-operator/acceptance/framework/internal/lifecycle"
	internaltesting "github.com/redpanda-data/redpanda-operator/acceptance/framework/internal/testing"
)

type scenario struct {
	*lifecycle.Cleaner

	t      *internaltesting.TestingT
	onFail func()
}

type scenarioHookTracker struct {
	registry *internaltesting.TagRegistry
	opts     *internaltesting.TestingOptions

	scenarios map[string]*scenario
	mutex     sync.Mutex
}

func newScenarioHookTracker(registry *internaltesting.TagRegistry, opts *internaltesting.TestingOptions) *scenarioHookTracker {
	return &scenarioHookTracker{
		registry:  registry,
		opts:      opts,
		scenarios: make(map[string]*scenario),
	}
}

func (s *scenarioHookTracker) start(ctx context.Context, sc *godog.Scenario, feature *feature, onFailure func()) (context.Context, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tags := tagsForScenario(feature.tags, sc.Tags)
	// make a copy of the current feature options
	opts := feature.options()

	cleaner := lifecycle.NewCleaner(godog.T(ctx), opts)
	ctx = internaltesting.TestingContext(ctx, opts)

	var err error
	for _, fn := range s.registry.Handlers(tags.flatten()) {
		var cleanup func(context.Context) error
		cleanup, err = fn.Handler(ctx, fn.Suffix)

		if cleanup != nil {
			cleaner.Cleanup(cleanup)
		}
	}

	s.scenarios[sc.Id] = &scenario{
		Cleaner: cleaner,
		onFail:  onFailure,
		// hold onto a reference of the TestingT we hand out
		// so that we can call cleanup on it
		t: internaltesting.T(ctx),
	}

	return ctx, err
}

func (s *scenarioHookTracker) finish(ctx context.Context, scenario *godog.Scenario) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	scene := s.scenarios[scenario.Id]
	if scene == nil {
		return
	}

	failure := scene.t.IsFailure()

	// clean up everything called in the scenario steps first
	scene.t.DoCleanup(ctx)
	// and then clean up the scenario hooks themselves
	scene.DoCleanup(ctx, failure)
	if failure {
		scene.onFail()
	}

	delete(s.scenarios, scenario.Id)
}
