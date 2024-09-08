package framework

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"testing"

	corev1 "k8s.io/api/core/v1"

	"github.com/stretchr/testify/require"
)

var suite *Suite

func TestMain(m *testing.M) {
	var err error

	suite, err = SuiteBuilderFromFlags().
		RegisterProvider("stub", NoopProvider).
		WithDefaultProvider("stub").
		Build()

	if err != nil {
		fmt.Printf("error running test suite: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestSuite(t *testing.T) {
	suite.RunT(t)
}

func stubGiven(ctx context.Context) error {
	T(ctx).ApplyFixture(ctx, "stub")

	return nil
}

func stubWhen(ctx context.Context, key, value string) error {
	t := T(ctx)

	var configMap corev1.ConfigMap
	require.NoError(t, t.Get(ctx, t.ResourceKey("stub-config-map"), &configMap))

	configMap.Data = map[string]string{key: value}

	require.NoError(t, t.Update(ctx, &configMap))

	return nil
}

func stubThen(ctx context.Context, key, value string) error {
	t := T(ctx)

	var configMap corev1.ConfigMap
	require.NoError(t, t.Get(ctx, t.ResourceKey("stub-config-map"), &configMap))

	require.Equal(t, value, configMap.Data[key])

	return nil
}

func init() {
	RegisterStep(`^there is a stub$`, stubGiven)
	RegisterStep(`^a user updates the stub key "([^"]*)" to "([^"]*)"$`, stubWhen)
	RegisterStep(`^the stub should have "([^"]*)" equal "([^"]*)"$`, stubThen)

	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGQUIT)
		buf := make([]byte, 1<<20)
		for {
			<-sigs
			stacklen := runtime.Stack(buf, true)
			log.Printf("=== received SIGQUIT ===\n*** goroutine dump...\n%s\n*** end\n", buf[:stacklen])
		}
	}()
}
