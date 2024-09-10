package steps

import (
	"context"
	"time"

	"github.com/redpanda-data/redpanda-operator/acceptance/framework"
	redpandav1alpha2 "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha2"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func checkClusterAvailability(ctx context.Context, clusterName string) {
	t := framework.T(ctx)

	require.Eventually(t, func() bool {
		var cluster redpandav1alpha2.Redpanda
		require.NoError(t, t.Get(ctx, t.ResourceKey(clusterName), &cluster))
		return t.HasCondition(metav1.Condition{
			Type:   "Ready",
			Status: metav1.ConditionTrue,
			Reason: "RedpandaClusterDeployed",
		}, cluster.Status.Conditions)
	}, 5*time.Minute, 5*time.Second)
}
