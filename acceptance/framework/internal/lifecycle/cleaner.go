package lifecycle

import (
	"context"
	"sync"
	"time"

	internaltesting "github.com/redpanda-data/redpanda-operator/acceptance/framework/internal/testing"
)

// Logger is an interface for a logger that the Cleaner
// can use on cleanup failures.
type Logger interface {
	Log(args ...interface{})
	Logf(format string, args ...interface{})
}

// Cleaner manages the registering and running of lifecycle
// hooks.
type Cleaner struct {
	logger          Logger
	retainOnFailure bool
	cleanupTimeout  time.Duration
	cleanupFns      []func(ctx context.Context, failed bool)
	mutex           sync.Mutex
}

// NewCleaner returns a new Cleaner object.
func NewCleaner(logger Logger, opt *internaltesting.TestingOptions) *Cleaner {
	return &Cleaner{
		logger:          logger,
		retainOnFailure: opt.RetainOnFailure,
		cleanupTimeout:  opt.CleanupTimeout,
	}
}

// Cleanup registers the callback to be called when DoCleanup is called.
func (c *Cleaner) Cleanup(fn internaltesting.CleanupFunc) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cleanupFns = append(c.cleanupFns, func(ctx context.Context, failed bool) {
		if failed && c.retainOnFailure {
			c.logger.Log("skipping cleanup due to test failure and retain flag being set")
			return
		}
		if err := fn(ctx); err != nil {
			c.logger.Logf("WARNING: error running cleanup hook: %v", err)
		}
	})
}

// DoCleanup calls all of the registered cleanup callbacks.
func (c *Cleaner) DoCleanup(ctx context.Context, failed bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	cancel := func() {}
	if c.cleanupTimeout != 0 {
		ctx, cancel = context.WithTimeout(ctx, c.cleanupTimeout)
	}
	defer cancel()

	for _, fn := range c.cleanupFns {
		fn(ctx, failed)
	}

	c.cleanupFns = nil
}
