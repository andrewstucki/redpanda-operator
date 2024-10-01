// Copyright 2024 Redpanda Data, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.md
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0

// Package redpanda reconciles resources that comes from Redpanda dictionary like Topic, ACL and more.
package redpanda

import (
	"context"
	"time"

	redpandav1alpha2ac "github.com/redpanda-data/redpanda-operator/operator/api/applyconfiguration/redpanda/v1alpha2"
	redpandav1alpha2 "github.com/redpanda-data/redpanda-operator/operator/api/redpanda/v1alpha2"
	internalclient "github.com/redpanda-data/redpanda-operator/operator/pkg/client"
	"github.com/redpanda-data/redpanda-operator/operator/pkg/client/kubernetes"
	"github.com/redpanda-data/redpanda-operator/operator/pkg/utils"
	"github.com/twmb/franz-go/pkg/kgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TopicReconciler reconciles a Topic object
type TopicReconciler struct {
	// extraOptions can be overridden in tests
	// to change the way the underlying clients
	// function, i.e. setting low timeouts
	extraOptions []kgo.Opt
}

//+kubebuilder:rbac:groups=cluster.redpanda.com,namespace=default,resources=topics,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=cluster.redpanda.com,namespace=default,resources=topics/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cluster.redpanda.com,namespace=default,resources=topics/finalizers,verbs=update

// For cluster scoped operator

//+kubebuilder:rbac:groups=cluster.redpanda.com,resources=topics,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=cluster.redpanda.com,resources=topics/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cluster.redpanda.com,resources=topics/finalizers,verbs=update

func (r *TopicReconciler) FinalizerPatch(request ResourceRequest[*redpandav1alpha2.Topic]) client.Patch {
	topic := request.object
	config := redpandav1alpha2ac.Topic(topic.Name, topic.Namespace)
	return kubernetes.ApplyPatch(config.WithFinalizers(FinalizerKey))
}

func (r *TopicReconciler) SyncResource(ctx context.Context, request ResourceRequest[*redpandav1alpha2.Topic]) (client.Patch, error) {
	topic := request.object
	createPatch := func(err error) (client.Patch, error) {
		var syncCondition metav1.Condition
		config := redpandav1alpha2ac.Topic(topic.Name, topic.Namespace)

		if err != nil {
			syncCondition, err = handleResourceSyncErrors(err)
		} else {
			syncCondition = redpandav1alpha2.ResourceSyncedCondition(topic.Name)
		}

		return kubernetes.ApplyPatch(config.WithStatus(redpandav1alpha2ac.TopicStatus().
			WithObservedGeneration(topic.Generation).
			WithConditions(utils.StatusConditionConfigs(topic.Status.Conditions, topic.Generation, []metav1.Condition{
				syncCondition,
			})...))), err
	}

	syncer, err := request.factory.Topics(ctx, topic, r.extraOptions...)
	if err != nil {
		return createPatch(err)
	}

	return createPatch(syncer.Sync(ctx, topic))
}

func (r *TopicReconciler) DeleteResource(ctx context.Context, request ResourceRequest[*redpandav1alpha2.Topic]) error {
	syncer, err := request.factory.Topics(ctx, request.object, r.extraOptions...)
	if err != nil {
		return err
	}
	return syncer.Delete(ctx, request.object)
}

func SetupTopicController(ctx context.Context, mgr ctrl.Manager) error {
	c := mgr.GetClient()
	config := mgr.GetConfig()
	factory := internalclient.NewFactory(config, c)
	controller := NewResourceController(c, factory, &TopicReconciler{}, "TopicReconciler")

	enqueueTopic, err := registerClusterSourceIndex(ctx, mgr, "topic", &redpandav1alpha2.Topic{}, &redpandav1alpha2.TopicList{})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&redpandav1alpha2.Topic{}).
		Watches(&redpandav1alpha2.Redpanda{}, enqueueTopic).
		// Every 5 minutes try and check to make sure no manual modifications
		// happened on the resource synced to the cluster and attempt to correct
		// any drift.
		Complete(controller.PeriodicallyReconcile(5 * time.Minute))
}
