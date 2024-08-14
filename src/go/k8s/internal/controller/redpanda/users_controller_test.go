package redpanda

import (
	"context"
	"slices"
	"testing"
	"time"

	gofakeit "github.com/brianvoe/gofakeit/v7"
	redpandav1alpha2 "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha2"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// TODO: right now we only set a bare minimum of the operation/type combinations
// that we support, we should do more
type testACL struct {
	Type        string `fake:"{randomstring:[topic,group,transactionalId]}"`
	Name        string `fake:"{word}"`
	PatternType string `fake:"{randomstring:[literal]}"`
	Operation   string `fake:"{randomstring:[Read,Write,Delete]}"`
}

type testUser struct {
	*redpandav1alpha2.User `fake:"skip"`

	Username   string    `fake:"{uuid}"`
	Password   string    `fake:"{uuid}"`
	SecretName string    `fake:"{uuid}"`
	SecretKey  string    `fake:"{uuid}"`
	Mechanism  string    `fake:"{randomstring:[scram-sha-256,scram-sha-512]}"`
	ACLs       []testACL `fakesize:"1,10"`
}

func (t *testUser) toCRD(cluster *testCluster) *redpandav1alpha2.User {
	rules := []redpandav1alpha2.ACLRule{}
	for _, acl := range t.ACLs {
		rules = append(rules, redpandav1alpha2.ACLRule{
			Type: "allow",
			Resource: redpandav1alpha2.ACLResourceSpec{
				Type:        acl.Type,
				Name:        acl.Name,
				PatternType: acl.PatternType,
			},
			Operations: []string{acl.Operation},
		})
	}

	return &redpandav1alpha2.User{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.Username,
			Namespace: cluster.namespace,
		},
		Spec: redpandav1alpha2.UserSpec{
			KafkaAPISpec: cluster.kafkaAPISpec,
			AdminAPISpec: cluster.adminAPISpec,
			Authentication: redpandav1alpha2.UserAuthenticationSpec{
				Type: t.Mechanism,
				Password: &redpandav1alpha2.Password{
					ValueFrom: redpandav1alpha2.PasswordSource{
						SecretKeyRef: redpandav1alpha2.SecretKeyRef{
							Name: t.SecretName,
							Key:  t.SecretKey,
						},
					},
				},
			},
			Authorization: redpandav1alpha2.UserAuthorizationSpec{
				ACLs: rules,
			},
		},
	}
}

func createTestUser(t *testing.T, ctx context.Context, cluster *testCluster) *testUser {
	u := new(testUser)
	require.NoError(t, gofakeit.Struct(u))
	u.User = u.toCRD(cluster)
	require.NoError(t, cluster.Create(ctx, u.User))

	return u
}

func TestReconcileUser(t *testing.T) { // nolint:funlen // These tests have clear subtests.
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	cluster := createTestCluster(t, ctx)

	t.Run("lifecycle", func(t *testing.T) {
		reconciler := NewUserController(cluster, cluster)
		user := createTestUser(t, ctx, cluster)

		adminClient, err := cluster.RedpandaAdminClient(ctx, user.User)
		require.NoError(t, err)

		userClient, err := cluster.UserClient(ctx, user.User)
		require.NoError(t, err)

		req := ctrl.Request{
			NamespacedName: client.ObjectKeyFromObject(user),
		}

		requireEventually := func(t *testing.T, assertion func() bool) {
			require.Eventually(t, func() bool {
				// reconcile and ensure we've not errored
				result, err := reconciler.Reconcile(ctx, req)
				require.NoError(t, err)
				require.False(t, result.Requeue)

				return assertion()
			}, 5*time.Second, 10*time.Millisecond)
		}

		requireUserExists := func(t *testing.T, exists bool) {
			requireEventually(t, func() bool {
				users, err := adminClient.ListUsers(ctx)
				if err != nil {
					return false
				}
				return slices.Contains(users, user.RedpandaName()) == exists
			})
		}

		requireACLLength := func(t *testing.T, expected []testACL) {
			requireEventually(t, func() bool {
				acls, err := userClient.ListACLs(ctx)
				if err != nil {
					return false
				}
				return len(acls) == len(expected)
			})
		}

		// creation and finalizer
		requireUserExists(t, true)
		cluster.FetchLatest(t, ctx, user.User)
		require.True(t, controllerutil.ContainsFinalizer(user.User, FinalizerKey))
		requireACLLength(t, user.ACLs)

		// update
		user.Spec.Authorization = redpandav1alpha2.UserAuthorizationSpec{}
		cluster.UpdateObject(t, ctx, user.User)
		requireUserExists(t, true)
		requireACLLength(t, nil)

		// delete
		cluster.DeleteObject(t, ctx, user.User)
		requireUserExists(t, false)
		requireACLLength(t, nil)
	})
}
