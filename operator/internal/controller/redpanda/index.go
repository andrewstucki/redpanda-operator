// Copyright 2024 Redpanda Data, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.md
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0

package redpanda

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	redpandav1alpha2 "github.com/redpanda-data/redpanda-operator/operator/api/redpanda/v1alpha2"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type clientList[T client.Object] interface {
	client.ObjectList
	GetItems() []T
}

type clusterIndex struct {
	object     client.Object
	indexingFn func(o client.Object) []string
}

var (
	userClusterIndex        = clusterReferenceIndexName("user")
	schemaClusterIndex      = clusterReferenceIndexName("schema")
	deploymentClusterIndex  = clusterReferenceIndexName("deployment")
	statefulsetClusterIndex = clusterReferenceIndexName("statefulset")

	clusterIndices = map[string]*clusterIndex{
		userClusterIndex: {
			object:     &redpandav1alpha2.User{},
			indexingFn: indexByClusterSource,
		},
		schemaClusterIndex: {
			object:     &redpandav1alpha2.Schema{},
			indexingFn: indexByClusterSource,
		},
		deploymentClusterIndex: {
			object:     &appsv1.Deployment{},
			indexingFn: indexHelmManagedObjectCluster,
		},
		statefulsetClusterIndex: {
			object:     &appsv1.StatefulSet{},
			indexingFn: indexHelmManagedObjectCluster,
		},
	}
)

func clusterReferenceIndexName(name string) string {
	return fmt.Sprintf("__%s_referencing_cluster", name)
}

func registerClusterIndex(ctx context.Context, mgr ctrl.Manager, name string) error {
	indexName := clusterReferenceIndexName(name)

	index, ok := clusterIndices[indexName]
	if !ok {
		return fmt.Errorf("invalid cluster index: %q", name)
	}

	return mgr.GetFieldIndexer().IndexField(ctx, index.object, indexName, index.indexingFn)
}

func indexByClusterSource(o client.Object) []string {
	clusterReferencingObject := o.(redpandav1alpha2.ClusterReferencingObject)
	source := clusterReferencingObject.GetClusterSource()

	clusters := []string{}
	if source != nil && source.ClusterRef != nil {
		cluster := types.NamespacedName{Namespace: clusterReferencingObject.GetNamespace(), Name: source.ClusterRef.Name}
		clusters = append(clusters, cluster.String())
	}

	return clusters
}

func sourceClusters[T client.Object, U clientList[T]](ctx context.Context, c client.Client, list U, name string, nn types.NamespacedName) ([]reconcile.Request, error) {
	err := c.List(ctx, list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(clusterReferenceIndexName(name), nn.String()),
	})
	if err != nil {
		return nil, err
	}

	requests := []reconcile.Request{}
	for _, item := range list.GetItems() { //nolint:gocritic // this is necessary
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: item.GetNamespace(),
				Name:      item.GetName(),
			},
		})
	}

	return requests, nil
}

func enqueueFromSourceCluster[T client.Object, U clientList[T]](mgr ctrl.Manager, name string, l U) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
		list := reflect.New(reflect.TypeOf(l).Elem()).Interface().(U)
		requests, err := sourceClusters(ctx, mgr.GetClient(), list, "schema", client.ObjectKeyFromObject(o))
		if err != nil {
			mgr.GetLogger().V(1).Info(fmt.Sprintf("possibly skipping %s reconciliation due to failure to fetch %s associated with cluster", name, name), "error", err)
			return nil
		}
		return requests
	})
}

func clusterForHelmManagedObject(o client.Object) (types.NamespacedName, bool) {
	labels := o.GetLabels()
	clusterName := labels["app.kubernetes.io/instance"]
	if clusterName == "" {
		return types.NamespacedName{}, false
	}

	role := labels["app.kubernetes.io/name"]
	if !slices.Contains([]string{"redpanda", "console"}, role) {
		return types.NamespacedName{}, false
	}

	if _, ok := labels["batch.kubernetes.io/job-name"]; ok {
		return types.NamespacedName{}, false
	}

	return types.NamespacedName{
		Namespace: o.GetNamespace(),
		Name:      clusterName,
	}, true
}

func indexHelmManagedObjectCluster(o client.Object) []string {
	nn, found := clusterForHelmManagedObject(o)
	if !found {
		return nil
	}
	role := o.GetLabels()["app.kubernetes.io/name"]

	// we add two cache keys:
	// 1. namespace/name of the cluster
	// 2. namespace/name/role to identify if this is a console or redpanda component

	baseID := nn.String()
	roleID := baseID + "/" + role

	return []string{baseID, roleID}
}

func consoleDeploymentsForCluster(ctx context.Context, c client.Client, cluster *redpandav1alpha2.Redpanda) ([]*appsv1.Deployment, error) {
	key := client.ObjectKeyFromObject(cluster).String() + "/console"

	deploymentList := &appsv1.DeploymentList{}
	err := c.List(ctx, deploymentList, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(deploymentClusterIndex, key),
	})
	if err != nil {
		return nil, err
	}

	return mapFn(ptr.To, deploymentList.Items), nil
}

func redpandaStatefulSetsForCluster(ctx context.Context, c client.Client, cluster *redpandav1alpha2.Redpanda) ([]*appsv1.StatefulSet, error) {
	key := client.ObjectKeyFromObject(cluster).String() + "/redpanda"

	ssList := &appsv1.StatefulSetList{}
	err := c.List(ctx, ssList, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(statefulsetClusterIndex, key),
	})
	if err != nil {
		return nil, err
	}

	return mapFn(ptr.To, ssList.Items), nil
}

func mapFn[T any, U any](fn func(T) U, a []T) []U {
	s := make([]U, len(a))
	for i := 0; i < len(a); i++ {
		s[i] = fn(a[i])
	}
	return s
}
