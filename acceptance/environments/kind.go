package environments

import (
	"context"
	"testing"

	"github.com/cucumber/godog"
)

type kindEnvironment struct{}

func (k *kindEnvironment) Initialize(m *testing.M) error {
	return noopInitialize(m)
}

func (k *kindEnvironment) BeforeSuite(ctx *godog.TestSuiteContext) error {
	return noopBeforeSuite(ctx)
}

func (k *kindEnvironment) AfterSuite(ctx *godog.TestSuiteContext) error {
	return noopAfterSuite(ctx)
}

func (k *kindEnvironment) BeforeScenario(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
	return noopBeforeScenario(ctx, sc)
}

func (k *kindEnvironment) AfterScenario(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
	return noopAfterScenario(ctx, sc, err)
}
