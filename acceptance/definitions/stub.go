package definitions

import (
	"context"

	"github.com/cucumber/godog"
)

func init() {
	RegisterStep("^there is a stub$", stub)
	RegisterStep("^a user stubs$", stub)
	RegisterStep("^there should be a stub$", stub)
}

func stub(ctx context.Context) error {
	return godog.ErrPending
}
