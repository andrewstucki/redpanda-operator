package redpanda

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha2"
	"github.com/redpanda-data/redpanda-operator/src/go/k8s/internal/controller/redpanda/users"
	"github.com/redpanda-data/redpanda-operator/src/go/k8s/internal/testutils"
	"github.com/redpanda-data/redpanda-operator/src/go/k8s/internal/util/kafka"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/redpanda"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestReconcileUser(t *testing.T) { // nolint:funlen // These tests have clear subtests.
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	testEnv := testutils.RedpandaTestEnv{}
	cfg, err := testEnv.StartRedpandaTestEnv(false)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	err = v1alpha2.AddToScheme(scheme.Scheme)
	require.NoError(t, err)

	c, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	require.NoError(t, err)
	require.NotNil(t, c)

	var seedBroker string
	var adminEndpoint string

	factory, err := kafka.NewClientFactory(cfg)
	require.NoError(t, err)

	testNamespace := "default"
	{
		container, err := redpanda.Run(
			ctx,
			"docker.redpanda.com/redpandadata/redpanda:v24.2.1",
			redpanda.WithEnableSASL(),
			redpanda.WithNewServiceAccount("superuser", "test"),
			redpanda.WithSuperusers("superuser"),
			redpanda.WithEnableSchemaRegistryHTTPBasicAuth(),
		)
		require.NoError(t, err)
		logs, err := container.Logs(ctx)
		require.NoError(t, err)

		go func() {
			io.Copy(os.Stdout, logs)
		}()

		t.Cleanup(func() {
			ctxCleanup, cancelCleanup := context.WithTimeout(context.Background(), time.Minute*2)
			defer cancelCleanup()
			if err = container.Terminate(ctxCleanup); err != nil {
				t.Fatalf("failed to terminate container: %s", err)
			}
		})

		seedBroker, err = container.KafkaSeedBroker(ctx)
		require.NoError(t, err)
		adminEndpoint, err = container.AdminAPIAddress(ctx)
		require.NoError(t, err)
	}

	tr := NewUserController(c, factory)

	require.NoError(t, factory.Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret",
			Namespace: testNamespace,
		},
		StringData: map[string]string{
			"password": "test",
		},
	}))

	t.Run("create_user", func(t *testing.T) {
		username := "create-test-user"

		createUser := v1alpha2.User{
			ObjectMeta: metav1.ObjectMeta{
				Name:      username,
				Namespace: testNamespace,
			},
			Spec: v1alpha2.UserSpec{
				KafkaAPISpec: &v1alpha2.KafkaAPISpec{
					Brokers:   []string{seedBroker},
					AdminURLs: []string{adminEndpoint},
					SASL: &v1alpha2.KafkaSASL{
						Username: "superuser",
						Password: v1alpha2.SecretKeyRef{
							Name: "secret",
							Key:  "password",
						},
						Mechanism: v1alpha2.SASLMechanismScramSHA256,
					},
				},
				Authentication: v1alpha2.UserAuthenticationSpec{
					Type: "scram-sha-512",
					Password: &v1alpha2.Password{
						ValueFrom: v1alpha2.PasswordSource{
							SecretKeyRef: v1alpha2.SecretKeyRef{
								Name: "user-password",
							},
						},
					},
				},
				Authorization: v1alpha2.UserAuthorizationSpec{
					ACLs: []v1alpha2.ACLRule{{
						Type: "allow",
						Resource: v1alpha2.ACLResourceSpec{
							Type: "topic",
							Name: "foo",
						},
						Operations: []string{"Write"},
					}},
				},
			},
		}

		err := c.Create(ctx, &createUser)
		require.NoError(t, err)

		req := ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      username,
				Namespace: testNamespace,
			},
		}

		// finalizer
		result, err := tr.Reconcile(ctx, req)
		require.NoError(t, err)
		require.False(t, result.Requeue)
		printUserInfo(t, "INITIAL", ctx, &createUser, factory)

		{
			// creation
			result, err = tr.Reconcile(ctx, req)
			require.NoError(t, err)
			require.False(t, result.Requeue)

			printUserInfo(t, "CREATE", ctx, &createUser, factory)
		}

		{
			// update
			require.NoError(t, factory.Get(ctx, types.NamespacedName{Namespace: testNamespace, Name: createUser.Name}, &createUser))
			createUser.Spec.Authorization = v1alpha2.UserAuthorizationSpec{}
			require.NoError(t, factory.Update(ctx, &createUser))

			result, err = tr.Reconcile(ctx, req)
			require.NoError(t, err)
			require.False(t, result.Requeue)

			printUserInfo(t, "UPDATE", ctx, &createUser, factory)
		}

		{
			// delete
			require.NoError(t, factory.Delete(ctx, &createUser))

			result, err = tr.Reconcile(ctx, req)
			require.NoError(t, err)
			require.False(t, result.Requeue)

			printUserInfo(t, "DELETE", ctx, &createUser, factory)
		}
	})
}

func printUserInfo(t *testing.T, step string, ctx context.Context, user *v1alpha2.User, factory *kafka.ClientFactory) {
	builder := users.NewClientBuilder(factory)

	client, err := factory.GetAdminClient(ctx, user.Namespace, user.Spec.KafkaAPISpec)
	require.NoError(t, err)

	userClient, err := builder.ForUser(ctx, user)
	require.NoError(t, err)

	users, err := client.ListUsers(ctx)
	require.NoError(t, err)

	userACLs, err := userClient.ListACLs(ctx)
	require.NoError(t, err)

	fmt.Printf("===========%s===========\n", step)
	fmt.Printf("Users: %v\n", users)
	fmt.Printf("User (%q) ACLs: %+v\n", user.RedpandaName(), userACLs.Resources)
	fmt.Printf("===========%s===========\n", strings.Repeat("=", len(step)))
}
