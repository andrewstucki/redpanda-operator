package clients

import (
	"context"
	"errors"

	"github.com/go-logr/logr"
	"github.com/redpanda-data/common-go/rpadmin"
	"github.com/redpanda-data/helm-charts/pkg/redpanda"
	redpandav1alpha1 "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha1"
	redpandav1alpha2 "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha2"
	"github.com/twmb/franz-go/pkg/kgo"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ErrEmptyBrokerList = errors.New("empty broker list")

// ClientFactory is responsible for creating both high-level and low-level clients used in our
// controllers.
//
// Calling its `Kafka*` methods will initialize a low-level kgo.Client instance
// based on the connection parameters contained within the corresponding CRD struct passed in
// at method invocation.
//
// Calling its `RedpandaAdmin*` methods will initialize a low-level rpadmin.AdminAPI instance
// based on the connection parameters contained within the corresponding CRD struct passed in
// at method invocation.
//
// Additionally, high-level clients that act as functional wrappers to the low-level
// clients to ease high-level operations (i.e. syncing all primitives for a v1alpha2.User) may
// be initialized through calling the appropriate `*Client` methods.
type ClientFactory interface {
	// WithDialer causes the underlying connections to use a different dialer implementation.
	WithDialer(dialer redpanda.DialContextFunc) ClientFactory
	// WithLogger specifies a logger to be passed to the underlying clients.
	WithLogger(logger logr.Logger) ClientFactory

	// KafkaClient initializes a kgo.Client based on the spec of the passed in struct.
	// The struct *must* implement either the KafkaConnectedObject interface of the ClusterReferencingObject
	// interface to properly initialize.
	KafkaClient(ctx context.Context, object client.Object, opts ...kgo.Opt) (*kgo.Client, error)

	// RedpandaAdminClient initializes a rpadmin.AdminAPI client based on the spec of the passed in struct.
	// The struct *must* implement either the AdminConnectedObject interface of the ClusterReferencingObject
	// interface to properly initialize.
	RedpandaAdminClient(ctx context.Context, object client.Object) (*rpadmin.AdminAPI, error)

	// UserClient returns a client that can be used for managing Users and ACLs for a Redpanda cluster based on
	// the configuration and connection information passed in via a v1alpha2.User struct.
	UserClient(ctx context.Context, user *redpandav1alpha2.User, opts ...kgo.Opt) (UserClient, error)
}

var ErrInvalidKafkaClientObject = errors.New("cannot initialize Kafka API client from given object")
var ErrInvalidRedpandaClientObject = errors.New("cannot initialize Redpanda Admin API client from given object")

func (c *clientFactory) KafkaClient(ctx context.Context, obj client.Object, opts ...kgo.Opt) (*kgo.Client, error) {
	namespace := obj.GetNamespace()

	if o, ok := obj.(ClusterReferencingObject); ok {
		if ref := o.GetClusterRef(); ref != nil {
			var cluster redpandav1alpha2.Redpanda

			if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: ref.Name}, &cluster); err != nil {
				return nil, err
			}

			return c.KafkaForCluster(ctx, &cluster, opts...)
		}
	}

	var metricsNamespace *string
	if o, ok := obj.(KafkaConnectedObjectWithMetrics); ok {
		metricsNamespace = o.GetMetricsNamespace()
	}

	if o, ok := obj.(KafkaConnectedObject); ok {
		if spec := o.GetKafkaAPISpec(); spec != nil {
			return c.KafkaForSpec(ctx, namespace, metricsNamespace, spec, opts...)
		}
	}

	return nil, ErrInvalidKafkaClientObject
}

type KafkaConnectedObject interface {
	client.Object
	GetKafkaAPISpec() *redpandav1alpha2.KafkaAPISpec
}

type KafkaConnectedObjectWithMetrics interface {
	KafkaConnectedObject
	GetMetricsNamespace() *string
}

func (c *clientFactory) RedpandaAdminClient(ctx context.Context, obj client.Object) (*rpadmin.AdminAPI, error) {
	namespace := obj.GetNamespace()

	if o, ok := obj.(ClusterReferencingObject); ok {
		if ref := o.GetClusterRef(); ref != nil {
			var cluster redpandav1alpha2.Redpanda

			if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: ref.Name}, &cluster); err != nil {
				return nil, err
			}

			return c.RedpandaAdminForCluster(ctx, &cluster)
		}
	}

	if o, ok := obj.(AdminConnectedObject); ok {
		if spec := o.GetAdminAPISpec(); spec != nil {
			return c.RedpandaAdminForSpec(ctx, namespace, spec)
		}
	}

	return nil, ErrInvalidRedpandaClientObject
}

type AdminConnectedObject interface {
	client.Object
	GetAdminAPISpec() *redpandav1alpha2.AdminAPISpec
}

type ClusterReferencingObject interface {
	client.Object
	GetClusterRef() *redpandav1alpha2.ClusterRef
}

type clientFactory struct {
	client.Client
	logger logr.Logger
	config *rest.Config
	dialer redpanda.DialContextFunc
}

var _ ClientFactory = (*clientFactory)(nil)

func NewClientFactory(config *rest.Config) (ClientFactory, error) {
	s := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(s); err != nil {
		return nil, err
	}

	if err := redpandav1alpha2.AddToScheme(s); err != nil {
		return nil, err
	}

	if err := redpandav1alpha1.AddToScheme(s); err != nil {
		return nil, err
	}

	client, err := client.New(config, client.Options{Scheme: s})
	if err != nil {
		return nil, err
	}

	return &clientFactory{
		Client: client,
		config: config,
		logger: logr.Discard(),
	}, nil
}

func (c *clientFactory) WithDialer(dialer redpanda.DialContextFunc) ClientFactory {
	c.dialer = dialer
	return c
}

func (c *clientFactory) WithLogger(logger logr.Logger) ClientFactory {
	c.logger = logger
	return c
}
