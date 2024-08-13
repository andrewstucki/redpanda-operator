package redpanda

import (
	"context"

	redpandav1alpha2 "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha2"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	userClusterIndex = "__user_referencing_cluster"
)

func userCluster(user *redpandav1alpha2.User) types.NamespacedName {
	return types.NamespacedName{Namespace: user.Namespace, Name: user.Spec.ClusterRef.Name}
}

func userClusterStatus(user *redpandav1alpha2.User) types.NamespacedName {
	return types.NamespacedName{Namespace: user.Namespace, Name: user.Status.ClusterRef.Name}
}

func registerUserClusterIndex(ctx context.Context, mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(ctx, &redpandav1alpha2.User{}, userClusterIndex, indexUserCluster)
}

func indexUserCluster(o client.Object) []string {
	user := o.(*redpandav1alpha2.User)

	clusters := []string{}
	if user.Spec.ClusterRef != nil {
		clusters = append(clusters, userCluster(user).String())
	}
	if user.Status.ClusterRef != nil {
		clusters = append(clusters, userClusterStatus(user).String())
	}

	return clusters
}

func usersForCluster(ctx context.Context, c client.Client, nn types.NamespacedName) ([]reconcile.Request, error) {
	childList := &redpandav1alpha2.UserList{}
	err := c.List(ctx, childList, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(userClusterIndex, nn.String()),
	})

	if err != nil {
		return nil, err
	}

	requests := []reconcile.Request{}
	for _, item := range childList.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: item.GetNamespace(),
				Name:      item.GetName(),
			},
		})
	}

	return requests, nil
}
