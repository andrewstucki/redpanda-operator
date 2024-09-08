package tags

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/redpanda-data/redpanda-operator/acceptance/framework"
)

func ClusterTag(ctx context.Context, suffix string) (func(context.Context) error, error) {
	if suffix == "" {
		return nil, errors.New("clusters tags can only be used with a suffix")
	}

	t := framework.T(ctx)

	cluster := filepath.Join("clusters", suffix)

	t.ApplyManifestNoCleanup(ctx, cluster)

	return func(ctx context.Context) error {
		t.DeleteManifest(ctx, cluster)
		return nil
	}, nil
}
