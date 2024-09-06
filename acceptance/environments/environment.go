package environments

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

var (
	ErrMultipleTagsSpecified = errors.New("multiple tags detected, please pass only a single environment tag")
	ErrNoTagsSpecified       = errors.New("no tags detected, please run with an environment tag")
	ErrUnsupportedTag        = errors.New("unsupported tag")
)

// Environment defines the Kubernetes provider specific hooks
// that we use for running features.
type Environment interface {
	Initialize(m *testing.M) error
	BeforeSuite(ctx *godog.TestSuiteContext) error
	AfterSuite(ctx *godog.TestSuiteContext) error
	BeforeScenario(ctx context.Context, sc *godog.Scenario) (context.Context, error)
	AfterScenario(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error)
}

func noopInitialize(_ *testing.M) error               { return nil }
func noopBeforeSuite(_ *godog.TestSuiteContext) error { return nil }
func noopAfterSuite(_ *godog.TestSuiteContext) error  { return nil }
func noopBeforeScenario(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
	return ctx, nil
}
func noopAfterScenario(ctx context.Context, _ *godog.Scenario, err error) (context.Context, error) {
	return ctx, err
}

var supportedEnvironments = map[string]Environment{
	"@eks":  &eksEnvironment{},
	"@gke":  &gkeEnvironment{},
	"@aks":  &aksEnvironment{},
	"@kind": &kindEnvironment{},
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
