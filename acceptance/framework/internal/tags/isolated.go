package tags

import (
	"context"

	internaltesting "github.com/redpanda-data/redpanda-operator/acceptance/framework/internal/testing"
)

func IsolatedTag(ctx context.Context, suffix string) (func(context.Context) error, error) {
	t := internaltesting.T(ctx)

	oldNamespace := t.Namespace()
	newNamespace := t.CreateNamespace(ctx)
	t.SetNamespace(newNamespace)

	return func(ctx context.Context) error {
		t.DeleteNamespace(ctx, newNamespace)
		t.SetNamespace(oldNamespace)

		return nil
	}, nil
}
