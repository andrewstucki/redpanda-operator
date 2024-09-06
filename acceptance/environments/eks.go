package environments

import (
	"context"
	"testing"

	"github.com/cucumber/godog"
)

type eksEnvironment struct{}

func (e *eksEnvironment) Initialize(m *testing.M) error {
	return noopInitialize(m)
}

func (e *eksEnvironment) BeforeSuite(ctx *godog.TestSuiteContext) error {
	return noopBeforeSuite(ctx)
}

func (e *eksEnvironment) AfterSuite(ctx *godog.TestSuiteContext) error {
	return noopAfterSuite(ctx)
}

func (e *eksEnvironment) BeforeScenario(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
	return noopBeforeScenario(ctx, sc)
}

func (e *eksEnvironment) AfterScenario(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
	return noopAfterScenario(ctx, sc, err)
}
