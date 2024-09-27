package steps

import (
	"context"

	framework "github.com/redpanda-data/redpanda-operator/harpoon"
	redpandav1alpha2 "github.com/redpanda-data/redpanda-operator/operator/api/redpanda/v1alpha2"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func schemaIsSuccessfullySynced(ctx context.Context, t framework.TestingT, schema string) {
	var schemaObject redpandav1alpha2.Schema
	require.NoError(t, t.Get(ctx, t.ResourceKey(schema), &schemaObject))

	// make sure the resource is stable
	checkStableResource(ctx, t, &schemaObject)

	// make sure it's synchronized
	t.RequireCondition(metav1.Condition{
		Type:   redpandav1alpha2.ReadyCondition,
		Status: metav1.ConditionTrue,
		Reason: redpandav1alpha2.SucceededReason,
	}, schemaObject.Status.Conditions)
}

func thereIsNoSchema(ctx context.Context, schema, cluster string) {
	clientsForCluster(ctx, cluster).ExpectNoSchema(ctx, schema)
}

func iShouldBeAbleToCheckCompatibilityAgainst(ctx context.Context, t framework.TestingT, schema, cluster string) {
	clients := clientsForCluster(ctx, cluster)
	clients.ExpectSchema(ctx, schema)
}
