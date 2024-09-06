package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"github.com/redpanda-data/redpanda-operator/acceptance/definitions"
	"github.com/redpanda-data/redpanda-operator/acceptance/environments"
	"github.com/spf13/pflag"
)

var opts = godog.Options{
	Output: colors.Colored(os.Stdout),
	Format: "progress",
}

func init() {
	godog.BindCommandLineFlags("godog.", &opts)
}

func checkIrrecoverableError(err error) {
	if err != nil {
		fmt.Printf("error running tests: %v\n", err)
		os.Exit(1)
	}
}

func TestMain(m *testing.M) {
	pflag.Parse()
	opts.Paths = pflag.Args()

	environment, err := environments.GetEnvironmentForTags(opts.Tags)
	checkIrrecoverableError(err)
	checkIrrecoverableError(environment.Initialize(m))

	status := godog.TestSuite{
		Name: "acceptance",
		TestSuiteInitializer: func(ctx *godog.TestSuiteContext) {
			checkIrrecoverableError(environment.BeforeSuite(ctx))
			checkIrrecoverableError(environment.AfterSuite(ctx))
		},
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			ctx.Before(environment.BeforeScenario)
			ctx.After(environment.AfterScenario)
			definitions.Scenarios(ctx)
		},
		Options: &opts,
	}.Run()

	if st := m.Run(); st > status {
		status = st
	}

	os.Exit(status)
}
