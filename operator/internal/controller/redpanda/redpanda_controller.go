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
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	helmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	helmv2beta2 "github.com/fluxcd/helm-controller/api/v2beta2"
	"github.com/fluxcd/pkg/apis/meta"
	"github.com/fluxcd/pkg/runtime/logger"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/go-logr/logr"
	redpandav1alpha2ac "github.com/redpanda-data/redpanda-operator/operator/api/applyconfiguration/redpanda/v1alpha2"
	redpandav1alpha2 "github.com/redpanda-data/redpanda-operator/operator/api/redpanda/v1alpha2"
	internalclient "github.com/redpanda-data/redpanda-operator/operator/pkg/client"
	"github.com/redpanda-data/redpanda-operator/operator/pkg/client/kubernetes"
	"github.com/redpanda-data/redpanda-operator/operator/pkg/collections"
	"github.com/redpanda-data/redpanda-operator/operator/pkg/functional"
	"github.com/redpanda-data/redpanda-operator/operator/pkg/resources"
	"github.com/redpanda-data/redpanda-operator/operator/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	managedPath = "/managed"

	revisionPath        = "/revision"
	componentLabelValue = "redpanda-statefulset"

	HelmChartConstraint = "5.9.3"

	NotManaged = "false"
)

// RedpandaReconciler reconciles a Redpanda object
type RedpandaReconciler struct {
	client.Client
	factory *internalclient.Factory
	Scheme  *runtime.Scheme
}

func SetupRedpandaReconciler(ctx context.Context, mgr ctrl.Manager) error {
	c := mgr.GetClient()
	config := mgr.GetConfig()
	scheme := mgr.GetScheme()
	factory := internalclient.NewFactory(config, c)

	return (&RedpandaReconciler{
		Client:  c,
		Scheme:  scheme,
		factory: factory,
	}).SetupWithManager(ctx, mgr)
}

// flux resources main resources
// +kubebuilder:rbac:groups=helm.toolkit.fluxcd.io,namespace=default,resources=helmreleases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=helm.toolkit.fluxcd.io,namespace=default,resources=helmreleases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=helm.toolkit.fluxcd.io,namespace=default,resources=helmreleases/finalizers,verbs=update
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,namespace=default,resources=helmcharts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,namespace=default,resources=helmcharts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,namespace=default,resources=helmcharts/finalizers,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,namespace=default,resources=helmrepositories,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,namespace=default,resources=helmrepositories/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,namespace=default,resources=helmrepositories/finalizers,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,namespace=default,resources=gitrepository,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,namespace=default,resources=gitrepository/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,namespace=default,resources=gitrepository/finalizers,verbs=get;create;update;patch;delete

// flux additional resources
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,namespace=default,resources=buckets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,namespace=default,resources=gitrepositories,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,namespace=default,resources=replicasets,verbs=get;list;watch;create;update;patch;delete

// any resource that Redpanda helm creates and flux controller needs to reconcile them
// +kubebuilder:rbac:groups="",namespace=default,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,namespace=default,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,namespace=default,resources=roles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,namespace=default,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,namespace=default,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,namespace=default,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,namespace=default,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,namespace=default,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete;
// +kubebuilder:rbac:groups=policy,namespace=default,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,namespace=default,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cert-manager.io,namespace=default,resources=certificates,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups=cert-manager.io,namespace=default,resources=issuers,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups="monitoring.coreos.com",namespace=default,resources=servicemonitors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,namespace=default,resources=ingresses,verbs=get;list;watch;create;update;patch;delete

// for the migration purposes to disable reconciliation of cluster and console custom resources
// +kubebuilder:rbac:groups=redpanda.vectorized.io,namespace=default,resources=clusters,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=redpanda.vectorized.io,namespace=default,resources=consoles,verbs=get;list;watch;update;patch

// redpanda resources
// +kubebuilder:rbac:groups=cluster.redpanda.com,namespace=default,resources=redpandas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.redpanda.com,namespace=default,resources=redpandas/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.redpanda.com,namespace=default,resources=redpandas/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,namespace=default,resources=events,verbs=create;patch

// SetupWithManager sets up the controller with the Manager.
func (r *RedpandaReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	if err := registerHelmReferencedIndex(ctx, mgr, "statefulset", &appsv1.StatefulSet{}); err != nil {
		return err
	}
	if err := registerHelmReferencedIndex(ctx, mgr, "deployment", &appsv1.Deployment{}); err != nil {
		return err
	}

	helmManagedComponentPredicate, err := predicate.LabelSelectorPredicate(
		metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{{
				Key:      "app.kubernetes.io/name", // look for only redpanda or console pods
				Operator: metav1.LabelSelectorOpIn,
				Values:   []string{"redpanda", "console"},
			}, {
				Key:      "app.kubernetes.io/instance", // make sure we have a cluster name
				Operator: metav1.LabelSelectorOpExists,
			}, {
				Key:      "batch.kubernetes.io/job-name", // filter out the job pods since they also have name=redpanda
				Operator: metav1.LabelSelectorOpDoesNotExist,
			}},
		},
	)
	if err != nil {
		return err
	}

	managedWatchOption := builder.WithPredicates(helmManagedComponentPredicate)

	return ctrl.NewControllerManagedBy(mgr).
		For(&redpandav1alpha2.Redpanda{}).
		Owns(&sourcev1.HelmRepository{}).
		Owns(&helmv2beta1.HelmRelease{}).
		Owns(&helmv2beta2.HelmRelease{}).
		Watches(&appsv1.StatefulSet{}, enqueueClusterFromHelmManagedObject(), managedWatchOption).
		Watches(&appsv1.Deployment{}, enqueueClusterFromHelmManagedObject(), managedWatchOption).
		Complete(r)
}

func (r *RedpandaReconciler) Reconcile(c context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx, done := context.WithCancel(c)
	defer done()

	start := time.Now()
	log := ctrl.LoggerFrom(ctx).WithName("RedpandaReconciler.Reconcile")

	defer func() {
		durationMsg := fmt.Sprintf("reconciliation finished in %s", time.Since(start).String())
		log.Info(durationMsg)
	}()

	log.Info("Starting reconcile loop")

	rp := &redpandav1alpha2.Redpanda{}
	if err := r.Client.Get(ctx, req.NamespacedName, rp); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Examine if the object is under deletion
	if !rp.ObjectMeta.DeletionTimestamp.IsZero() {
		// we don't need to delete helm releases due to ownership GC'ing coming into play
		if controllerutil.RemoveFinalizer(rp, FinalizerKey) {
			if err := r.Client.Update(ctx, rp); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !isRedpandaManaged(ctx, rp) {
		// if no longer managed by us, attempt to remove the finalizer
		if controllerutil.RemoveFinalizer(rp, FinalizerKey) {
			if err := r.Client.Update(ctx, rp); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(rp, FinalizerKey) {
		config := redpandav1alpha2ac.Redpanda(rp.Name, rp.Namespace)
		patch := kubernetes.ApplyPatch(config.WithFinalizers(FinalizerKey))
		if err := r.Patch(ctx, rp, patch, client.ForceOwnership, fieldOwner); err != nil {
			return ctrl.Result{}, err
		}
	}

	_, ok := rp.GetAnnotations()[resources.ManagedDecommissionAnnotation]
	if ok {
		return ctrl.Result{}, nil
	}

	patch, requeue, err := r.reconcile(ctx, rp)
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := r.Client.Status().Patch(ctx, rp, patch, client.ForceOwnership, fieldOwner); err != nil {
		return ctrl.Result{}, err
	}

	result := ctrl.Result{}
	if requeue {
		result.RequeueAfter = 10 * time.Second
	}

	return result, nil
}

func atLeast(version string) bool {
	if version == "" {
		return true
	}

	c, err := semver.NewConstraint(fmt.Sprintf(">= %s", HelmChartConstraint))
	if err != nil {
		// Handle constraint not being parsable.
		return false
	}

	v, err := semver.NewVersion(version)
	if err != nil {
		return false
	}

	return c.Check(v)
}

func (r *RedpandaReconciler) reconcile(ctx context.Context, rp *redpandav1alpha2.Redpanda) (client.Patch, bool, error) {
	log := ctrl.LoggerFrom(ctx)
	log.WithName("RedpandaReconciler.reconcile")

	// pull our deployments and stateful sets
	redpandaStatefulSets, err := redpandaStatefulSetsForCluster(ctx, r.Client, rp)
	if err != nil {
		return nil, false, err
	}
	consoleDeployments, err := consoleDeploymentsForCluster(ctx, r.Client, rp)
	if err != nil {
		return nil, false, err
	}

	// Check if HelmRepository exists or create it
	repo := r.createHelmRepositoryFromTemplate(rp)
	if err := r.Client.Patch(ctx, repo, client.Apply, client.ForceOwnership, fieldOwner); err != nil {
		return nil, false, fmt.Errorf("error patching HelmRepository: %w", err)
	}

	release, err := r.createHelmReleaseFromTemplate(rp)
	if err != nil {
		return nil, false, err
	}
	if err := r.Client.Patch(ctx, release, client.Apply, client.ForceOwnership, fieldOwner); err != nil {
		return nil, false, fmt.Errorf("error patching HelmRelease: %w", err)
	}

	isResourceReady := func(o client.Object, generation int64, conditions []metav1.Condition) bool {
		isCurrentGeneration := o.GetGeneration() == generation
		isStatusConditionReady := apimeta.IsStatusConditionTrue(conditions, meta.ReadyCondition) || apimeta.IsStatusConditionTrue(conditions, helmv2beta2.RemediatedCondition)
		return isCurrentGeneration && isStatusConditionReady
	}

	repoReady := isResourceReady(repo, repo.Status.ObservedGeneration, repo.Status.Conditions)
	releaseReady := isResourceReady(release, release.Status.ObservedGeneration, release.Status.Conditions)

	createPatch := func(condition metav1.Condition) client.Patch {
		patch := redpandav1alpha2ac.Redpanda(rp.Name, rp.Namespace).WithStatus(redpandav1alpha2ac.RedpandaStatus().
			WithObservedGeneration(rp.Generation).
			WithHelmReleaseReady(repoReady).
			WithHelmReleaseReady(releaseReady).
			WithHelmRepository(rp.GetHelmRepositoryName()).
			WithHelmRelease(rp.GetHelmReleaseName()).
			WithConditions(utils.StatusConditionConfigs(rp.Status.Conditions, rp.Generation, []metav1.Condition{
				condition,
			})...))
		return kubernetes.ApplyPatch(patch)
	}

	if !repoReady || !releaseReady {
		return createPatch(redpandav1alpha2.RedpandaNotReady("HelmChartNotReady", "Helm chart has not yet finished applying")), false, nil
	}

	if !rp.UseFlux() {
		// TODO (Rafal) Implement Redpanda helm chart templating with Redpanda Status Report
		// In the Redpanda.Status there will be only Conditions and Failures that would be used.

		if !atLeast(rp.Spec.ChartRef.ChartVersion) {
			// Do not error out to not requeue. User needs to first migrate helm release to at least 5.9.3 version
			return createPatch(redpandav1alpha2.RedpandaNotReady("ChartRefUnsupported", fmt.Sprintf("chart version needs to be at least %s. Currently it is %s", HelmChartConstraint, rp.Spec.ChartRef.ChartVersion))), false, nil
		}
	}

	for _, sts := range redpandaStatefulSets {
		decommission, err := r.needsDecommission(ctx, rp, sts)
		if err != nil {
			return nil, false, err
		}
		if decommission {
			requeue, err := r.reconcileDecommission(ctx, log, rp, sts)
			if err != nil {
				return nil, false, err
			}
			return createPatch(redpandav1alpha2.RedpandaNotReady("RedpandaPodsNotReady", "Cluster currently decommissioning dead nodes")), requeue, nil
		}
	}

	// check to make sure that our stateful set pods are all current
	if message, ready := checkStatefulSetStatus(redpandaStatefulSets); !ready {
		return createPatch(redpandav1alpha2.RedpandaNotReady("RedpandaPodsNotReady", message)), false, nil
	}

	// check to make sure that our deployment pods are all current
	if message, ready := checkDeploymentsStatus(consoleDeployments); !ready {
		return createPatch(redpandav1alpha2.RedpandaNotReady("ConsolePodsNotReady", message)), false, nil
	}

	return createPatch(redpandav1alpha2.RedpandaReady()), false, nil
}

func (r *RedpandaReconciler) createHelmReleaseFromTemplate(rp *redpandav1alpha2.Redpanda) (*helmv2beta2.HelmRelease, error) {
	values, err := rp.ValuesJSON()
	if err != nil {
		return nil, fmt.Errorf("could not parse clusterSpec to json: %w", err)
	}

	return &helmv2beta2.HelmRelease{
		TypeMeta: metav1.TypeMeta{
			Kind:       helmv2beta2.HelmReleaseKind,
			APIVersion: helmv2beta2.GroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            rp.GetHelmReleaseName(),
			Namespace:       rp.Namespace,
			OwnerReferences: []metav1.OwnerReference{rp.AsOwnerReference()},
		},
		Spec: helmv2beta2.HelmReleaseSpec{
			Suspend: !ptr.Deref(rp.Spec.ChartRef.UseFlux, true),
			Chart: helmv2beta2.HelmChartTemplate{
				Spec: helmv2beta2.HelmChartTemplateSpec{
					Chart:    "redpanda",
					Version:  rp.Spec.ChartRef.ChartVersion,
					Interval: &metav1.Duration{Duration: 1 * time.Minute},
					SourceRef: helmv2beta2.CrossNamespaceObjectReference{
						Kind:      "HelmRepository",
						Name:      rp.GetHelmRepositoryName(),
						Namespace: rp.Namespace,
					},
				},
			},
			Values:   values,
			Interval: metav1.Duration{Duration: 30 * time.Second},
			Timeout:  functional.Default(rp.Spec.ChartRef.Timeout, &metav1.Duration{Duration: 15 * time.Minute}),
			Upgrade: rp.Spec.ChartRef.MergeUpgrade(&helmv2beta2.Upgrade{
				// we skip waiting since relying on the Helm release process
				// to actually happen means that we block running any sort
				// of pending upgrades while we are attempting the upgrade job.
				DisableWait:        true,
				DisableWaitForJobs: true,
			}),
		},
	}, nil
}

func (r *RedpandaReconciler) createHelmRepositoryFromTemplate(rp *redpandav1alpha2.Redpanda) *sourcev1.HelmRepository {
	return &sourcev1.HelmRepository{
		TypeMeta: metav1.TypeMeta{
			Kind:       sourcev1.HelmRepositoryKind,
			APIVersion: sourcev1.GroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            rp.GetHelmRepositoryName(),
			Namespace:       rp.Namespace,
			OwnerReferences: []metav1.OwnerReference{rp.AsOwnerReference()},
		},
		Spec: sourcev1.HelmRepositorySpec{
			Suspend:  !ptr.Deref(rp.Spec.ChartRef.UseFlux, true),
			Interval: metav1.Duration{Duration: 30 * time.Second},
			URL:      redpandav1alpha2.RedpandaChartRepository,
		},
	}
}

func (r *RedpandaReconciler) needsDecommission(ctx context.Context, o *redpandav1alpha2.Redpanda, sts *appsv1.StatefulSet) (bool, error) {
	requestedReplicas := ptr.Deref(sts.Spec.Replicas, 0)

	adminAPI, err := r.factory.RedpandaAdminClient(ctx, o)
	if err != nil {
		return false, fmt.Errorf("error creating adminAPI: %w", err)
	}

	health, err := adminAPI.GetHealthOverview(ctx)
	if err != nil {
		return false, fmt.Errorf("could not make request to admin-api: %w", err)
	}

	if requestedReplicas == 0 || len(health.AllNodes) == 0 {
		return false, nil
	}

	return len(health.AllNodes) > int(requestedReplicas), nil
}

// reconcileDecommission performs decommission task after verifying that we should decommission the sts given
// 1. After requeue from decommission due to condition we have set, now we verify and perform tasks.
// 2. Retrieve sts information again, this time focusing on replicas and state
// 3. If we observe that we have not completed deletion, we requeue
// 4. We wait until the cluster is not ready, else requeue, this means that we have reached a steady state that we can proceed from
// 5. As in previous function, we build adminAPI client and get values files
// 6. Check if we have more nodes registered than requested, proceed since this is the first clue we need to decommission
// 7. We are in steady state, proceed if we have more or the same number of downed nodes then are in allNodes registered minus requested
// 8. For all the downed nodes, we get decommission-status, since we have waited for steady state we should be OK to do so
// 9. Any failures captured will force us to requeue and try again.
// 10. Attempt to delete the pvc and retain volumes if possible.
// 11. Finally, reset condition state to unknown if we have been successful so far.
//
//nolint:funlen // length looks good
func (r *RedpandaReconciler) reconcileDecommission(ctx context.Context, log logr.Logger, o *redpandav1alpha2.Redpanda, sts *appsv1.StatefulSet) (bool, error) {
	requestedReplicas := ptr.Deref(sts.Spec.Replicas, 0)

	statusReplicas := sts.Status.Replicas
	availableReplicas := sts.Status.AvailableReplicas

	// we have started decommission, but we want to requeue if we have not transitioned here. This should
	// avoid decommissioning the wrong node (broker) id
	if statusReplicas != requestedReplicas && sts.Status.UpdatedReplicas == 0 {
		log.Info("have not finished terminating and restarted largest ordinal, requeue here", "statusReplicas", statusReplicas, "availableReplicas", availableReplicas)
		return true, nil
	}

	// This helps avoid decommissioning nodes that are starting up where, say, a node has been removed
	// and you need to move it and start a new one
	if availableReplicas != 0 {
		log.Info("have not reached steady state yet, requeue here")
		return true, nil
	}

	adminAPI, err := r.factory.RedpandaAdminClient(ctx, o)
	if err != nil {
		return false, fmt.Errorf("error creating adminAPI: %w", err)
	}

	health, err := adminAPI.GetHealthOverview(ctx)
	if err != nil {
		return false, fmt.Errorf("could not make request to admin-api: %w", err)
	}

	// decommission looks like this:
	// 1) replicas = 2, and health: AllNodes:[0 1 2] NodesDown:[2]
	// will not heal on its own, we need to remove these downed nodes
	// 2) Downed node was replaced due to node being removed,

	if requestedReplicas == 0 || len(health.AllNodes) == 0 {
		return false, nil
	}

	var errList error
	// nolint:nestif // this is ok
	if len(health.AllNodes) > int(requestedReplicas) {
		// we are in decommission mode

		// first check if we have a controllerID before we perform the decommission, else requeue immediately
		// this happens when the controllerID node is being terminated, may show more than one node down at this point
		if health.ControllerID < 0 {
			log.Info("controllerID is not defined yet, we will requeue")
			return true, nil
		}

		nodesDownMap := collections.NewSet[int]()
		for _, node := range health.NodesDown {
			nodesDownMap.Add(node)
		}

		// perform decommission on down down-nodes but only if down nodes match count of all-nodes-replicas
		// the greater case takes care of the situation where we may also have additional ids here.
		if len(health.NodesDown) >= (len(health.AllNodes) - int(requestedReplicas)) {
			for podOrdinal := 0; podOrdinal < int(requestedReplicas); podOrdinal++ {
				singleNodeAdminAPI, err := adminAPI.ForBroker(ctx, podOrdinal)
				if err != nil {
					log.Error(err, "creating single node AdminAPI", "pod-ordinal", podOrdinal)
					return false, fmt.Errorf("creating single node AdminAPI for pod (%d): %w", podOrdinal, err)
				}
				nodeCfg, nodeErr := singleNodeAdminAPI.GetNodeConfig(ctx)
				if nodeErr != nil {
					log.Error(nodeErr, "getting node configuration", "pod-ordinal", podOrdinal)
					return false, fmt.Errorf("getting node configuration from pod (%d): %w", podOrdinal, nodeErr)
				}
				nodesDownMap.Delete(nodeCfg.NodeID)
			}

			for _, nodeID := range nodesDownMap.Values() {
				// Now we check the decommission status before continuing
				doDecommission := false
				status, decommStatusError := adminAPI.DecommissionBrokerStatus(ctx, nodeID)
				if decommStatusError != nil {
					log.Info("found for decommission status error", "decommStatusError", decommStatusError)
					// nolint:gocritic // no need for a switch, this is ok
					if strings.Contains(decommStatusError.Error(), "is not decommissioning") {
						doDecommission = true
					} else if strings.Contains(decommStatusError.Error(), "does not exists") {
						log.Info("nodeID does not exist, skipping", "nodeID", nodeID, "decommStatusError", decommStatusError)
					} else {
						errList = errors.Join(errList, fmt.Errorf("could get decommission status of broker: %w", decommStatusError))
					}
				}
				log.V(logger.DebugLevel).Info("decommission status", "status", status)

				if doDecommission {
					log.Info("all checks pass, attempting to decommission", "nodeID", nodeID)
					// we want a clear signal to avoid 400s here, the suspicion here is an invalid transitional state
					decomErr := adminAPI.DecommissionBroker(ctx, nodeID)
					if decomErr != nil && !strings.Contains(decomErr.Error(), "failed: Not Found") && !strings.Contains(decomErr.Error(), "failed: Bad Request") {
						errList = errors.Join(errList, fmt.Errorf("could not decommission broker: %w", decomErr))
					}
				}
			}
		}
	}

	// now we check pvcs
	if err = r.reconcilePVCs(log.WithName("DecommissionReconciler.reconcilePVCs"), ctx, o, sts); err != nil {
		errList = errors.Join(errList, fmt.Errorf("could not reconcile pvcs: %w", err))
	}

	if errList != nil {
		return false, fmt.Errorf("found errors %w", errList)
	}

	// now we need to
	patch := client.MergeFrom(sts.DeepCopy())
	// create condition here
	newCondition := appsv1.StatefulSetCondition{
		Type:               DecommissionCondition,
		Status:             corev1.ConditionUnknown,
		Message:            DecomConditionUnknownReasonMsg,
		Reason:             "Decommission completed",
		LastTransitionTime: metav1.Now(),
	}

	if updatedConditions, isUpdated := updateStatefulSetDecommissionConditions(&newCondition, sts.Status.Conditions); isUpdated {
		sts.Status.Conditions = updatedConditions
		if errPatch := r.Client.Status().Patch(ctx, sts, patch); errPatch != nil {
			return false, fmt.Errorf("unable to update sts status %q condition: %w", sts.Name, errPatch)
		}
	}

	return false, nil
}

func (r *RedpandaReconciler) reconcilePVCs(log logr.Logger, ctx context.Context, o *redpandav1alpha2.Redpanda, sts *appsv1.StatefulSet) error {
	if o.Spec.ClusterSpec.Storage == nil || o.Spec.ClusterSpec.Storage.PersistentVolume == nil || o.Spec.ClusterSpec.Storage.PersistentVolume.Enabled == nil {
		return nil
	}

	persistentStorageEnabled := *o.Spec.ClusterSpec.Storage.PersistentVolume.Enabled
	if !persistentStorageEnabled {
		return nil
	}

	log.Info("persistent storage enabled, checking if we need to remove something")
	podLabels := client.MatchingLabels{}

	for k, v := range sts.Spec.Template.Labels {
		podLabels[k] = v
	}

	podOpts := []client.ListOption{
		client.InNamespace(sts.Namespace),
		podLabels,
	}

	podList := &corev1.PodList{}
	if listPodErr := r.Client.List(ctx, podList, podOpts...); listPodErr != nil {
		return fmt.Errorf("could not list pods: %w", listPodErr)
	}

	templates := sts.Spec.VolumeClaimTemplates
	var vctLabels map[string]string
	for i := range templates {
		template := templates[i]
		if template.Name == "datadir" {
			vctLabels = template.Labels
			break
		}
	}

	vctMatchingLabels := client.MatchingLabels{}

	for k, v := range vctLabels {
		// TODO is this expected
		vctMatchingLabels[k] = v
		if k == K8sComponentLabelKey {
			vctMatchingLabels[k] = fmt.Sprintf("%s-statefulset", v)
		}
	}

	vctOpts := []client.ListOption{
		client.InNamespace(sts.Namespace),
		vctMatchingLabels,
	}

	// find the dataDir template
	// now cycle through pvcs, retain volumes for future but delete claims
	pvcList := &corev1.PersistentVolumeClaimList{}
	if listErr := r.Client.List(ctx, pvcList, vctOpts...); listErr != nil {
		return fmt.Errorf("could not get pvc list: %w", listErr)
	}

	pvcsBound := make(map[string]bool, 0)
	for i := range pvcList.Items {
		item := pvcList.Items[i]
		pvcsBound[item.Name] = false
	}

	for j := range podList.Items {
		pod := podList.Items[j]
		// skip pods that are being terminated
		if pod.GetDeletionTimestamp() != nil {
			continue
		}
		volumes := pod.Spec.Volumes
		for i := range volumes {
			volume := volumes[i]
			if volume.VolumeSource.PersistentVolumeClaim != nil {
				pvcsBound[volume.VolumeSource.PersistentVolumeClaim.ClaimName] = true
			}
		}
	}

	if pvcErrorList := r.tryToDeletePVC(log, ctx, ptr.Deref(sts.Spec.Replicas, 0), pvcsBound, pvcList); pvcErrorList != nil {
		return fmt.Errorf("errors found: %w", pvcErrorList)
	}

	return nil
}

func (r *RedpandaReconciler) tryToDeletePVC(log logr.Logger, ctx context.Context, replicas int32, isBoundList map[string]bool, pvcList *corev1.PersistentVolumeClaimList) error {
	var pvcErrorList error

	// here we sort the list of items, should be ordered by ordinal, and we remove the last first so we sort first then remove
	// only the first n matching the number of replicas
	keys := make([]string, 0)
	for k := range isBoundList {
		keys = append(keys, k)
	}

	// sort the pvc strings
	sort.Strings(keys)

	// remove first
	keys = keys[int(replicas):]

	// TODO: may want to not processes cases where we have more than 1 pvcs, so we force the 1 node at a time policy
	// this will avoid dead locking the cluster since you cannot add new nodes, or decomm

	for i := range pvcList.Items {
		item := pvcList.Items[i]

		if isBoundList[item.Name] || !isNameInList(item.Name, keys) {
			continue
		}

		// we are being deleted, before moving forward, try to update PV to avoid data loss
		bestTrySetRetainPV(r.Client, log, ctx, item.Spec.VolumeName, item.Namespace)

		// now we are ready to delete PVC
		if errDelete := r.Client.Delete(ctx, &item); errDelete != nil {
			pvcErrorList = errors.Join(pvcErrorList, fmt.Errorf("could not delete PVC %q: %w", item.Name, errDelete))
		}
	}

	return pvcErrorList
}

func isRedpandaManaged(ctx context.Context, redpandaCluster *redpandav1alpha2.Redpanda) bool {
	log := ctrl.LoggerFrom(ctx).WithName("RedpandaReconciler.isRedpandaManaged")

	managedAnnotationKey := redpandav1alpha2.GroupVersion.Group + managedPath
	if managed, exists := redpandaCluster.Annotations[managedAnnotationKey]; exists && managed == NotManaged {
		log.Info(fmt.Sprintf("management is disabled; to enable it, change the '%s' annotation to true or remove it", managedAnnotationKey))
		return false
	}
	return true
}

func checkDeploymentsStatus(deployments []*appsv1.Deployment) (string, bool) {
	return checkReplicasForList(func(o *appsv1.Deployment) (int32, int32, int32, int32) {
		return o.Status.UpdatedReplicas, o.Status.AvailableReplicas, o.Status.ReadyReplicas, ptr.Deref(o.Spec.Replicas, 0)
	}, deployments, "Deployment")
}

func checkStatefulSetStatus(ss []*appsv1.StatefulSet) (string, bool) {
	return checkReplicasForList(func(o *appsv1.StatefulSet) (int32, int32, int32, int32) {
		return o.Status.UpdatedReplicas, o.Status.AvailableReplicas, o.Status.ReadyReplicas, ptr.Deref(o.Spec.Replicas, 0)
	}, ss, "StatefulSet")
}

type replicasExtractor[T client.Object] func(o T) (updated, available, ready, total int32)

func checkReplicasForList[T client.Object](fn replicasExtractor[T], list []T, resource string) (string, bool) {
	var notReady sort.StringSlice
	for _, item := range list {
		updated, available, ready, total := fn(item)

		if updated != total || available != total || ready != total {
			name := client.ObjectKeyFromObject(item).String()
			item := fmt.Sprintf("%q (updated/available/ready/total: %d/%d/%d/%d)", name, updated, available, ready, total)
			notReady = append(notReady, item)
		}
	}
	if len(notReady) > 0 {
		notReady.Sort()

		return fmt.Sprintf("Not all %s replicas updated, available, and ready for [%s]", resource, strings.Join(notReady, "; ")), false
	}
	return "", true
}
