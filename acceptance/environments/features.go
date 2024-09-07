package environments

import (
	"sync"

	"github.com/cucumber/godog/formatters"
	messages "github.com/cucumber/messages/go/v21"
	acceptancetesting "github.com/redpanda-data/redpanda-operator/acceptance/testing"
)

// this exists since godog doesn't have any before/after
// feature hooks, it acts by reference counting the number
// of times a before hook has been called and then decrementing
// a counter based on the number of scenarios that should
// be run per-feature based on the nasty hack of hooking
// into the formatter interface of godog.
type featureHookTracker struct {
	features   map[string]int
	hasRun     map[string]bool
	namespaces map[string]string
	mutex      sync.RWMutex
}

func newFeatureHookTracker() *featureHookTracker {
	return &featureHookTracker{
		features:   make(map[string]int),
		hasRun:     make(map[string]bool),
		namespaces: make(map[string]string),
	}
}

func (f *featureHookTracker) Feature(doc *messages.GherkinDocument, uri string, data []byte) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.features[uri] = len(doc.Feature.Children)
}

func (f *featureHookTracker) namespace(uri string) string {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	return f.namespaces[uri]
}

func (f *featureHookTracker) clear(uri string) {
	delete(f.features, uri)
	delete(f.hasRun, uri)
	delete(f.namespaces, uri)
}

func (f *featureHookTracker) onFirstRun(uri string, fn func(namespace string)) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	firstRun := !f.hasRun[uri]
	if firstRun {
		namespace := acceptancetesting.AddSuffix("testing")
		f.hasRun[uri] = true
		f.namespaces[uri] = namespace
		fn(namespace)
	}
}

func (f *featureHookTracker) onFinished(uri string, fn func(namespace string)) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.features[uri] = f.features[uri] - 1

	if f.features[uri] <= 0 {
		namespace := f.namespaces[uri]
		fn(namespace)

		f.clear(uri)
	}
}

// This is all boilerplate to satisfy the formatters.Formatter interface

func (f *featureHookTracker) TestRunStarted() {}
func (f *featureHookTracker) Defined(*messages.Pickle, *messages.PickleStep, *formatters.StepDefinition) {
}
func (f *featureHookTracker) Pickle(pickle *messages.Pickle) {
}
func (f *featureHookTracker) Failed(*messages.Pickle, *messages.PickleStep, *formatters.StepDefinition, error) {
}
func (f *featureHookTracker) Passed(*messages.Pickle, *messages.PickleStep, *formatters.StepDefinition) {
}
func (f *featureHookTracker) Skipped(*messages.Pickle, *messages.PickleStep, *formatters.StepDefinition) {
}
func (f *featureHookTracker) Undefined(*messages.Pickle, *messages.PickleStep, *formatters.StepDefinition) {
}
func (f *featureHookTracker) Pending(*messages.Pickle, *messages.PickleStep, *formatters.StepDefinition) {
}
func (f *featureHookTracker) Summary() {}
