package steps

import (
	"context"

	"github.com/redpanda-data/redpanda-operator/acceptance/testing"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func init() {
	RegisterStep(`^I am in a random namespace$`, stubCreateNamespace)
	RegisterStep(`^there is a stub$`, stubGiven)
	RegisterStep(`^a user updates the stub key "([^"]*)" to "([^"]*)"$`, stubWhen)
	RegisterStep(`^the stub should have "([^"]*)" equal "([^"]*)"$`, stubThen)
}

func stubCreateNamespace(ctx context.Context) error {
	testing.T(ctx).CreateScenarioNamespace(ctx)

	return nil
}

func stubGiven(ctx context.Context) error {
	testing.T(ctx).ApplyFixture(ctx, "stub")

	return nil
}

func stubWhen(ctx context.Context, key, value string) error {
	t := testing.T(ctx)

	var configMap corev1.ConfigMap
	require.NoError(t, t.Get(ctx, t.ResourceKey("stub-config-map"), &configMap))

	configMap.Data = map[string]string{key: value}

	require.NoError(t, t.Update(ctx, &configMap))

	return nil
}

func stubThen(ctx context.Context, key, value string) error {
	t := testing.T(ctx)

	var configMap corev1.ConfigMap
	require.NoError(t, t.Get(ctx, t.ResourceKey("stub-config-map"), &configMap))

	require.Equal(t, value, configMap.Data[key])

	return nil
}
