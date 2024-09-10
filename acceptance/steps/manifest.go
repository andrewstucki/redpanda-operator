package steps

import (
	"context"
	"os"

	"github.com/cucumber/godog"
	framework "github.com/redpanda-data/redpanda-operator/harpoon"
	"github.com/stretchr/testify/require"
)

func iApplyKubernetesManifest(ctx context.Context, manifest *godog.DocString) error {
	t := framework.T(ctx)

	file, err := os.CreateTemp("", "manifest-*.yaml")
	require.NoError(t, err)

	_, err = file.Write([]byte(manifest.Content))
	require.NoError(t, err)
	require.NoError(t, file.Close())

	t.ApplyManifest(ctx, file.Name())

	t.Cleanup(func(ctx context.Context) {
		require.NoError(t, os.RemoveAll(file.Name()))
	})

	return nil
}
