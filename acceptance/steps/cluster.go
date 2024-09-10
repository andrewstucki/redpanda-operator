package steps

import (
	"context"
	"fmt"

	"github.com/redpanda-data/redpanda-operator/acceptance/framework"
	redpandav1alpha2 "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha2"
	"github.com/stretchr/testify/require"
)

func checkClusterAvailability(ctx context.Context, clusterName string) {
	t := framework.T(ctx)

	var cluster redpandav1alpha2.Redpanda
	require.NoError(t, t.Get(ctx, t.ResourceKey(clusterName), &cluster))
	fmt.Println(cluster.Status.Conditions)
}
