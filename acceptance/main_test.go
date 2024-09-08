package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/redpanda-data/redpanda-operator/acceptance/framework"
	_ "github.com/redpanda-data/redpanda-operator/acceptance/steps"
	"github.com/redpanda-data/redpanda-operator/acceptance/tags"
)

var suite *framework.Suite

func TestMain(m *testing.M) {
	var err error

	suite, err = framework.SuiteBuilderFromFlags().
		RegisterProvider("eks", framework.NoopProvider).
		RegisterProvider("gke", framework.NoopProvider).
		RegisterProvider("aks", framework.NoopProvider).
		RegisterProvider("k3d", framework.NoopProvider).
		WithDefaultProvider("k3d").
		RegisterTag("cluster", 1, tags.ClusterTag).
		Build()

	if err != nil {
		fmt.Printf("error running test suite: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestSuite(t *testing.T) {
	suite.RunT(t)
}
