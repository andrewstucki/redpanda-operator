package testing

import (
	"context"
	"time"

	"github.com/cucumber/godog"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type contextKey struct{}

var (
	testingContextKey contextKey = struct{}{}
	noopCancel                   = func() {}
)

// TestingOptions are configurable options for the testing environment
type TestingOptions struct {
	// RetainOnFailure tells the testing environment to retain
	// any provisioned resources in the case of a test failure.
	RetainOnFailure bool
	// KubectlOptions sets the options passed to the underlying
	// kubectl commands.
	KubectlOptions *KubectlOptions
	// Timeout is the timeout for any individual scenario to run.
	Timeout time.Duration
	// CleanupTimeout is the timeout for cleaning up resources
	CleanupTimeout time.Duration
	// Provider sets the provider for the test suite.
	Provider string
}

func (o *TestingOptions) Clone() *TestingOptions {
	return &TestingOptions{
		RetainOnFailure: o.RetainOnFailure,
		Timeout:         o.Timeout,
		CleanupTimeout:  o.CleanupTimeout,
		KubectlOptions:  o.KubectlOptions.Clone(),
		Provider:        o.Provider,
	}
}

// TestingT is a wrapper around godog's test implementation that
// itself wraps go's stdlib testing.T. It adds some helpers on top
// of godog and implements some of the functionality that godog doesn't
// expose when wrapping testing.T (i.e. Cleanup methods) as well as
// acting as an entry point for initializing connections to infrastructure
// resources.
type TestingT struct {
	godog.TestingT
	client.Client

	restConfig *rest.Config
	options    *TestingOptions
	cleanupFns []func(ctx context.Context)
	failure    bool
}

// TestingContext injects a TestingT into the given context.
func TestingContext(ctx context.Context, options *TestingOptions) context.Context {
	t := godog.T(ctx)

	client, err := kubernetesClient(options.KubectlOptions)
	require.NoError(t, err)

	restConfig, err := restConfig(options.KubectlOptions)
	require.NoError(t, err)

	return context.WithValue(ctx, testingContextKey, &TestingT{
		TestingT:   t,
		Client:     client,
		restConfig: restConfig,
		options:    options,
	})
}

// T pulls a TestingT from the given context.
func T(ctx context.Context) *TestingT {
	return ctx.Value(testingContextKey).(*TestingT)
}

// Cleanup registers a cleanup hook on the test that
// will run after the scenario finishes.
func (t *TestingT) Cleanup(fn func(context.Context) error) {
	t.cleanup(false, fn)
}

// DoCleanup calls the cleanup functions on the TestingT in the
// test context.
func (t *TestingT) DoCleanup(ctx context.Context) {
	cancel := noopCancel
	if t.options.CleanupTimeout != 0 {
		ctx, cancel = context.WithTimeout(ctx, t.options.CleanupTimeout)
	}
	defer cancel()

	for _, fn := range t.cleanupFns {
		fn(ctx)
	}
}

// IsFailure returns whether this test failed.
func (t *TestingT) IsFailure() bool {
	return t.failure
}

// Error fails the current test and logs the provided arguments. Equivalent to calling Log then
// Fail.
func (t *TestingT) Error(args ...interface{}) {
	t.failure = true
}

// Errorf fails the current test and logs the formatted message. Equivalent to calling Logf then
// Fail.
func (t *TestingT) Errorf(format string, args ...interface{}) {
	t.failure = true
	t.TestingT.Errorf(format, args...)
}

// Fail marks the current test as failed, but does not halt execution of the step.
func (t *TestingT) Fail() {
	t.failure = true
	t.TestingT.Fail()
}

// FailNow marks the current test as failed and halts execution of the step.
func (t *TestingT) FailNow() {
	t.failure = true
	t.TestingT.FailNow()
}

// Fatal logs the provided arguments, marks the test as failed and halts execution of the step.
func (t *TestingT) Fatal(args ...interface{}) {
	t.failure = true
	t.TestingT.Fatal(args...)
}

// Fatal logs the formatted message, marks the test as failed and halts execution of the step.
func (t *TestingT) Fatalf(format string, args ...interface{}) {
	t.failure = true
	t.TestingT.Fatalf(format, args...)
}

// ApplyFixture applies a set of kubernetes manifests via kubectl.
func (t *TestingT) ApplyFixture(ctx context.Context, fileOrDirectory string) {
	t.ApplyNamespacedFixture(ctx, fileOrDirectory, t.options.KubectlOptions.Namespace)
}

// ApplyNamespacedFixture applies a set of kubernetes manifests via kubectl.
func (t *TestingT) ApplyNamespacedFixture(ctx context.Context, fileOrDirectory, namespace string) {
	opts := t.options.KubectlOptions.Clone()
	opts.Namespace = namespace

	_, err := kubectlApply(ctx, fileOrDirectory, opts)
	require.NoError(t, err)

	t.cleanup(true, func(ctx context.Context) error {
		_, err := kubectlDelete(ctx, fileOrDirectory, opts)
		return err
	})
}

// ResourceKey returns a types.NamespaceName that can be used with the Kubernetes client,
// but scoped to the namespace given by the underlying KubectlOptions.
func (t *TestingT) ResourceKey(name string) types.NamespacedName {
	return types.NamespacedName{
		Namespace: t.Namespace(),
		Name:      name,
	}
}

// Namespace returns the namespace that this testing scenario is running in.
func (t *TestingT) Namespace() string {
	return t.options.KubectlOptions.Namespace
}

// SetNamespace sets the current namespace we're in.
func (t *TestingT) SetNamespace(namespace string) {
	t.options.KubectlOptions.Namespace = namespace
}

// CreateNamespace creates a temporary namespace for the tests in this scenario to run.
func (t *TestingT) CreateNamespace(ctx context.Context) string {
	namespace := AddSuffix("scenario")
	err := createNamespace(ctx, namespace, t.options.KubectlOptions)
	require.NoError(t, err)
	return namespace
}

// DeleteNamespace deletes a temporary namespace for the tests in this scenario to run.
func (t *TestingT) DeleteNamespace(ctx context.Context, namespace string) {
	err := deleteNamespace(ctx, namespace, t.options.KubectlOptions)
	require.NoError(t, err)
}

func (t *TestingT) cleanup(checkRetain bool, fn func(context.Context) error) {
	t.cleanupFns = append(t.cleanupFns, func(ctx context.Context) {
		if checkRetain && t.failure && t.options.RetainOnFailure {
			t.Log("skipping cleanup due to test failure and retain flag being set")
			return
		}
		if err := fn(ctx); err != nil {
			t.Logf("WARNING: error running cleanup hook: %v", err)
		}
	})
}
