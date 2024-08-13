package redpanda

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/redpanda-data/common-go/rpadmin"
	redpandav1alpha2 "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha2"
	"github.com/redpanda-data/redpanda-operator/src/go/k8s/internal/util/kafka"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	errUnsupportedSASLMechansim = errors.New("unsupported SASL mechanism")
	supportedSASLMechanisms     = map[string]kadm.ScramMechanism{
		"SCRAM-SHA-256": kadm.ScramSha256,
		"SCRAM-SHA-512": kadm.ScramSha512,
	}
)

func normalizeSASL(mechanism string) (kadm.ScramMechanism, error) {
	sasl, ok := supportedSASLMechanisms[strings.ToUpper(mechanism)]
	if !ok {
		return 0, errUnsupportedSASLMechansim
	}

	return sasl, nil
}

// UserController provides users for clusters
type UserController struct {
	client.Client
	factory *kafka.ClientFactory
}

// NewUserController creates UserController
func NewUserController(c client.Client, factory *kafka.ClientFactory) *UserController {
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
	if err := r.Client.Get(ctx, req.NamespacedName, user); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if controllerutil.AddFinalizer(user, FinalizerKey) {
		return ctrl.Result{}, r.Update(ctx, user)
	}

	adminClient, err := r.getAdminClient(ctx, user)
	if err != nil {
		return ctrl.Result{}, err
	}

	hasUser, err := adminClient.HasUser(ctx, user.RedpandaName())
	if err != nil {
		return ctrl.Result{}, err
	}

	if !user.DeletionTimestamp.IsZero() {
		if hasUser {
			if err := adminClient.DeleteUser(ctx, user.RedpandaName()); err != nil {
				return ctrl.Result{}, err
			}
			if err := adminClient.DeleteAllACLs(ctx, user.RedpandaName()); err != nil {
				return ctrl.Result{}, err
			}
		}

		if controllerutil.RemoveFinalizer(user, FinalizerKey) {
			return ctrl.Result{}, r.Update(ctx, user)
		}

		return ctrl.Result{}, nil
	}

	var acls *kmsg.DescribeACLsResponse
	if !hasUser {
		if err := adminClient.CreateUser(ctx, user.RedpandaName(), "changeme", rpadmin.ScramSha512); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		acls, err = adminClient.ListACLs(ctx, user.RedpandaName())
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	creations, deletions, err := calculateACLs(user, acls)
	if err != nil {
		return ctrl.Result{}, err
	}

	fmt.Println("CREATIONS/DELETIONS", creations, deletions)

	if err := adminClient.CreateACLs(ctx, creations); err != nil {
		return ctrl.Result{}, err
	}
	if err := adminClient.DeleteACLs(ctx, deletions); err != nil {
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
		return ctrl.Result{}, r.Client.Status().Update(ctx, user)
	}

	return ctrl.Result{}, nil
}

type aclRule struct {
	ResourceType        kmsg.ACLResourceType
	ResourceName        string
	ResourcePatternType kmsg.ACLResourcePatternType
	Principal           string
	Host                string
	Operation           kmsg.ACLOperation
	PermissionType      kmsg.ACLPermissionType
}

func (r aclRule) toDeletionFilter() kmsg.DeleteACLsRequestFilter {
	return kmsg.DeleteACLsRequestFilter{
		ResourceType:        r.ResourceType,
		ResourceName:        &r.ResourceName,
		ResourcePatternType: r.ResourcePatternType,
		Principal:           &r.Principal,
		Host:                &r.Host,
		Operation:           r.Operation,
		PermissionType:      r.PermissionType,
	}
}

func aclRuleFromCreation(creation kmsg.CreateACLsRequestCreation) aclRule {
	return aclRule{
		ResourceType:        creation.ResourceType,
		ResourceName:        creation.ResourceName,
		ResourcePatternType: creation.ResourcePatternType,
		Principal:           creation.Principal,
		Host:                creation.Host,
		Operation:           creation.Operation,
		PermissionType:      creation.PermissionType,
	}
}

func aclRuleSetFromACLs(acls *kmsg.DescribeACLsResponse) map[aclRule]struct{} {
	rules := map[aclRule]struct{}{}

	if acls == nil {
		return rules
	}

	for _, acls := range acls.Resources {
		for _, acl := range acls.ACLs {
			rules[aclRule{
				ResourceType:        acls.ResourceType,
				ResourceName:        acls.ResourceName,
				ResourcePatternType: acls.ResourcePatternType,
				Principal:           acl.Principal,
				Host:                acl.Host,
				Operation:           acl.Operation,
				PermissionType:      acl.PermissionType,
			}] = struct{}{}
		}
	}

	return rules
}

func calculateACLs(user *redpandav1alpha2.User, acls *kmsg.DescribeACLsResponse) ([]kmsg.CreateACLsRequestCreation, []kmsg.DeleteACLsRequestFilter, error) {
	existing := aclRuleSetFromACLs(acls)
	toDelete := aclRuleSetFromACLs(acls)

	creations := map[aclRule]kmsg.CreateACLsRequestCreation{}

	for _, rule := range user.Spec.Authorization.ACLs {
		resourceType, err := kmsg.ParseACLResourceType(rule.Resource.Type)
		if err != nil {
			return nil, nil, err
		}

		permType, err := kmsg.ParseACLPermissionType(rule.Type)
		if err != nil {
			return nil, nil, err
		}

		patternType, err := kmsg.ParseACLResourcePatternType(rule.Resource.PatternType)
		if err != nil {
			return nil, nil, err
		}

		for _, operation := range rule.Operations {
			op, err := kmsg.ParseACLOperation(operation)
			if err != nil {
				return nil, nil, err
			}

			var creation kmsg.CreateACLsRequestCreation

			creation.ResourceType = resourceType
			creation.Host = rule.Host
			if creation.Host == "" {
				creation.Host = "*"
			}
			creation.PermissionType = permType
			creation.ResourcePatternType = patternType
			creation.ResourceName = rule.Resource.Name
			creation.Principal = "User:" + user.RedpandaName()
			creation.Operation = op

			aclRule := aclRuleFromCreation(creation)
			delete(toDelete, aclRule)

			if _, exists := existing[aclRule]; exists {
				continue
			}

			creations[aclRule] = creation
		}
	}

	var flattenedCreations []kmsg.CreateACLsRequestCreation
	for _, creation := range creations {
		flattenedCreations = append(flattenedCreations, creation)
	}

	var deletions []kmsg.DeleteACLsRequestFilter
	for deletion := range toDelete {
		deletions = append(deletions, deletion.toDeletionFilter())
	}

	return flattenedCreations, deletions, nil
}

type adminClient struct {
	kafkaClient       *kgo.Client
	kafkaAdminClient  *kadm.Client
	adminClient       *rpadmin.AdminAPI
	scramAPISupported bool
}

func newAdminClient(kafkaClient *kgo.Client, kafkaAdminClient *kadm.Client, rpClient *rpadmin.AdminAPI, scramAPISupported bool) *adminClient {
	return &adminClient{
		kafkaClient:       kafkaClient,
		kafkaAdminClient:  kafkaAdminClient,
		adminClient:       rpClient,
		scramAPISupported: scramAPISupported,
	}
}

func (c *adminClient) HasUser(ctx context.Context, username string) (bool, error) {
	if c.scramAPISupported {
		scrams, err := c.kafkaAdminClient.DescribeUserSCRAMs(ctx, username)
		if err != nil {
			return false, err
		}
		return len(scrams) == 0, nil
	}

	users, err := c.adminClient.ListUsers(ctx)
	if err != nil {
		return false, err
	}

	return slices.Contains(users, username), nil
}

func (c *adminClient) ListACLs(ctx context.Context, username string) (*kmsg.DescribeACLsResponse, error) {
	ptrUsername := kmsg.StringPtr("User:" + username)

	req := kmsg.NewPtrDescribeACLsRequest()
	req.PermissionType = kmsg.ACLPermissionTypeAny
	req.ResourceType = kmsg.ACLResourceTypeAny
	req.Principal = ptrUsername
	req.Operation = kmsg.ACLOperationAny

	response, err := req.RequestWith(ctx, c.kafkaClient)
	if err != nil {
		return nil, err
	}
	if response.ErrorMessage != nil {
		return nil, errors.New(*response.ErrorMessage)
	}

	return response, nil
}

func (c *adminClient) CreateACLs(ctx context.Context, acls []kmsg.CreateACLsRequestCreation) error {
	if len(acls) == 0 {
		return nil
	}

	req := kmsg.NewPtrCreateACLsRequest()
	req.Creations = acls

	creation, err := req.RequestWith(ctx, c.kafkaClient)
	if err != nil {
		return err
	}

	for _, result := range creation.Results {
		if result.ErrorMessage != nil {
			return errors.New(*result.ErrorMessage)
		}
	}

	return nil
}

func (c *adminClient) DeleteACLs(ctx context.Context, deletions []kmsg.DeleteACLsRequestFilter) error {
	if len(deletions) == 0 {
		return nil
	}

	req := kmsg.NewPtrDeleteACLsRequest()
	req.Filters = deletions

	_, err := req.RequestWith(ctx, c.kafkaClient)
	return err
}

func (c *adminClient) DeleteAllACLs(ctx context.Context, username string) error {
	ptrAny := kmsg.StringPtr("any")
	ptrUsername := kmsg.StringPtr("User:" + username)

	req := kmsg.NewPtrDeleteACLsRequest()
	req.Filters = []kmsg.DeleteACLsRequestFilter{{
		ResourceName:   ptrAny,
		Host:           ptrAny,
		PermissionType: kmsg.ACLPermissionTypeAny,
		ResourceType:   kmsg.ACLResourceTypeAny,
		Principal:      ptrUsername,
		Operation:      kmsg.ACLOperationAny,
	}}

	_, err := req.RequestWith(ctx, c.kafkaClient)
	return err
}

func (c *adminClient) CreateUser(ctx context.Context, username, password, mechanism string) error {
	if c.scramAPISupported {
		sasl, err := normalizeSASL(mechanism)
		if err != nil {
			return err
		}
		_, err = c.kafkaAdminClient.AlterUserSCRAMs(ctx, nil, []kadm.UpsertSCRAM{{
			User:      username,
			Password:  password,
			Mechanism: sasl,
		}})
		return err
	}

	return c.adminClient.CreateUser(ctx, username, password, mechanism)
}

func (c *adminClient) DeleteUser(ctx context.Context, username string) error {
	if c.scramAPISupported {
		_, err := c.kafkaAdminClient.AlterUserSCRAMs(ctx, []kadm.DeleteSCRAM{{
			User: username,
		}}, nil)
		return err
	}

	return c.adminClient.DeleteUser(ctx, username)
}

func (r *UserController) getAdminClient(ctx context.Context, user *redpandav1alpha2.User) (*adminClient, error) {
	kafkaClient, err := r.getKafkaAdminClient(ctx, user)
	if err != nil {
		return nil, err
	}

	adminClient, err := r.getRedpandaAdminClient(ctx, user)
	if err != nil {
		return nil, err
	}

	kafkaAdminClient := kadm.NewClient(kafkaClient)
	brokerAPI, err := kafkaAdminClient.ApiVersions(ctx)
	if err != nil {
		return nil, err
	}

	for _, api := range brokerAPI {
		_, _, supported := api.KeyVersions(kmsg.DescribeUserSCRAMCredentials.Int16())
		if supported {
			return newAdminClient(kafkaClient, kafkaAdminClient, adminClient, true), nil
		}
	}
	return newAdminClient(kafkaClient, kafkaAdminClient, adminClient, false), nil
}

func (r *UserController) getKafkaAdminClient(ctx context.Context, user *redpandav1alpha2.User) (*kgo.Client, error) {
	if user.Spec.ClusterRef != nil {
		var cluster redpandav1alpha2.Redpanda
		if err := r.Get(ctx, types.NamespacedName{Name: user.Spec.ClusterRef.Name, Namespace: user.Namespace}, &cluster); err != nil {
			return nil, err
		}
		return r.factory.GetClusterClient(ctx, &cluster)
	}

	if user.Spec.KafkaAPISpec != nil {
		return r.factory.GetClient(ctx, user.Namespace, nil, user.Spec.KafkaAPISpec)
	}

	return nil, errors.New("unable to determine cluster connection info")
}

func (r *UserController) getRedpandaAdminClient(ctx context.Context, user *redpandav1alpha2.User) (*rpadmin.AdminAPI, error) {
	if user.Spec.ClusterRef != nil {
		var cluster redpandav1alpha2.Redpanda
		if err := r.Get(ctx, types.NamespacedName{Name: user.Spec.ClusterRef.Name, Namespace: user.Namespace}, &cluster); err != nil {
			return nil, err
		}
		return r.factory.GetAdminClusterClient(ctx, &cluster)
	}

	if user.Spec.KafkaAPISpec != nil {
		return r.factory.GetAdminClient(ctx, user.Namespace, user.Spec.KafkaAPISpec)
	}

	return nil, errors.New("unable to determine cluster connection info")
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
