package redpanda

import (
	"context"
	"testing"
	"time"

	gofakeit "github.com/brianvoe/gofakeit/v7"
	"github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha2"
	"github.com/redpanda-data/redpanda-operator/src/go/k8s/internal/clients"
	"github.com/redpanda-data/redpanda-operator/src/go/k8s/internal/testutils"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/redpanda"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type testCluster struct {
	client.Client
	clients.ClientFactory

	kafkaAPISpec *v1alpha2.KafkaAPISpec
	adminAPISpec *v1alpha2.AdminAPISpec
	namespace    string
}

func (c *testCluster) FetchLatest(t *testing.T, ctx context.Context, o client.Object) {
	require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(o), o))
}

func (c *testCluster) UpdateObject(t *testing.T, ctx context.Context, o client.Object) {
	require.NoError(t, c.Update(ctx, o))
}

func (c *testCluster) DeleteObject(t *testing.T, ctx context.Context, o client.Object) {
	require.NoError(t, c.Delete(ctx, o))
}

func createTestCluster(t *testing.T, ctx context.Context) *testCluster {
	testEnv := testutils.RedpandaTestEnv{}
	cfg, err := testEnv.StartRedpandaTestEnv(false)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	err = v1alpha2.AddToScheme(scheme.Scheme)
	require.NoError(t, err)

	c, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	require.NoError(t, err)
	require.NotNil(t, c)

	factory, err := clients.NewClientFactory(cfg)
	require.NoError(t, err)

	namespace := gofakeit.UUID()
	secretName := gofakeit.UUID()
	secretKey := gofakeit.UUID()
	superuser := gofakeit.UUID()
	password := gofakeit.UUID()

	container, err := redpanda.Run(
		ctx,
		"docker.redpanda.com/redpandadata/redpanda:v24.2.1",
		redpanda.WithEnableSASL(),
		redpanda.WithNewServiceAccount(superuser, password),
		redpanda.WithSuperusers(superuser),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		ctxCleanup, cancelCleanup := context.WithTimeout(context.Background(), time.Minute*2)
		defer cancelCleanup()
		if err = container.Terminate(ctxCleanup); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	})

	seedBroker, err := container.KafkaSeedBroker(ctx)
	require.NoError(t, err)

	adminEndpoint, err := container.AdminAPIAddress(ctx)
	require.NoError(t, err)

	require.NoError(t, c.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespace,
			Namespace: namespace,
		},
	}))

	require.NoError(t, c.Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		StringData: map[string]string{
			secretKey: password,
		},
	}))

	return &testCluster{
		Client:        c,
		ClientFactory: factory,
		namespace:     namespace,
		kafkaAPISpec: &v1alpha2.KafkaAPISpec{
			Brokers: []string{seedBroker},
			SASL: &v1alpha2.KafkaSASL{
				Username: superuser,
				Password: v1alpha2.SecretKeyRef{
					Name: secretName,
					Key:  secretKey,
				},
				Mechanism: v1alpha2.SASLMechanismScramSHA256,
			},
		},
		adminAPISpec: &v1alpha2.AdminAPISpec{
			URLs: []string{adminEndpoint},
			SASL: &v1alpha2.AdminSASL{
				Username: superuser,
				Password: v1alpha2.SecretKeyRef{
					Name: secretName,
					Key:  secretKey,
				},
				Mechanism: v1alpha2.SASLMechanismScramSHA256,
			},
		},
	}
}
