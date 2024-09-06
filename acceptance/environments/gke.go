package environments

import (
	"context"
	"testing"

	"github.com/cucumber/godog"
)

type gkeEnvironment struct{}

func (g *gkeEnvironment) Initialize(m *testing.M) error {
	return noopInitialize(m)
}

func (g *gkeEnvironment) BeforeSuite(ctx *godog.TestSuiteContext) error {
	return noopBeforeSuite(ctx)
}

func (g *gkeEnvironment) AfterSuite(ctx *godog.TestSuiteContext) error {
	return noopAfterSuite(ctx)
}

func (g *gkeEnvironment) BeforeScenario(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
	return noopBeforeScenario(ctx, sc)
}

func (g *gkeEnvironment) AfterScenario(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
	return noopAfterScenario(ctx, sc, err)
}
