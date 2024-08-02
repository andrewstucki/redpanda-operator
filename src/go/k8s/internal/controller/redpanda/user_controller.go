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
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	redpandav1alpha2 "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha2"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

const serviceLabel = "kubernetes.io/service-name"

// UserController provides users for clusters
type UserController struct {
	client.Client
	generator *passwordGenerator
}

// NewUserController creates UserController
func NewUserController(c client.Client) *UserController {
	return &UserController{
		Client:    c,
		generator: newPasswordGenerator(),
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

	client, err := r.getRedpandaClient(ctx, user)
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
			password, err := r.generator.generate()
			if err != nil {
				return ctrl.Result{}, err
			}
			scram.Password = password
		}
		// TODO: fetch password
		// password := rand.
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

const (
	userClusterIndex = "__user_referencing_cluster"
)

func userCluster(user *redpandav1alpha2.User) types.NamespacedName {
	nn := types.NamespacedName{Namespace: user.Spec.ClusterRef.Namespace, Name: user.Spec.ClusterRef.Name}
	if nn.Namespace == "" {
		nn.Namespace = user.Namespace
	}
	return nn
}

func registerUserClusterIndex(ctx context.Context, mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(ctx, &redpandav1alpha2.User{}, userClusterIndex, indexUserCluster)
}

func indexUserCluster(o client.Object) []string {
	user := o.(*redpandav1alpha2.User)
	return []string{userCluster(user).String()}
}

func usersForCluster(ctx context.Context, c client.Client, nn types.NamespacedName) ([]reconcile.Request, error) {
	childList := &redpandav1alpha2.UserList{}
	err := c.List(ctx, childList, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(userClusterIndex, nn.String()),
	})

	if err != nil {
		return nil, err
	}

	requests := []reconcile.Request{}
	for _, item := range childList.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: item.GetNamespace(),
				Name:      item.GetName(),
			},
		})
	}

	return requests, nil
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

func (r *UserController) getRedpandaClient(ctx context.Context, user *redpandav1alpha2.User) (*kadm.Client, error) {
	cluster, err := r.getClusterFromRef(ctx, user.Namespace, &user.Spec.ClusterRef)
	if err != nil {
		// TODO: set not found status on the User object if not found
		return nil, err
	}

	return getInternalAdminClient(ctx, r.Client, cluster)
}

func getInternalAdminClient(ctx context.Context, c client.Client, cluster *redpandav1alpha2.Redpanda) (*kadm.Client, error) {
	brokers, err := getBrokers(ctx, c, cluster)
	if err != nil {
		return nil, err
	}

	if len(brokers) == 0 {
		return nil, errors.New("no brokers")
	}

	clientCerts, pool, err := getCertificates(ctx, c, cluster)
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

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, err
	}

	return kadm.NewClient(client), nil
}

// client, server, error
func getCertificates(ctx context.Context, c client.Client, cluster *redpandav1alpha2.Redpanda) ([]tls.Certificate, *x509.CertPool, error) {
	client, server, err := getCertificateSecrets(ctx, c, cluster)
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

// client, server, error
func getCertificateSecrets(ctx context.Context, c client.Client, cluster *redpandav1alpha2.Redpanda) (*corev1.Secret, *corev1.Secret, error) {
	tls := cluster.Spec.ClusterSpec.TLS
	if tls == nil {
		return getDefaultCertificateSecrets(ctx, c, cluster.Namespace)
	}
	if tls.Enabled != nil && !*tls.Enabled {
		return nil, nil, nil
	}
	if internal, ok := tls.Certs["default"]; ok {
		return getCertificateSecretsFromRefs(ctx, c, cluster.Namespace, internal.SecretRef, internal.ClientSecretRef)
	}
	return getDefaultCertificateSecrets(ctx, c, cluster.Namespace)
}

func getDefaultCertificateSecrets(ctx context.Context, c client.Client, namespace string) (*corev1.Secret, *corev1.Secret, error) {
	return getCertificateSecretsFromRefs(ctx, c, namespace, &redpandav1alpha2.SecretRef{Name: "redpanda-default-cert"}, nil)
}

func getCertificateSecretsFromRefs(ctx context.Context, c client.Client, namespace string, server, client *redpandav1alpha2.SecretRef) (*corev1.Secret, *corev1.Secret, error) {
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

func getBrokers(ctx context.Context, c client.Client, cluster *redpandav1alpha2.Redpanda) ([]string, error) {
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
			if port.Name != nil && *port.Name == "admin" && port.Port != nil {
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

// public PasswordGenerator(int length, String firstCharacterAlphabet, String alphabet) {
// 	this.length = length;
// 	this.firstCharacterAlphabet = firstCharacterAlphabet;
// 	this.alphabet = alphabet;
// }

// public static final ConfigParameter<Integer> SCRAM_SHA_PASSWORD_LENGTH = new ConfigParameter<>("STRIMZI_SCRAM_SHA_PASSWORD_LENGTH", strictlyPositive(INTEGER), "32",  CONFIG_VALUES);

type passwordGenerator struct {
	reader          io.Reader
	length          int
	firstCharacters string
	alphabet        string
}

func newPasswordGenerator() *passwordGenerator {
	return &passwordGenerator{
		reader:          rand.Reader,
		length:          32,
		firstCharacters: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ",
		alphabet:        "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
	}
}

func (p *passwordGenerator) generate() (string, error) {
	var password strings.Builder
	nextIndex := func(length int) (int, error) {
		n, err := rand.Int(p.reader, big.NewInt(int64(length)))
		if err != nil {
			return -1, err
		}
		return int(n.Int64()), nil
	}

	index, err := nextIndex(len(p.firstCharacters))
	if err != nil {
		return "", err
	}
	password.WriteByte(p.firstCharacters[index])

	for i := 0; i < p.length; i++ {
		index, err := nextIndex(len(p.alphabet))
		if err != nil {
			return "", err
		}
		password.WriteByte(p.alphabet[index])
	}

	return password.String(), nil
}
