package framework

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"github.com/redpanda-data/redpanda-operator/acceptance/framework/internal/tags"
	internaltesting "github.com/redpanda-data/redpanda-operator/acceptance/framework/internal/testing"
	"github.com/redpanda-data/redpanda-operator/acceptance/framework/internal/tracking"
)

func setShortTimeout(timeout *time.Duration, short time.Duration) {
	if testing.Short() && *timeout == 0 {
		*timeout = short
	}
}

type SuiteBuilder struct {
	testingOpts     *internaltesting.TestingOptions
	opts            godog.Options
	registry        *internaltesting.TagRegistry
	providers       map[string]Provider
	defaultProvider string
	injectors       []func(context.Context) context.Context
}

func SuiteBuilderFromFlags() *SuiteBuilder {
	godogOpts := godog.Options{Output: colors.Colored(os.Stdout)}

	var config string
	options := &internaltesting.TestingOptions{}

	flag.BoolVar(&options.RetainOnFailure, "retain", false, "retain resources when a scenario fails")
	flag.DurationVar(&options.CleanupTimeout, "cleanup-timeout", 0, "timeout for running any cleanup routines after a scenario")
	flag.DurationVar(&options.Timeout, "timeout", 0, "timeout for running any individual test")
	flag.StringVar(&config, "kube-config", "", "path to kube-config to use for scenario runs")
	flag.StringVar(&options.Provider, "provider", "", "provider for the test suite")
	flag.StringVar(&godogOpts.Format, "output-format", "pretty", "godog output format")
	flag.BoolVar(&godogOpts.NoColors, "no-color", false, "print only in black and white")
	flag.Int64Var(&godogOpts.Randomize, "seed", -1, "seed for tests, set to -1 for a random seed")

	flag.Parse()

	options.KubectlOptions = internaltesting.NewKubectlOptions(config)

	setShortTimeout(&options.Timeout, 2*time.Minute)
	setShortTimeout(&options.CleanupTimeout, 2*time.Minute)

	registry := internaltesting.NewTagRegistry()
	registry.Register("isolated", -1000, tags.IsolatedTag)

	return &SuiteBuilder{
		testingOpts: options,
		opts:        godogOpts,
		registry:    registry,
		providers:   make(map[string]Provider),
	}
}

func (b *SuiteBuilder) WithDefaultProvider(name string) *SuiteBuilder {
	b.defaultProvider = name
	return b
}

func (b *SuiteBuilder) RegisterTag(tag string, priority int, handler TagHandler) *SuiteBuilder {
	b.registry.Register(tag, priority, internaltesting.TagHandler(handler))
	return b
}

func (b *SuiteBuilder) RegisterProvider(name string, provider Provider) *SuiteBuilder {
	b.providers[name] = provider
	return b
}

func (b *SuiteBuilder) InjectContext(fn func(context.Context) context.Context) *SuiteBuilder {
	b.injectors = append(b.injectors, fn)
	return b
}

func (b *SuiteBuilder) Build() (*Suite, error) {
	tracker := tracking.NewFeatureHookTracker(b.registry, b.testingOpts)
	opts := tracker.RegisterFormatter(b.opts)

	providerName := b.testingOpts.Provider
	if providerName == "" {
		providerName = b.defaultProvider
	}

	provider, ok := b.providers[providerName]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %q", providerName)
	}

	if err := provider.Initialize(); err != nil {
		return nil, err
	}
	ctx := provider.GetBaseContext()
	opts.DefaultContext = ctx
	opts.Tags = fmt.Sprintf("~@skip:%s", providerName)

	return &Suite{
		suite: &godog.TestSuite{
			Name: "acceptance",
			TestSuiteInitializer: func(suiteContext *godog.TestSuiteContext) {
				suiteContext.BeforeSuite(func() {
					if err := provider.Setup(ctx); err != nil {
						fmt.Printf("error setting up test suite: %v\n", err)
						os.Exit(1)
					}
				})
				suiteContext.AfterSuite(func() {
					cancel := func() {}
					cleanupTimeout := b.testingOpts.CleanupTimeout
					if cleanupTimeout != 0 {
						ctx, cancel = context.WithTimeout(ctx, cleanupTimeout)
					}
					defer cancel()

					if tracker.SuiteFailed() && b.testingOpts.RetainOnFailure {
						fmt.Println("skipping cleanup due to test failure and retain flag being set")
						return
					}
					if err := provider.Teardown(ctx); err != nil {
						fmt.Printf("WARNING: error running provider teardown: %v\n", err)
					}
				})
			},
			ScenarioInitializer: func(ctx *godog.ScenarioContext) {
				ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
					ctx, err := tracker.Scenario(ctx, sc)
					if err != nil {
						return nil, err
					}
					for _, fn := range b.injectors {
						ctx = fn(ctx)
					}
					return ctx, nil
				})
				ctx.After(func(ctx context.Context, sc *godog.Scenario, _ error) (context.Context, error) {
					tracker.ScenarioFinished(ctx, sc)
					return ctx, nil
				})

				getSteps(ctx)
			},
			Options: &opts,
		},
		options: b.testingOpts,
	}, nil
}

type Suite struct {
	suite   *godog.TestSuite
	options *internaltesting.TestingOptions
}

func (s *Suite) RunM(m *testing.M) {
	status := s.suite.Run()

	if st := m.Run(); st > status {
		status = st
	}

	os.Exit(status)
}

func (s *Suite) RunT(t *testing.T) {
	s.suite.Options.TestingT = t
	s.suite.Run()
}
