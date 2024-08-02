package util

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"

	redpandav1alpha1 "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/cluster.redpanda.com/v1alpha1"
	redpandav1alpha2 "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha2"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	serviceLabel                = "kubernetes.io/service-name"
	helmInternalCertificateName = "default"
	defaultCertSecretName       = "redpanda-default-cert"

	adminPort = "admin"
)

type ClientFactory struct {
	client.Client
}

func NewClientFactory(client client.Client) *ClientFactory {
	return &ClientFactory{
		Client: client,
	}
}

// Admin returns a client able to communicate with the cluster defined by the given KafkaAPISpec.
func (c *ClientFactory) Admin(ctx context.Context, namespace string, spec *redpandav1alpha1.KafkaAPISpec) (*kadm.Client, error) {
	client, err := c.getClient(ctx, namespace, spec)
	if err != nil {
		return nil, err
	}

	return kadm.NewClient(client), nil
}

// ClusterAdmin returns a client able to communicate with the given Redpanda cluster.
func (c *ClientFactory) ClusterAdmin(ctx context.Context, cluster *redpandav1alpha2.Redpanda) (*kadm.Client, error) {
	client, err := c.getClusterClient(ctx, adminPort, cluster)
	if err != nil {
		return nil, err
	}

	return kadm.NewClient(client), nil
}

// getClient returns a simple kgo.Client able to communicate with the given cluster specified via KafkaAPISpec.
func (c *ClientFactory) getClient(ctx context.Context, namespace string, spec *redpandav1alpha1.KafkaAPISpec) (*kgo.Client, error) {
	if len(spec.Brokers) == 0 {
		return nil, errors.New("no brokers")
	}

	clientCerts, pool, err := c.getCertificates(ctx, namespace, spec)
	if err != nil {
		return nil, err
	}

	opts := []kgo.Opt{
		kgo.SeedBrokers(spec.Brokers...),
	}

	if pool != nil {
		opts = append(opts, kgo.DialTLSConfig(&tls.Config{
			Certificates: clientCerts,
			MinVersion:   tls.VersionTLS12,
			RootCAs:      pool,
		}))
	}

	return kgo.NewClient(opts...)
}

// getClusterClient returns a simple kgo.Client able to communicate with the given cluster.
func (c *ClientFactory) getClusterClient(ctx context.Context, portName string, cluster *redpandav1alpha2.Redpanda) (*kgo.Client, error) {
	brokers, err := c.getClusterBrokers(ctx, portName, cluster)
	if err != nil {
		return nil, err
	}

	if len(brokers) == 0 {
		return nil, errors.New("no brokers")
	}

	clientCerts, pool, err := c.getClusterCertificates(ctx, cluster)
	if err != nil {
		return nil, err
	}

	opts := []kgo.Opt{
		kgo.SeedBrokers(brokers...),
	}

	if pool != nil {
		opts = append(opts, kgo.DialTLSConfig(&tls.Config{
			Certificates: clientCerts,
			MinVersion:   tls.VersionTLS12,
			RootCAs:      pool,
		}))
	}

	return kgo.NewClient(opts...)
}

// getClusterBrokers fetches the internal broker addresses for a Redpanda cluster, returning
// hostname:port pairs where port is the named port exposed by the internal cluster service.
func (c *ClientFactory) getClusterBrokers(ctx context.Context, portName string, cluster *redpandav1alpha2.Redpanda) ([]string, error) {
	service := &corev1.Service{}
	if err := c.Get(ctx, client.ObjectKeyFromObject(cluster), service); err != nil {
		return nil, err
	}
	endpointSlices := &discoveryv1.EndpointSliceList{}
	if err := c.List(ctx, endpointSlices, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set{
			serviceLabel: service.Name,
		}),
		Namespace: service.Namespace,
	}); err != nil {
		return nil, err
	}

	addresses := []string{}
	for _, endpointSlice := range endpointSlices.Items {
		for _, port := range endpointSlice.Ports {
			if port.Name != nil && *port.Name == portName && port.Port != nil {
				for _, endpoint := range endpointSlice.Endpoints {
					for _, address := range endpoint.Addresses {
						addresses = append(addresses, fmt.Sprintf("%s:%d", address, *port.Port))
					}
				}
			}
		}
	}
	return addresses, nil
}

// getClusterCertificates fetches and parses client and server certificates for connecting
// to a Redpanda cluster
func (c *ClientFactory) getClusterCertificates(ctx context.Context, cluster *redpandav1alpha2.Redpanda) ([]tls.Certificate, *x509.CertPool, error) {
	client, server, err := c.getClusterCertificateSecrets(ctx, cluster)
	if err != nil {
		return nil, nil, err
	}

	if server == nil {
		return nil, nil, nil
	}

	serverPubKey := server.Data[corev1.TLSCertKey]
	caBlock, _ := pem.Decode(serverPubKey)
	ca, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	pool := x509.NewCertPool()
	pool.AddCert(ca)

	if client != nil {
		clientCert := client.Data[corev1.TLSCertKey]
		clientPrivateKey := client.Data[corev1.TLSPrivateKeyKey]
		clientKey, err := tls.X509KeyPair(clientCert, clientPrivateKey)
		if err != nil {
			return nil, nil, err
		}
		return []tls.Certificate{clientKey}, pool, nil
	}

	return nil, pool, nil
}

func (c *ClientFactory) getCertificates(ctx context.Context, namespace string, spec *redpandav1alpha1.KafkaAPISpec) ([]tls.Certificate, *x509.CertPool, error) {
	clientCert, clientPrivateKey, serverPubKey, err := c.getCertificateSecrets(ctx, namespace, spec)
	if err != nil {
		return nil, nil, err
	}

	if serverPubKey == nil {
		return nil, nil, nil
	}

	caBlock, _ := pem.Decode(serverPubKey)
	ca, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	pool := x509.NewCertPool()
	pool.AddCert(ca)

	if clientCert != nil && clientPrivateKey != nil {
		clientKey, err := tls.X509KeyPair(clientCert, clientPrivateKey)
		if err != nil {
			return nil, nil, err
		}
		return []tls.Certificate{clientKey}, pool, nil
	}

	return nil, pool, nil
}

// getCertificateSecrets fetches the client and server certs for a Redpanda cluster
func (c *ClientFactory) getCertificateSecrets(ctx context.Context, namespace string, spec *redpandav1alpha1.KafkaAPISpec) ([]byte, []byte, []byte, error) {
	tls := spec.TLS
	if tls == nil {
		return nil, nil, nil, nil
	}

	if tls.CaCert != nil {
		return c.getCertificateSecretsFromV1Refs(ctx, namespace, spec.TLS.CaCert, spec.TLS.Cert, spec.TLS.Key)
	}

	return c.getDefaultCertificateV1Secrets(ctx, namespace, spec.TLS.Cert, spec.TLS.Key)
}

// getClusterCertificateSecrets fetches the client and server certs for a Redpanda cluster
func (c *ClientFactory) getClusterCertificateSecrets(ctx context.Context, cluster *redpandav1alpha2.Redpanda) (*corev1.Secret, *corev1.Secret, error) {
	tls := cluster.Spec.ClusterSpec.TLS
	if tls == nil {
		return c.getDefaultCertificateSecrets(ctx, cluster.Namespace)
	}
	if tls.Enabled != nil && !*tls.Enabled {
		return nil, nil, nil
	}
	if internal, ok := tls.Certs[helmInternalCertificateName]; ok {
		return c.getCertificateSecretsFromRefs(ctx, cluster.Namespace, internal.SecretRef, internal.ClientSecretRef)
	}
	return c.getDefaultCertificateSecrets(ctx, cluster.Namespace)
}

// getDefaultCertificateSecrets fetches the default server secret
func (c *ClientFactory) getDefaultCertificateSecrets(ctx context.Context, namespace string) (*corev1.Secret, *corev1.Secret, error) {
	return c.getCertificateSecretsFromRefs(ctx, namespace, &redpandav1alpha2.SecretRef{Name: defaultCertSecretName}, nil)
}

// getCertificateSecretsFromRefs fetches a server and client (if specified) secret from SecretRefs
func (c *ClientFactory) getCertificateSecretsFromRefs(ctx context.Context, namespace string, server, client *redpandav1alpha2.SecretRef) (*corev1.Secret, *corev1.Secret, error) {
	serverSecret := &corev1.Secret{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: server.Name}, serverSecret); err != nil {
		return nil, nil, err
	}
	if client != nil {
		clientSecret := &corev1.Secret{}
		if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: client.Name}, clientSecret); err != nil {
			return nil, nil, err
		}
		return clientSecret, serverSecret, nil
	}
	return nil, serverSecret, nil
}

// getDefaultCertificateV1Secrets fetches the default server secret
func (c *ClientFactory) getDefaultCertificateV1Secrets(ctx context.Context, namespace string, clientCert, clientKey *redpandav1alpha1.SecretKeyRef) ([]byte, []byte, []byte, error) {
	return c.getCertificateSecretsFromV1Refs(ctx, namespace, &redpandav1alpha1.SecretKeyRef{Name: defaultCertSecretName, Key: corev1.TLSCertKey}, clientCert, clientKey)
}

// getCertificateSecretsFromV1Refs fetches a server and client (if specified) secret from SecretRefs
func (c *ClientFactory) getCertificateSecretsFromV1Refs(ctx context.Context, namespace string, server, clientCert, clientKey *redpandav1alpha1.SecretKeyRef) ([]byte, []byte, []byte, error) {
	serverSecret := &corev1.Secret{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: server.Name}, serverSecret); err != nil {
		return nil, nil, nil, err
	}

	serverCA := serverSecret.Data[server.Key]

	if clientCert != nil && clientKey != nil {
		clientCertSecret := &corev1.Secret{}
		if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: clientCert.Name}, clientCertSecret); err != nil {
			return nil, nil, nil, err
		}
		clientKeySecret := &corev1.Secret{}
		if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: clientKey.Name}, clientKeySecret); err != nil {
			return nil, nil, nil, err
		}
		return clientCertSecret.Data[clientCert.Key], clientKeySecret.Data[clientKey.Key], serverCA, nil
	}
	return nil, nil, serverCA, nil
}
