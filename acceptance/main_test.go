package main

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/redpanda-data/helm-charts/pkg/helm"
	"github.com/redpanda-data/redpanda-operator/acceptance/framework"
	_ "github.com/redpanda-data/redpanda-operator/acceptance/steps"
	"github.com/redpanda-data/redpanda-operator/acceptance/tags"
)

var (
	imageRepo = "localhost/redpanda-operator"
	imageTag  = "dev"
	suite     *framework.Suite
)

func TestMain(m *testing.M) {
	var err error

	suite, err = framework.SuiteBuilderFromFlags().
		RegisterProvider("eks", framework.NoopProvider).
		RegisterProvider("gke", framework.NoopProvider).
		RegisterProvider("aks", framework.NoopProvider).
		RegisterProvider("k3d", framework.NoopProvider).
		WithDefaultProvider("k3d").
		WithHelmChart("https://charts.jetstack.io", "jetstack", "cert-manager", helm.InstallOptions{
			Name:            "cert-manager",
			Namespace:       "cert-manager",
			Version:         "v1.14.2",
			CreateNamespace: true,
			Values: map[string]any{
				"installCRDs": true,
			},
		}).
		WithCRDDirectory("../src/go/k8s/config/crd/bases").
		WithCRDDirectory("../src/go/k8s/config/crd/bases/toolkit.fluxcd.io").
		OnFeature(func(ctx context.Context, t framework.TestingT) {
			t.InstallHelmChart(ctx, "https://charts.redpanda.com", "redpanda", "operator", helm.InstallOptions{
				Name:      "redpanda-operator",
				Namespace: t.IsolateNamespace(ctx),
				Values: map[string]any{
					"logLevel": "trace",
					"image": map[string]any{
						"tag":        imageTag,
						"repository": imageRepo,
					},
					"rbac": map[string]any{
						"createAdditionalControllerCRs": true,
						"createRPKBundleCRs":            true,
					},
					"additionalCmdFlags": []string{"--additional-controllers=all"},
				},
			})
		}).
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
