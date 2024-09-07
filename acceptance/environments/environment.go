package environments

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/formatters"
	acceptancetesting "github.com/redpanda-data/redpanda-operator/acceptance/testing"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	ErrMultipleTagsSpecified = errors.New("multiple tags detected, please pass only a single environment tag")
	ErrNoTagsSpecified       = errors.New("no tags detected, please run with an environment tag")
	ErrUnsupportedTag        = errors.New("unsupported tag")
)

// Environment defines the Kubernetes provider specific hooks
// that we use for running features.
type Environment interface {
	Initialize(m *testing.M, opts *acceptancetesting.TestingOptions) error
	TestSuiteInitializer(ctx *godog.TestSuiteContext)
	ScenarioInitializer(ctx *godog.ScenarioContext)
	RegisterFormatter(opts godog.Options) godog.Options
}

type baseEnvironment struct {
	opts            *acceptancetesting.TestingOptions
	tracker         *featureHookTracker
	scenarioTracker *scenarioHookTracker
}

func (b *baseEnvironment) Initialize(m *testing.M, opts *acceptancetesting.TestingOptions) error {
	b.opts = opts
	b.tracker = newFeatureHookTracker()
	b.scenarioTracker = newScenarioHookTracker()
	return nil
}

func (b *baseEnvironment) TestSuiteInitializer(ctx *godog.TestSuiteContext) {}

func (b *baseEnvironment) ScenarioInitializer(ctx *godog.ScenarioContext) {
	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		b.tracker.onFirstRun(sc.Uri, func(namespace string) {
			b.onFeatureStart(ctx, namespace)
		})

		return b.scenarioTracker.add(ctx, sc.Id, b.tracker.namespace(sc.Uri), b.opts), nil
	})

	ctx.After(func(ctx context.Context, sc *godog.Scenario, _ error) (context.Context, error) {
		b.scenarioTracker.cleanup(ctx, sc.Id)

		b.tracker.onFinished(sc.Uri, func(namespace string) {
			b.onFeatureEnd(ctx, namespace)
		})

		return ctx, nil
	})
}

func (b *baseEnvironment) onFeatureStart(ctx context.Context, namespace string) {
	// generate a new namespace per feature to run features in isolation
	// and potentially clean it up after the feature finishes running

	opts := b.opts.Clone()
	opts.KubectlOptions = opts.KubectlOptions.WithNamespace(namespace)

	client, err := acceptancetesting.Client(opts.KubectlOptions)
	require.NoError(godog.T(ctx), err)

	namespaceObject := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: opts.KubectlOptions.Namespace,
		},
	}

	require.NoError(godog.T(ctx), client.Create(ctx, namespaceObject))
}

func (b *baseEnvironment) onFeatureEnd(ctx context.Context, namespace string) {
	opts := b.opts.Clone()
	opts.KubectlOptions = opts.KubectlOptions.WithNamespace(namespace)

	cancel := func() {}
	if b.opts.CleanupTimeout != 0 {
		ctx, cancel = context.WithTimeout(ctx, b.opts.CleanupTimeout)
	}
	defer cancel()

	// we can safely call delete on a namespace that's not empty
	// because a finalizer will keep it from getting GC'd
	// so no need to check retention here
	namespaceObject := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	client, err := acceptancetesting.Client(opts.KubectlOptions)
	require.NoError(godog.T(ctx), err)

	if err := client.Delete(ctx, namespaceObject); err != nil {
		godog.T(ctx).Logf("WARNING: error running cleanup hook: %v", err)
	}
}

func (b *baseEnvironment) RegisterFormatter(opts godog.Options) godog.Options {
	formatters.Format("custom", "", func(suite string, out io.Writer) formatters.Formatter {
		return b.tracker
	})
	opts.Format += ",custom"
	return opts
}

var supportedEnvironments = map[string]Environment{
	"@eks":  &eksEnvironment{&baseEnvironment{}},
	"@gke":  &gkeEnvironment{&baseEnvironment{}},
	"@aks":  &aksEnvironment{&baseEnvironment{}},
	"@kind": &kindEnvironment{&baseEnvironment{}},
}

// GetEnvironmentForTags returns the environment used to run the
// the acceptance tests or an error if the environment cannot be
// detected based on the tags passed it.
func GetEnvironmentForTags(tags string) (Environment, error) {
	tags = strings.Trim(tags, " ")
	if tags == "" {
		return nil, ErrNoTagsSpecified
	}

	splitTags := strings.Split(tags, " ")
	if len(splitTags) != 1 {
		return nil, ErrMultipleTagsSpecified
	}

	tag := splitTags[0]

	if environment, ok := supportedEnvironments[tag]; ok {
		return environment, nil
	}

	return nil, fmt.Errorf("%w: %s", ErrUnsupportedTag, tag)
}
