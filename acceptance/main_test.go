package main

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/redpanda-data/helm-charts/pkg/helm"
	"github.com/redpanda-data/redpanda-operator/acceptance/framework"
	_ "github.com/redpanda-data/redpanda-operator/acceptance/steps"
	"github.com/redpanda-data/redpanda-operator/acceptance/tags"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	imageRepo = "localhost/redpanda-operator"
	imageTag  = "dev"
	suite     *framework.Suite
)

func TestMain(m *testing.M) {
	log.SetLogger(logr.Discard())

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
			t.Log("Installing Redpanda operator chart")
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
			t.Log("Successfully installed Redpanda operator chart")

			// hack until we get the RBAC configuration for users copied over to the helm chart
			var role rbacv1.Role
			require.NoError(t, t.Get(ctx, t.ResourceKey("redpanda-operator"), &role))
			role.Rules = append(role.Rules, []rbacv1.PolicyRule{{
				Verbs:     []string{"get", "list", "patch", "update", "watch"},
				APIGroups: []string{"cluster.redpanda.com"},
				Resources: []string{"users"},
			}, {
				Verbs:     []string{"update"},
				APIGroups: []string{"cluster.redpanda.com"},
				Resources: []string{"users/finalizers"},
			}, {
				Verbs:     []string{"get", "patch", "update"},
				APIGroups: []string{"cluster.redpanda.com"},
				Resources: []string{"users/status"},
			}}...)

			require.NoError(t, t.Update(ctx, &role))
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
