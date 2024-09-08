package tracking

import (
	"context"
	"io"
	"sync"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/formatters"
	messages "github.com/cucumber/messages/go/v21"
	"github.com/redpanda-data/redpanda-operator/acceptance/framework/internal/lifecycle"
	internaltesting "github.com/redpanda-data/redpanda-operator/acceptance/framework/internal/testing"
)

type feature struct {
	*lifecycle.Cleaner

	opts           *internaltesting.TestingOptions
	scenariosToRun int
	isRunning      bool
	hasStepFailure bool
	tags           *tagset
}

func (f *feature) options() *internaltesting.TestingOptions {
	return f.opts.Clone()
}

// FeatureHookTracker exists since godog doesn't have any before/after
// feature hooks, it acts by reference counting the number
// of times a before hook has been called and then decrementing
// a counter based on the number of scenarios that should
// be run per-feature based on the nasty hack of hooking
// into the formatter interface of godog.
type FeatureHookTracker struct {
	scenarios *scenarioHookTracker

	failedSuite bool
	registry    *internaltesting.TagRegistry
	opts        *internaltesting.TestingOptions

	features map[string]*feature
	mutex    sync.RWMutex
}

func NewFeatureHookTracker(registry *internaltesting.TagRegistry, opts *internaltesting.TestingOptions) *FeatureHookTracker {
	return &FeatureHookTracker{
		scenarios: newScenarioHookTracker(registry, opts),
		registry:  registry,
		opts:      opts,
		features:  make(map[string]*feature),
	}
}

func (f *FeatureHookTracker) Scenario(ctx context.Context, scenario *godog.Scenario) (context.Context, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	features := f.features[scenario.Uri]

	if !features.isRunning {
		opts := f.opts.Clone()

		features.isRunning = true
		features.opts = opts
		features.Cleaner = lifecycle.NewCleaner(godog.T(ctx), opts)

		var err error
		for _, fn := range f.registry.Handlers(features.tags.flatten()) {
			var cleanup func(context.Context) error
			// we make a throwaway context here so that we can leverage the
			// testing wrapper in our tag helpers -- any changes to the underlying
			// options will be propagated down the feature, scenario, step chain
			cleanup, err = fn.Handler(internaltesting.TestingContext(ctx, opts), fn.Suffix)
			if cleanup != nil {
				features.Cleanup(cleanup)
			}
		}

		f.features[scenario.Uri] = features

		if err != nil {
			return ctx, err
		}
	}

	return f.scenarios.start(ctx, scenario, features, func() {
		f.scenarioFailed(scenario)
	})
}

func (f *FeatureHookTracker) ScenarioFinished(ctx context.Context, scenario *godog.Scenario) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	features := f.features[scenario.Uri]
	features.scenariosToRun--

	f.features[scenario.Uri] = features

	f.scenarios.finish(ctx, scenario)
	if features.scenariosToRun <= 0 {
		delete(f.features, scenario.Uri)
		features.DoCleanup(ctx, features.hasStepFailure)
	}
}

func (f *FeatureHookTracker) SuiteFailed() bool {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	return f.failedSuite
}

func (f *FeatureHookTracker) scenarioFailed(scenario *godog.Scenario) {
	// this is always run with the mutex held due to being called
	// in the call stack of ScenarioFinished
	f.failedSuite = true

	features := f.features[scenario.Uri]
	features.hasStepFailure = true

	f.features[scenario.Uri] = features
}

// Feature tracks the feature when it is run.
func (f *FeatureHookTracker) Feature(doc *messages.GherkinDocument, uri string, data []byte) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	children := FilterChildren(f.opts.Provider, doc.Feature.Children)

	f.features[uri] = &feature{
		scenariosToRun: len(children),
		tags:           tagsForFeature(doc.Feature.Tags),
	}
}

func (f *FeatureHookTracker) RegisterFormatter(opts godog.Options) godog.Options {
	formatters.Format("custom", "", func(suite string, out io.Writer) formatters.Formatter {
		return f
	})
	opts.Format += ",custom"
	return opts
}

// This is all boilerplate to satisfy the formatters.Formatter interface

func (f *FeatureHookTracker) TestRunStarted() {}
func (f *FeatureHookTracker) Defined(*messages.Pickle, *messages.PickleStep, *formatters.StepDefinition) {
}
func (f *FeatureHookTracker) Pickle(pickle *messages.Pickle) {
}
func (f *FeatureHookTracker) Failed(*messages.Pickle, *messages.PickleStep, *formatters.StepDefinition, error) {
}
func (f *FeatureHookTracker) Passed(*messages.Pickle, *messages.PickleStep, *formatters.StepDefinition) {
}
func (f *FeatureHookTracker) Skipped(*messages.Pickle, *messages.PickleStep, *formatters.StepDefinition) {
}
func (f *FeatureHookTracker) Undefined(*messages.Pickle, *messages.PickleStep, *formatters.StepDefinition) {
}
func (f *FeatureHookTracker) Pending(*messages.Pickle, *messages.PickleStep, *formatters.StepDefinition) {
}
func (f *FeatureHookTracker) Summary() {}
