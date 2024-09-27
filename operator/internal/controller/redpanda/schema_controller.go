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

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	redpandav1alpha2ac "github.com/redpanda-data/redpanda-operator/operator/api/applyconfiguration/redpanda/v1alpha2"
	redpandav1alpha2 "github.com/redpanda-data/redpanda-operator/operator/api/redpanda/v1alpha2"
	internalclient "github.com/redpanda-data/redpanda-operator/operator/pkg/client"
	"github.com/redpanda-data/redpanda-operator/operator/pkg/client/kubernetes"
	"github.com/redpanda-data/redpanda-operator/operator/pkg/utils"
	"github.com/twmb/franz-go/pkg/sr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:rbac:groups=cluster.redpanda.com,namespace=default,resources=schemas,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=cluster.redpanda.com,namespace=default,resources=schemas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cluster.redpanda.com,namespace=default,resources=schemas/finalizers,verbs=update

// For cluster scoped operator

//+kubebuilder:rbac:groups=cluster.redpanda.com,resources=schemas,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=cluster.redpanda.com,resources=schemas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cluster.redpanda.com,resources=schemas/finalizers,verbs=update

// SchemaReconciler reconciles a schema object
type SchemaReconciler struct{}

func (r *SchemaReconciler) FinalizerPatch(request ResourceRequest[*redpandav1alpha2.Schema]) client.Patch {
	schema := request.object
	config := redpandav1alpha2ac.Schema(schema.Name, schema.Namespace)
	return kubernetes.ApplyPatch(config.WithFinalizers(FinalizerKey))
}

func (r *SchemaReconciler) SyncResource(ctx context.Context, request ResourceRequest[*redpandav1alpha2.Schema]) (client.Patch, error) {
	schema := request.object
	createPatch := func(err error, versions []int) (client.Patch, error) {
		var syncCondition metav1.Condition
		config := redpandav1alpha2ac.Schema(schema.Name, schema.Namespace)

		if err != nil {
			syncCondition, err = handleResourceSyncErrors(err)
		} else {
			syncCondition = redpandav1alpha2.SchemaSyncedCondition(schema.Name)
		}

		return kubernetes.ApplyPatch(config.WithStatus(redpandav1alpha2ac.SchemaStatus().
			WithObservedGeneration(schema.Generation).
			WithVersions(versions...).
			WithConditions(utils.StatusConditionConfigs(schema.Status.Conditions, schema.Generation, []metav1.Condition{
				syncCondition,
			})...))), err
	}

	versions := schema.Status.Versions
	client, err := request.factory.SchemaRegistryClient(ctx, schema)
	if err != nil {
		return createPatch(err, versions)
	}

	dirty := true
	if len(schema.Status.Versions) > 0 {
		subjectSchema, err := client.SchemaByVersion(ctx, schema.Name, -1)
		if err != nil {
			return createPatch(err, versions)
		}
		dirty = !schema.Matches(&subjectSchema)
	}

	versions = schema.Status.Versions
	if dirty {
		subjectSchema, err := client.CreateSchema(ctx, schema.Name, schema.ToKafka())
		if err != nil {
			return createPatch(err, versions)
		}
		versions = append(versions, subjectSchema.Version)
	}

	return createPatch(nil, versions)
}

func (r *SchemaReconciler) DeleteResource(ctx context.Context, request ResourceRequest[*redpandav1alpha2.Schema]) error {
	schema := request.object

	client, err := request.factory.SchemaRegistryClient(ctx, schema)
	if err != nil {
		return ignoreAllConnectionErrors(request.logger, err)
	}

	if _, err := client.DeleteSubject(ctx, schema.Name, sr.SoftDelete); err != nil {
		return ignoreAllConnectionErrors(request.logger, err)
	}
	if _, err := client.DeleteSubject(ctx, schema.Name, sr.HardDelete); err != nil {
		return ignoreAllConnectionErrors(request.logger, err)
	}

	return nil
}

func SetupSchemaController(ctx context.Context, mgr ctrl.Manager) error {
	c := mgr.GetClient()
	config := mgr.GetConfig()
	factory := internalclient.NewFactory(config, c)
	controller := NewResourceController(c, factory, &SchemaReconciler{}, "SchemaReconciler")

	if err := registerClusterIndex(ctx, mgr, "schema"); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&redpandav1alpha2.Schema{}).
		Watches(&redpandav1alpha2.Redpanda{}, enqueueFromSourceCluster(mgr, "schema", &redpandav1alpha2.SchemaList{})).
		// Every 5 minutes try and check to make sure no manual modifications
		// happened on the resource synced to the cluster and attempt to correct
		// any drift.
		Complete(controller.PeriodicallyReconcile(5 * time.Minute))
}
