package clients

import (
	"errors"

	"github.com/go-logr/logr"
	"github.com/redpanda-data/helm-charts/pkg/redpanda"
	redpandav1alpha1 "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha1"
	redpandav1alpha2 "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha2"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ErrEmptyBrokerList = errors.New("empty broker list")

type ClientFactory struct {
	client.Client
	logger logr.Logger
	config *rest.Config
	dialer redpanda.DialContextFunc
}

func NewClientFactory(config *rest.Config) (*ClientFactory, error) {
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

	return &ClientFactory{
		Client: client,
		config: config,
		logger: logr.Discard(),
	}, nil
}

func (c *ClientFactory) WithDialer(dialer redpanda.DialContextFunc) *ClientFactory {
	c.dialer = dialer
	return c
}

func (c *ClientFactory) WithLogger(logger logr.Logger) *ClientFactory {
	c.logger = logger
	return c
}
