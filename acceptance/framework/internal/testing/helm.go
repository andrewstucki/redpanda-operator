package testing

import (
	"context"

	"github.com/redpanda-data/helm-charts/pkg/helm"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
)

func (t *TestingT) InstallHelmChart(ctx context.Context, url, repo, chart string, options helm.InstallOptions) {
	helmClient, err := helm.New(helm.Options{
		KubeConfig: rest.CopyConfig(t.restConfig),
	})
	require.NoError(t, err)
	require.NoError(t, helmClient.RepoAdd(ctx, repo, url))
	require.NotEqual(t, "", options.Namespace, "namespace must not be blank")
	require.NotEqual(t, "", options.Name, "name must not be blank")

	options.CreateNamespace = true

	_, err = helmClient.Install(ctx, repo+"/"+chart, options)
	require.NoError(t, err)

	t.Cleanup(func(ctx context.Context) {
		require.NoError(t, helmClient.Uninstall(ctx, helm.Release{
			Name:      options.Name,
			Namespace: options.Namespace,
		}))
	})
}
