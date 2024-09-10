package framework

import (
	"context"
)

func isolatedTag(ctx context.Context, t TestingT, args ...string) context.Context {
	t.IsolateNamespace(ctx)
	return ctx
}
