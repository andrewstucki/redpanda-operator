package tags

import (
	"context"
	"path/filepath"

	"github.com/redpanda-data/redpanda-operator/acceptance/framework"
	"github.com/stretchr/testify/require"
)

func ClusterTag(ctx context.Context, t framework.TestingT, args ...string) context.Context {
	require.Greater(t, len(args), 0, "clusters tags can only be used with additional arguments")
	name := args[0]

	t.ApplyManifest(ctx, filepath.Join("clusters", name))

	return ctx
}
