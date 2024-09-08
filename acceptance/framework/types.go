package framework

import (
	"context"

	"github.com/cucumber/godog"
	"k8s.io/apimachinery/pkg/types"

	internaltesting "github.com/redpanda-data/redpanda-operator/acceptance/framework/internal/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Redefine the interfaces from the internal package

type TagHandler func(context.Context) (func(context.Context) error, error)

type TestingT interface {
	godog.TestingT
	client.Client

	Cleanup(fn func(context.Context) error)
	ResourceKey(name string) types.NamespacedName
	ApplyFixture(ctx context.Context, fileOrDirectory string)
	ApplyNamespacedFixture(ctx context.Context, fileOrDirectory, namespace string)
	CreateNamespace(ctx context.Context) string
	DeleteNamespace(ctx context.Context, namespace string)
}

type Provider interface {
	Initialize() error
	Setup(ctx context.Context) error
	Teardown(ctx context.Context) error
	GetBaseContext() context.Context
}

func T(ctx context.Context) TestingT {
	return internaltesting.T(ctx)
}

var NoopProvider = &noopProvider{}

type noopProvider struct{}

func (n *noopProvider) Initialize() error {
	return nil
}

func (n *noopProvider) Setup(ctx context.Context) error {
	return nil
}

func (n *noopProvider) Teardown(ctx context.Context) error {
	return nil
}

func (n *noopProvider) GetBaseContext() context.Context {
	return context.Background()
}
