package users

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/redpanda-data/common-go/rpadmin"
	redpandav1alpha2 "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha2"
	"github.com/redpanda-data/redpanda-operator/src/go/k8s/internal/util/kafka"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

type Client struct {
	user              *redpandav1alpha2.User
	factory           *kafka.ClientFactory
	kafkaClient       *kgo.Client
	kafkaAdminClient  *kadm.Client
	adminClient       *rpadmin.AdminAPI
	scramAPISupported bool
}

func newClient(user *redpandav1alpha2.User, factory *kafka.ClientFactory, kafkaClient *kgo.Client, kafkaAdminClient *kadm.Client, rpClient *rpadmin.AdminAPI, scramAPISupported bool) *Client {
	return &Client{
		user:              user,
		factory:           factory,
		kafkaClient:       kafkaClient,
		kafkaAdminClient:  kafkaAdminClient,
		adminClient:       rpClient,
		scramAPISupported: scramAPISupported,
	}
}

func (c *Client) username() string {
	return c.user.RedpandaName()
}

func (c *Client) userACLName() string {
	return "User:" + c.username()
}

func (c *Client) HasUser(ctx context.Context) (bool, error) {
	if c.scramAPISupported {
		scrams, err := c.kafkaAdminClient.DescribeUserSCRAMs(ctx, c.username())
		if err != nil {
			return false, err
		}
		return len(scrams) == 0, nil
	}

	users, err := c.adminClient.ListUsers(ctx)
	if err != nil {
		return false, err
	}

	return slices.Contains(users, c.username()), nil
}

func (c *Client) ListACLs(ctx context.Context) (*kmsg.DescribeACLsResponse, error) {
	ptrUsername := kmsg.StringPtr(c.userACLName())

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

func (c *Client) CreateACLs(ctx context.Context, acls []kmsg.CreateACLsRequestCreation) error {
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

func (c *Client) DeleteACLs(ctx context.Context, deletions []kmsg.DeleteACLsRequestFilter) error {
	if len(deletions) == 0 {
		return nil
	}

	req := kmsg.NewPtrDeleteACLsRequest()
	req.Filters = deletions

	response, err := req.RequestWith(ctx, c.kafkaClient)
	if err != nil {
		return err
	}

	for _, result := range response.Results {
		if result.ErrorMessage != nil {
			return errors.New(*result.ErrorMessage)
		}
	}

	return nil
}

func (c *Client) DeleteAllACLs(ctx context.Context) error {
	ptrAny := kmsg.StringPtr("any")
	ptrUsername := kmsg.StringPtr(c.userACLName())

	req := kmsg.NewPtrDeleteACLsRequest()
	req.Filters = []kmsg.DeleteACLsRequestFilter{{
		ResourceName:   ptrAny,
		Host:           ptrAny,
		PermissionType: kmsg.ACLPermissionTypeAny,
		ResourceType:   kmsg.ACLResourceTypeAny,
		Principal:      ptrUsername,
		Operation:      kmsg.ACLOperationAny,
	}}

	response, err := req.RequestWith(ctx, c.kafkaClient)
	if err != nil {
		return err
	}

	for _, result := range response.Results {
		if result.ErrorMessage != nil {
			return errors.New(*result.ErrorMessage)
		}
	}

	return nil
}

func (c *Client) getPassword(ctx context.Context) (string, error) {
	passwordInfo := c.user.Spec.Authentication.Password

	if passwordInfo != nil {
		secret := passwordInfo.ValueFrom.SecretKeyRef.Name
		key := passwordInfo.ValueFrom.SecretKeyRef.Key
		if key == "" {
			key = "password"
		}

		var passwordSecret corev1.Secret
		nn := types.NamespacedName{Namespace: c.user.Namespace, Name: secret}
		if err := c.factory.Get(ctx, nn, &passwordSecret); err != nil {
			if !apierrors.IsNotFound(err) {
				return "", err
			}

			return c.generateAndStorePassword(ctx, nn, key)
		}

		data, ok := passwordSecret.Data[key]
		if !ok {
			return c.generateAndStorePassword(ctx, nn, key)
		}

		return string(data), nil
	}

	return "", nil
}

func (c *Client) generateAndStorePassword(ctx context.Context, nn types.NamespacedName, key string) (string, error) {
	password := "changeme"

	if err := c.factory.Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nn.Namespace,
			Name:      nn.Name,
		},
		Data: map[string][]byte{
			key: []byte(password),
		},
	}); err != nil {
		return "", err
	}

	return password, nil
}

func (c *Client) CreateUser(ctx context.Context) error {
	password, err := c.getPassword(ctx)
	if err != nil {
		return err
	}

	if c.scramAPISupported {
		sasl, err := normalizeSASL(c.user.Spec.Authentication.Type)
		if err != nil {
			return err
		}
		_, err = c.kafkaAdminClient.AlterUserSCRAMs(ctx, nil, []kadm.UpsertSCRAM{{
			User:      c.username(),
			Password:  password,
			Mechanism: sasl,
		}})
		return err
	}

	return c.adminClient.CreateUser(ctx, c.username(), password, strings.ToUpper(c.user.Spec.Authentication.Type))
}

func (c *Client) DeleteUser(ctx context.Context) error {
	if c.scramAPISupported {
		_, err := c.kafkaAdminClient.AlterUserSCRAMs(ctx, []kadm.DeleteSCRAM{{
			User: c.username(),
		}}, nil)
		return err
	}

	return c.adminClient.DeleteUser(ctx, c.username())
}

func (c *Client) SyncACLs(ctx context.Context) error {
	acls, err := c.ListACLs(ctx)
	if err != nil {
		return err
	}

	creations, deletions, err := calculateACLs(c.user, acls)
	if err != nil {
		return err
	}

	if err := c.CreateACLs(ctx, creations); err != nil {
		return err
	}
	if err := c.DeleteACLs(ctx, deletions); err != nil {
		return err
	}

	return nil
}

type ClientBuilder struct {
	factory *kafka.ClientFactory
}

func NewClientBuilder(factory *kafka.ClientFactory) *ClientBuilder {
	return &ClientBuilder{
		factory: factory,
	}
}

func (c *ClientBuilder) ForUser(ctx context.Context, user *redpandav1alpha2.User) (*Client, error) {
	kafkaClient, err := c.kafkaClientForUser(ctx, user)
	if err != nil {
		return nil, err
	}

	adminClient, err := c.redpandaAdminClientForUser(ctx, user)
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
			return newClient(user, c.factory, kafkaClient, kafkaAdminClient, adminClient, true), nil
		}
	}

	return newClient(user, c.factory, kafkaClient, kafkaAdminClient, adminClient, false), nil
}

func (c *ClientBuilder) kafkaClientForUser(ctx context.Context, user *redpandav1alpha2.User) (*kgo.Client, error) {
	if user.Spec.ClusterRef != nil {
		var cluster redpandav1alpha2.Redpanda
		if err := c.factory.Get(ctx, types.NamespacedName{Name: user.Spec.ClusterRef.Name, Namespace: user.Namespace}, &cluster); err != nil {
			return nil, err
		}
		return c.factory.GetClusterClient(ctx, &cluster)
	}

	if user.Spec.KafkaAPISpec != nil {
		return c.factory.GetClient(ctx, user.Namespace, nil, user.Spec.KafkaAPISpec)
	}

	return nil, errors.New("unable to determine cluster connection info")
}

func (c *ClientBuilder) redpandaAdminClientForUser(ctx context.Context, user *redpandav1alpha2.User) (*rpadmin.AdminAPI, error) {
	if user.Spec.ClusterRef != nil {
		var cluster redpandav1alpha2.Redpanda
		if err := c.factory.Get(ctx, types.NamespacedName{Name: user.Spec.ClusterRef.Name, Namespace: user.Namespace}, &cluster); err != nil {
			return nil, err
		}
		return c.factory.GetAdminClusterClient(ctx, &cluster)
	}

	if user.Spec.KafkaAPISpec != nil {
		return c.factory.GetAdminClient(ctx, user.Namespace, user.Spec.KafkaAPISpec)
	}

	return nil, errors.New("unable to determine cluster connection info")
}
