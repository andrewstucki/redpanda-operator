package environments

import (
	"context"
	"testing"

	"github.com/cucumber/godog"
)

type aksEnvironment struct{}

func (a *aksEnvironment) Initialize(m *testing.M) error {
	return noopInitialize(m)
}

func (a *aksEnvironment) BeforeSuite(ctx *godog.TestSuiteContext) error {
	return noopBeforeSuite(ctx)
}

func (a *aksEnvironment) AfterSuite(ctx *godog.TestSuiteContext) error {
	return noopAfterSuite(ctx)
}

func (a *aksEnvironment) BeforeScenario(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
	return noopBeforeScenario(ctx, sc)
}

func (a *aksEnvironment) AfterScenario(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
	return noopAfterScenario(ctx, sc, err)
}
