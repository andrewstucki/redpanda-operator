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
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	redpandav1alpha1 "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha2"
)

// RedpandaUserController provides users for clusters
type RedpandaUserController struct {
	client.Client
}

// NewRedpandaUserController creates RedpandaUserController
func NewRedpandaUserController(c client.Client) *RedpandaUserController {
	return &RedpandaUserController{
		Client:                     c,
	}
}

// Reconcile reconciles a Redpanda user.
func (r *RedpandaUserController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx, done := context.WithCancel(ctx)
	defer done()

	log := ctrl.LoggerFrom(ctx).WithName("RedpandaUserController.Reconcile")

	log.Info("Starting reconcile loop")
	defer log.Info("Finished reconcile loop")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RedpandaUserController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&redpandav1alpha1.RedpandaUser{}).
		Complete(r)
}
