package testing

import (
	"context"

	redpandav1alpha1 "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha1"
	redpandav1alpha2 "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Client returns a new controllerruntime client.
func Client(options ...*KubectlOptions) (client.Client, error) {
	restConfig, err := RestConfig(options...)
	if err != nil {
		return nil, err
	}

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := redpandav1alpha1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := redpandav1alpha2.AddToScheme(scheme); err != nil {
		return nil, err
	}

	return client.New(restConfig, client.Options{Scheme: scheme})
}

// CreateNamespace creates the given namespace.
func CreateNamespace(ctx context.Context, namespace string, options ...*KubectlOptions) error {
	client, err := Client(options...)
	if err != nil {
		return err
	}

	return client.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	})
}

// DeleteNamespace deletes the given namespace.
func DeleteNamespace(ctx context.Context, namespace string, options ...*KubectlOptions) error {
	client, err := Client(options...)
	if err != nil {
		return err
	}

	return client.Delete(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	})
}

// RestConfig returns the kubernetes configuration given the KubectlOptions.
func RestConfig(options ...*KubectlOptions) (*rest.Config, error) {
	mergedOptions := defaultOptions()
	for _, option := range options {
		mergedOptions = mergedOptions.merge(option)
	}

	loading := clientcmd.NewDefaultClientConfigLoadingRules()
	loading.Precedence = append([]string{mergedOptions.ConfigPath}, loading.Precedence...)
	config := clientcmd.NewInteractiveDeferredLoadingClientConfig(loading, &clientcmd.ConfigOverrides{}, nil)

	return config.ClientConfig()
}
