package steps

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/redpanda-data/common-go/rpadmin"
	"github.com/redpanda-data/helm-charts/pkg/kube"
	"github.com/redpanda-data/redpanda-operator/acceptance/framework"
	redpandav1alpha2 "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha2"
	"github.com/redpanda-data/redpanda-operator/src/go/k8s/pkg/client"
	"github.com/redpanda-data/redpanda-operator/src/go/k8s/pkg/client/acls"
	"github.com/redpanda-data/redpanda-operator/src/go/k8s/pkg/client/users"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

type clusterClients struct {
	Kafka         *kgo.Client
	RedpandaAdmin *rpadmin.AdminAPI
	Users         *users.Client
	ACLs          *acls.Syncer
}

func clientsForCluster(t framework.TestingT, ctx context.Context, cluster string) *clusterClients {
	// we construct a fake user to grab all of the clients for the cluster
	referencer := &redpandav1alpha2.User{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.Namespace(),
		},
		Spec: redpandav1alpha2.UserSpec{
			ClusterSource: &redpandav1alpha2.ClusterSource{
				ClusterRef: &redpandav1alpha2.ClusterRef{
					Name: cluster,
				},
			},
		},
	}

	factory := client.NewFactory(t.RestConfig(), t).WithDialer(kube.NewPodDialer(t.RestConfig()).DialContext)

	kafka, err := factory.KafkaClient(ctx, referencer)
	require.NoError(t, err)

	redpanda, err := factory.RedpandaAdminClient(ctx, referencer)
	require.NoError(t, err)

	users, err := factory.Users(ctx, referencer)
	require.NoError(t, err)

	syncer, err := factory.ACLs(ctx, referencer)
	require.NoError(t, err)

	return &clusterClients{
		Kafka:         kafka,
		RedpandaAdmin: redpanda,
		Users:         users,
		ACLs:          syncer,
	}
}

func usersFromTable(t framework.TestingT, cluster string, table *godog.Table) []*redpandav1alpha2.User {
	var users []*redpandav1alpha2.User

	for i, row := range table.Rows {
		// skip the header row:
		// | name | password | mechanism | acls |
		if i == 0 {
			continue
		}
		name, password, mechanism, acls := row.Cells[0].Value, row.Cells[1].Value, row.Cells[2].Value, row.Cells[3].Value
		name, password, mechanism, acls = strings.TrimSpace(name), strings.TrimSpace(password), strings.TrimSpace(mechanism), strings.TrimSpace(acls)
		user := &redpandav1alpha2.User{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: t.Namespace(),
				Name:      name,
			},
			Spec: redpandav1alpha2.UserSpec{
				ClusterSource: &redpandav1alpha2.ClusterSource{
					ClusterRef: &redpandav1alpha2.ClusterRef{
						Name: cluster,
					},
				},
			},
		}
		if mechanism != "" || password != "" {
			user.Spec.Authentication = &redpandav1alpha2.UserAuthenticationSpec{
				Type: ptr.To(redpandav1alpha2.SASLMechanism(mechanism)),
				Password: redpandav1alpha2.Password{
					Value: password,
					ValueFrom: &redpandav1alpha2.PasswordSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: name + "-password",
							},
						},
					},
				},
			}
		}
		if acls != "" {
			user.Spec.Authorization = &redpandav1alpha2.UserAuthorizationSpec{}
			require.NoError(t, json.Unmarshal([]byte(acls), &user.Spec.Authorization.ACLs))
		}

		users = append(users, user)
	}

	return users
}

func iCreateCRDbasedUsers(ctx context.Context, cluster string, users *godog.Table) error {
	t := framework.T(ctx)

	for _, user := range usersFromTable(t, cluster, users) {
		t.Logf("Creating user %q", user.Name)
		require.NoError(t, t.Create(ctx, user))
	}

	return nil
}

func shouldExistAndBeAbleToAuthenticateToTheCluster(ctx context.Context, user, cluster string) error {
	t := framework.T(ctx)

	clients := clientsForCluster(t, ctx, cluster)

	t.Logf("Checking for user %q in cluster %q", user, cluster)
	require.Eventually(t, func() bool {
		users, err := clients.Users.List(ctx)
		require.NoError(t, err)

		return slices.Contains(users, user)
	}, 10*time.Second, 1*time.Second)
	t.Logf("Found user %q in cluster %q", user, cluster)

	// TODO: add authentication check

	return nil
}

func thereIsNoUser(ctx context.Context, user, cluster string) error {
	t := framework.T(ctx)

	clients := clientsForCluster(t, ctx, cluster)

	t.Logf("Checking that user %q does not exist in cluster %q", user, cluster)
	require.Eventually(t, func() bool {
		users, err := clients.Users.List(ctx)
		require.NoError(t, err)

		return !slices.Contains(users, user)
	}, 10*time.Second, 1*time.Second)
	t.Logf("Found no user %q in cluster %q", user, cluster)

	return nil
}
