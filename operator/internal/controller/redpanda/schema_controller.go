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
	"errors"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	redpandav1alpha2ac "github.com/redpanda-data/redpanda-operator/operator/api/applyconfiguration/redpanda/v1alpha2"
	redpandav1alpha2 "github.com/redpanda-data/redpanda-operator/operator/api/redpanda/v1alpha2"
	internalclient "github.com/redpanda-data/redpanda-operator/operator/pkg/client"
	"github.com/redpanda-data/redpanda-operator/operator/pkg/client/kubernetes"
	"github.com/redpanda-data/redpanda-operator/operator/pkg/utils"
	"github.com/twmb/franz-go/pkg/sr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SchemaReconciler reconciles a schema object
type SchemaReconciler struct {
	client.Client
	internalclient.ClientFactory
}

//+kubebuilder:rbac:groups=cluster.redpanda.com,namespace=default,resources=schemas,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=cluster.redpanda.com,namespace=default,resources=schemas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cluster.redpanda.com,namespace=default,resources=schemas/finalizers,verbs=update

// For cluster scoped operator

//+kubebuilder:rbac:groups=cluster.redpanda.com,resources=schemas,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=cluster.redpanda.com,resources=schemas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cluster.redpanda.com,resources=schemas/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *SchemaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx).WithName("SchemaReconciler.Reconcile")
	l.V(1).Info("Starting reconcile loop")
	start := time.Now()
	defer func() {
		l.V(1).Info("Finished reconciling", "elapsed", time.Since(start))
	}()

	schema := &redpandav1alpha2.Schema{}
	if err := r.Get(ctx, req.NamespacedName, schema); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !schema.DeletionTimestamp.IsZero() {
		if err := r.deleteSchema(ctx, schema); err != nil {
			return ctrl.Result{}, err
		}
		if controllerutil.RemoveFinalizer(schema, FinalizerKey) {
			return ctrl.Result{}, r.Update(ctx, schema)
		}
		return ctrl.Result{}, nil
	}

	config := redpandav1alpha2ac.Schema(schema.Name, schema.Namespace)

	if !controllerutil.ContainsFinalizer(schema, FinalizerKey) {
		patch := kubernetes.ApplyPatch(config.WithFinalizers(FinalizerKey))
		if err := r.Patch(ctx, schema, patch, client.ForceOwnership, fieldOwner); err != nil {
			return ctrl.Result{}, err
		}
	}

	syncCondition, versions, err := r.syncSchema(ctx, schema)
	patch := kubernetes.ApplyPatch(config.WithStatus(redpandav1alpha2ac.SchemaStatus().
		WithObservedGeneration(schema.Generation).
		WithVersions(versions...).
		WithConditions(utils.StatusConditionConfigs(schema.Status.Conditions, schema.Generation, []metav1.Condition{
			syncCondition,
		})...)))
	syncError := r.Status().Patch(ctx, schema, patch, client.ForceOwnership, fieldOwner)

	return ctrl.Result{}, errors.Join(err, syncError)
}

func (r *SchemaReconciler) syncSchema(ctx context.Context, schema *redpandav1alpha2.Schema) (condition metav1.Condition, versions []int, err error) {
	handleErrors := func(err error) (metav1.Condition, []int, error) {
		// If we have a known terminal error, just set the sync condition and don't re-run reconciliation.
		if internalclient.IsInvalidClusterError(err) {
			return redpandav1alpha2.SchemaNotSyncedCondition(redpandav1alpha2.SchemaConditionReasonClusterRefInvalid, err), schema.Status.Versions, nil
		}
		if internalclient.IsConfigurationError(err) {
			return redpandav1alpha2.UserNotSyncedCondition(redpandav1alpha2.SchemaConditionReasonConfigurationInvalid, err), schema.Status.Versions, nil
		}
		if internalclient.IsTerminalClientError(err) {
			return redpandav1alpha2.UserNotSyncedCondition(redpandav1alpha2.SchemaConditionReasonTerminalClientError, err), schema.Status.Versions, nil
		}

		// otherwise, set a generic unexpected error and return an error so we can re-reconcile.
		return redpandav1alpha2.SchemaNotSyncedCondition(redpandav1alpha2.SchemaConditionReasonUnexpectedError, err), schema.Status.Versions, err
	}

	client, err := r.ClientFactory.SchemaRegistryClient(ctx, schema)
	if err != nil {
		return handleErrors(err)
	}

	toCreate := sr.Schema{Schema: schema.Spec.Text}

	dirty := true
	if len(schema.Status.Versions) > 0 {
		subjectSchema, err := client.SchemaByVersion(ctx, schema.Name, -1)
		if err != nil {
			return handleErrors(err)
		}
		dirty = !schema.Matches(&subjectSchema)
	}

	versions = schema.Status.Versions
	if dirty {
		subjectSchema, err := client.CreateSchema(ctx, schema.Name, toCreate)
		if err != nil {
			return handleErrors(err)
		}
		versions = append(versions, subjectSchema.Version)
	}

	return redpandav1alpha2.SchemaSyncedCondition(schema.Name), versions, nil
}

func (r *SchemaReconciler) deleteSchema(ctx context.Context, schema *redpandav1alpha2.Schema) error {
	l := log.FromContext(ctx).WithName("SchemaReconciler.Reconcile")
	l.V(2).Info("Deleting schema data from cluster")

	ignoreAllConnectionErrors := func(err error) error {
		// If we have known errors where we're unable to actually establish
		// a connection to the cluster due to say, invalid connection parameters
		// we're going to just skip the cleanup phase since we likely won't be
		// able to clean ourselves up anyway.
		if internalclient.IsTerminalClientError(err) ||
			internalclient.IsConfigurationError(err) ||
			internalclient.IsInvalidClusterError(err) {
			// We use Info rather than Error here because we don't want
			// to ignore the verbosity settings. This is really only for
			// debugging purposes.
			l.V(2).Info("Ignoring non-retryable client error", "error", err)
			return nil
		}
		return err
	}

	client, err := r.ClientFactory.SchemaRegistryClient(ctx, schema)
	if err != nil {
		return ignoreAllConnectionErrors(err)
	}

	if _, err := client.DeleteSubject(ctx, schema.Name, sr.SoftDelete); err != nil {
		return ignoreAllConnectionErrors(err)
	}
	if _, err := client.DeleteSubject(ctx, schema.Name, sr.HardDelete); err != nil {
		return ignoreAllConnectionErrors(err)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SchemaReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	if err := registerSchemaClusterIndex(ctx, mgr); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&redpandav1alpha2.Schema{}).
		Watches(&redpandav1alpha2.Redpanda{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			requests, err := schemasForCluster(ctx, r, client.ObjectKeyFromObject(o))
			if err != nil {
				mgr.GetLogger().V(1).Info("possibly skipping schema reconciliation due to failure to fetch schemas associated with cluster", "error", err)
				return nil
			}
			return requests
		})).
		Complete(r)
}
