// Copyright 2021 Redpanda Data, Inc.
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

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	redpandav1alpha2 "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha2"
	"github.com/redpanda-data/redpanda-operator/src/go/k8s/internal/util"
	"github.com/twmb/franz-go/pkg/kadm"
)

// UserController provides users for clusters
type UserController struct {
	client.Client
	clients   *util.ClientFactory
	generator *util.PasswordGenerator
}

// NewUserController creates UserController
func NewUserController(c client.Client, clients *util.ClientFactory) *UserController {
	return &UserController{
		Client:    c,
		clients:   clients,
		generator: util.NewPasswordGenerator(),
	}
}

// Reconcile reconciles a Redpanda user.
func (r *UserController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx, done := context.WithCancel(ctx)
	defer done()

	log := ctrl.LoggerFrom(ctx).WithName("UserController.Reconcile")

	log.Info("Starting reconcile loop")
	defer log.Info("Finished reconcile loop")

	user := &redpandav1alpha2.User{}
	if err := r.Client.Get(ctx, req.NamespacedName, user); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// TODO: finalizer

	var client *kadm.Client
	var err error

	if user.Spec.ClusterRef != nil {
		var cluster *redpandav1alpha2.Redpanda
		cluster, err = r.getClusterFromRef(ctx, user.Namespace, user.Spec.ClusterRef)
		if err != nil {
			// TODO: set not found status on the User object if not found
			return ctrl.Result{}, err
		}
		client, err = r.clients.ClusterAdmin(ctx, cluster)
	} else {
		client, err = r.clients.Admin(ctx, user.Namespace, user.Spec.KafkaAPISpec)
	}

	if err != nil {
		return ctrl.Result{}, err
	}

	// https://github.com/strimzi/strimzi-kafka-operator/blob/f640509e4e812671e5b93f3cf6d0f41eabde2bc8/user-operator/src/main/java/io/strimzi/operator/user/operator/KafkaUserOperator.java#L387-L399
	switch user.Spec.Authentication.Type {
	case "scram-sha-512":
		scram := kadm.UpsertSCRAM{
			User:      user.Name,
			Mechanism: kadm.ScramSha512,
			// https://github.com/strimzi/strimzi-kafka-operator/blob/f640509e4e812671e5b93f3cf6d0f41eabde2bc8/user-operator/src/main/java/io/strimzi/operator/user/operator/ScramCredentialsOperator.java#L35
			Iterations: 4096,
		}
		if user.Spec.Authentication.Password == nil {
			password, err := r.generator.Generate()
			if err != nil {
				return ctrl.Result{}, err
			}
			scram.Password = password
		}
		// TODO: fetch password
		_, err := client.AlterUserSCRAMs(ctx, nil, []kadm.UpsertSCRAM{scram})
		if err != nil {
			return ctrl.Result{}, err
		}
	case "tls":
	case "tls-external":
	}

	return ctrl.Result{}, nil
}

// +kubebuilder:rbac:groups=cluster.redpanda.com,namespace=default,resources=users,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.redpanda.com,namespace=default,resources=users/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.redpanda.com,namespace=default,resources=users/finalizers,verbs=update

// For cluster scoped operator

//+kubebuilder:rbac:groups=cluster.redpanda.com,resources=users,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=cluster.redpanda.com,resources=users/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cluster.redpanda.com,resources=users/finalizers,verbs=update

// SetupWithManager sets up the controller with the Manager.
func (r *UserController) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	if err := registerUserClusterIndex(ctx, mgr); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&redpandav1alpha2.User{}).
		Watches(&redpandav1alpha2.Redpanda{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			// TODO: consider using a lossy workqueue rather than just no-oping errors here
			requests, err := usersForCluster(ctx, r.Client, client.ObjectKeyFromObject(o))
			if err != nil {
				// TODO: log something
				return nil
			}
			return requests
		})).
		Complete(r)
}

func (r *UserController) getClusterFromRef(ctx context.Context, namespace string, ref *redpandav1alpha2.ClusterRef) (*redpandav1alpha2.Redpanda, error) {
	cluster := &redpandav1alpha2.Redpanda{}
	key := types.NamespacedName{Name: ref.Name, Namespace: ref.Namespace}
	if ref.Namespace == "" {
		key.Namespace = namespace
	}
	if err := r.Get(ctx, key, cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}
