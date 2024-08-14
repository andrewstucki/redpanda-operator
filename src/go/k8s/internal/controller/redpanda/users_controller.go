package redpanda

import (
	"context"

	redpandav1alpha2 "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha2"
	"github.com/redpanda-data/redpanda-operator/src/go/k8s/internal/controller/redpanda/users"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// UserController provides users for clusters
type UserController struct {
	client.Client
	factory *users.ClientFactory
}

// NewUserController creates UserController
func NewUserController(c client.Client, factory *users.ClientFactory) *UserController {
	return &UserController{
		Client:  c,
		factory: factory,
	}
}

// Reconcile reconciles a Redpanda user.
func (r *UserController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx).WithName("UserController.Reconcile")

	log.Info("Starting reconcile loop")
	defer log.Info("Finished reconcile loop")

	user := &redpandav1alpha2.User{}
	if err := r.Get(ctx, req.NamespacedName, user); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if controllerutil.AddFinalizer(user, FinalizerKey) {
		return ctrl.Result{}, r.Update(ctx, user)
	}

	client, err := r.factory.ClientForUser(ctx, user)
	if err != nil {
		return ctrl.Result{}, err
	}

	hasUser, err := client.HasUser(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !user.DeletionTimestamp.IsZero() {
		if hasUser {
			if err := client.DeleteUser(ctx); err != nil {
				return ctrl.Result{}, err
			}
			if err := client.DeleteAllACLs(ctx); err != nil {
				return ctrl.Result{}, err
			}
		}

		if controllerutil.RemoveFinalizer(user, FinalizerKey) {
			return ctrl.Result{}, r.Update(ctx, user)
		}

		return ctrl.Result{}, nil
	}

	if !hasUser {
		if err := client.CreateUser(ctx); err != nil {
			return ctrl.Result{}, err
		}
	}

	if err := client.SyncACLs(ctx); err != nil {
		return ctrl.Result{}, err
	}

	if apimeta.SetStatusCondition(&user.Status.Conditions, metav1.Condition{
		Type:               "Created",
		Status:             metav1.ConditionTrue,
		ObservedGeneration: user.Generation,
		Reason:             "Created",
		Message:            "User successfully created",
	}) || user.Status.ClusterRef != user.Spec.ClusterRef {
		user.Status.ClusterRef = user.Spec.ClusterRef

		return ctrl.Result{}, r.Status().Update(ctx, user)
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
			requests, err := usersForCluster(ctx, r.Client, client.ObjectKeyFromObject(o))
			if err != nil {
				mgr.GetLogger().V(1).Info("skipping reconciliation due to fetching error", "error", err)
				return nil
			}
			return requests
		})).
		Complete(r)
}
