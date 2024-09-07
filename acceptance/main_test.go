package main

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"github.com/redpanda-data/redpanda-operator/acceptance/environments"
	"github.com/redpanda-data/redpanda-operator/acceptance/steps"
	acceptancetesting "github.com/redpanda-data/redpanda-operator/acceptance/testing"
	"github.com/spf13/pflag"
)

var (
	opts = godog.Options{
		Output: colors.Colored(os.Stdout),
		Format: "pretty",
		// set random order runs by default
		Randomize: -1,
	}

	retain         bool
	kubeconfig     string
	timeout        time.Duration
	cleanupTimeout time.Duration
)

func init() {
	godog.BindCommandLineFlags("godog.", &opts)
	pflag.BoolVar(&retain, "retain", false, "retain resources when a scenario fails")
	pflag.DurationVar(&cleanupTimeout, "cleanup-timeout", 0, "timeout for running any cleanup routines after a scenario")
	pflag.DurationVar(&timeout, "timeout", 0, "timeout for running any individual test")
	pflag.StringVar(&kubeconfig, "kube-config", "", "path to kube-config to use for scenario runs")

}

func checkIrrecoverableError(err error) {
	if err != nil {
		fmt.Printf("error running tests: %v\n", err)
		os.Exit(1)
	}
}

func setShortTimeout(timeout *time.Duration, short time.Duration) {
	if testing.Short() && *timeout == 0 {
		*timeout = short
	}
}

func TestMain(m *testing.M) {
	pflag.Parse()
	opts.Paths = pflag.Args()

	setShortTimeout(&timeout, 2*time.Minute)
	setShortTimeout(&cleanupTimeout, 2*time.Minute)

	environment, err := environments.GetEnvironmentForTags(opts.Tags)
	checkIrrecoverableError(err)
	checkIrrecoverableError(environment.Initialize(m, &acceptancetesting.TestingOptions{
		RetainOnFailure: retain,
		Timeout:         timeout,
		CleanupTimeout:  cleanupTimeout,
		KubectlOptions:  acceptancetesting.NewKubectlOptions(kubeconfig),
	}))

	opts = environment.RegisterFormatter(opts)

	suite := godog.TestSuite{
		Name:                 "acceptance",
		TestSuiteInitializer: environment.TestSuiteInitializer,
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			environment.ScenarioInitializer(ctx)
			steps.Scenarios(ctx)
		},
		Options: &opts,
	}

	status := suite.Run()

	if st := m.Run(); st > status {
		status = st
	}

	os.Exit(status)
}

// stub test to make sure we don't get go test warning
// about no tests being found

func TestNoop(t *testing.T) {}
