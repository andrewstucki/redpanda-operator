package steps

import (
	"context"

	"github.com/cucumber/godog"
	framework "github.com/redpanda-data/redpanda-operator/harpoon"
	redpandav1alpha2 "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha2"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func iCreateCRDbasedUsers(ctx context.Context, cluster string, users *godog.Table) error {
	t := framework.T(ctx)

	for _, user := range usersFromTable(t, cluster, users) {
		t.Logf("Creating user %q", user.Name)
		require.NoError(t, t.Create(ctx, user))

		// make sure the resource is stable
		checkStableResource(ctx, t, user)

		// make sure it's synchronized
		t.RequireCondition(metav1.Condition{
			Type:   redpandav1alpha2.UserConditionTypeSynced,
			Status: metav1.ConditionTrue,
			Reason: redpandav1alpha2.UserConditionReasonSynced,
		}, user.Status.Conditions)
	}

	return nil
}

func shouldExistAndBeAbleToAuthenticateToTheCluster(ctx context.Context, user, cluster string) error {
	t := framework.T(ctx)

	clientsForCluster(t, ctx, cluster, nil).ExpectUser(ctx, user)

	// Now we do the same check, but authenticated as the user

	var userObject redpandav1alpha2.User
	require.NoError(t, t.Get(ctx, t.ResourceKey(user), &userObject))

	clientsForCluster(t, ctx, cluster, &userObject).ExpectUser(ctx, user)

	return nil
}

func thereIsNoUser(ctx context.Context, user, cluster string) error {
	t := framework.T(ctx)

	clientsForCluster(t, ctx, cluster, nil).ExpectNoUser(ctx, user)

	return nil
}
