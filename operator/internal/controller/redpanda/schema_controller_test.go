// Copyright 2024 Redpanda Data, Inc.
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
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/redpanda"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	redpandav1alpha2 "github.com/redpanda-data/redpanda-operator/operator/api/redpanda/v1alpha2"
	"github.com/redpanda-data/redpanda-operator/operator/internal/testutils"
	internalclient "github.com/redpanda-data/redpanda-operator/operator/pkg/client"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func TestSchemaReconcile(t *testing.T) { // nolint:funlen // These tests have clear subtests.
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	server := &envtest.APIServer{}
	etcd := &envtest.Etcd{}

	testEnv := testutils.RedpandaTestEnv{
		Environment: envtest.Environment{
			ControlPlane: envtest.ControlPlane{
				APIServer: server,
				Etcd:      etcd,
			},
		},
	}
	cfg, err := testEnv.StartRedpandaTestEnv(false)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	t.Cleanup(func() {
		_ = testEnv.Stop()
	})

	container, err := redpanda.Run(ctx, "docker.redpanda.com/redpandadata/redpanda:v23.2.8",
		redpanda.WithEnableSchemaRegistryHTTPBasicAuth(),
		redpanda.WithEnableKafkaAuthorization(),
		redpanda.WithEnableSASL(),
		redpanda.WithSuperusers("superuser"),
		redpanda.WithNewServiceAccount("superuser", "password"),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
	})

	schemaRegistry, err := container.SchemaRegistryAddress(ctx)
	require.NoError(t, err)

	err = redpandav1alpha2.AddToScheme(scheme.Scheme)
	require.NoError(t, err)

	c, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	require.NoError(t, err)
	require.NotNil(t, c)

	factory := internalclient.NewFactory(cfg, c)

	// ensure we have a secret which we can pull a password from
	err = c.Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "superuser",
			Namespace: metav1.NamespaceDefault,
		},
		Data: map[string][]byte{
			"password": []byte("password"),
		},
	})
	require.NoError(t, err)

	err = c.Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "invalidsuperuser",
			Namespace: metav1.NamespaceDefault,
		},
		Data: map[string][]byte{
			"password": []byte("invalid"),
		},
	})
	require.NoError(t, err)

	reconciler := SchemaReconciler{
		Client:        c,
		ClientFactory: factory,
	}

	validClusterSource := &redpandav1alpha2.ClusterSource{
		StaticConfiguration: &redpandav1alpha2.StaticConfigurationSource{
			SchemaRegistry: &redpandav1alpha2.SchemaRegistrySpec{
				URLs: []string{schemaRegistry},
				SASL: &redpandav1alpha2.SchemaRegistrySASL{
					Username: "superuser",
					Password: redpandav1alpha2.SecretKeyRef{
						Name: "superuser",
						Key:  "password",
					},
					Mechanism: redpandav1alpha2.SASLMechanismScramSHA256,
				},
			},
		},
	}

	invalidAuthClusterSourceBadPassword := &redpandav1alpha2.ClusterSource{
		StaticConfiguration: &redpandav1alpha2.StaticConfigurationSource{
			SchemaRegistry: &redpandav1alpha2.SchemaRegistrySpec{
				URLs: []string{schemaRegistry},
				SASL: &redpandav1alpha2.SchemaRegistrySASL{
					Username: "superuser",
					Password: redpandav1alpha2.SecretKeyRef{
						Name: "invalidsuperuser",
						Key:  "password",
					},
					Mechanism: redpandav1alpha2.SASLMechanismScramSHA256,
				},
			},
		},
	}

	invalidAuthClusterSourceNoSASL := &redpandav1alpha2.ClusterSource{
		StaticConfiguration: &redpandav1alpha2.StaticConfigurationSource{
			SchemaRegistry: &redpandav1alpha2.SchemaRegistrySpec{
				URLs: []string{schemaRegistry},
			},
		},
	}

	invalidClusterRefSource := &redpandav1alpha2.ClusterSource{
		ClusterRef: &redpandav1alpha2.ClusterRef{
			Name: "nonexistent",
		},
	}

	baseSchema := &redpandav1alpha2.Schema{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: metav1.NamespaceDefault,
		},
		Spec: redpandav1alpha2.SchemaSpec{
			ClusterSource: validClusterSource,
			Text: `{
			"type": "record",
			"name": "test",
			"fields":
				[
				{
					"type": "string",
					"name": "field1"
				},
				{
					"type": "int",
					"name": "field2"
				}
				]
			}`,
		},
	}

	syncedClusterRefCondition := redpandav1alpha2.SchemaSyncedCondition("test")

	invalidClusterRefCondition := redpandav1alpha2.SchemaNotSyncedCondition(
		redpandav1alpha2.SchemaConditionReasonClusterRefInvalid, errors.New("test"),
	)

	clientErrorCondition := redpandav1alpha2.SchemaNotSyncedCondition(
		redpandav1alpha2.SchemaConditionReasonTerminalClientError, errors.New("test"),
	)

	for name, tt := range map[string]struct {
		mutate            func(schema *redpandav1alpha2.Schema)
		expectedCondition metav1.Condition
	}{
		"success": {
			expectedCondition: syncedClusterRefCondition,
		},
		"error - invalid cluster ref": {
			mutate: func(schema *redpandav1alpha2.Schema) {
				schema.Spec.ClusterSource = invalidClusterRefSource
			},
			expectedCondition: invalidClusterRefCondition,
		},
		"error - client error no SASL": {
			mutate: func(schema *redpandav1alpha2.Schema) {
				schema.Spec.ClusterSource = invalidAuthClusterSourceNoSASL
			},
			expectedCondition: clientErrorCondition,
		},
		"error - client error invalid credentials": {
			mutate: func(schema *redpandav1alpha2.Schema) {
				schema.Spec.ClusterSource = invalidAuthClusterSourceBadPassword
			},
			expectedCondition: clientErrorCondition,
		},
	} {
		t.Run(name, func(t *testing.T) {
			schema := baseSchema.DeepCopy()
			schema.Name = "schema" + strconv.Itoa(int(time.Now().UnixNano()))

			if tt.mutate != nil {
				tt.mutate(schema)
			}

			key := client.ObjectKeyFromObject(schema)
			req := ctrl.Request{NamespacedName: key}

			require.NoError(t, c.Create(ctx, schema))
			_, err = reconciler.Reconcile(ctx, req)
			require.NoError(t, err)

			require.NoError(t, c.Get(ctx, key, schema))
			require.Equal(t, []string{FinalizerKey}, schema.Finalizers)
			require.Len(t, schema.Status.Conditions, 1)
			require.Equal(t, tt.expectedCondition.Type, schema.Status.Conditions[0].Type)
			require.Equal(t, tt.expectedCondition.Reason, schema.Status.Conditions[0].Reason)
			require.Equal(t, tt.expectedCondition.Status, schema.Status.Conditions[0].Status)

			if tt.expectedCondition.Status == metav1.ConditionTrue { //nolint:nestif // ignore
				schemaClient, err := factory.SchemaRegistryClient(ctx, schema)
				require.NoError(t, err)
				require.NotNil(t, schemaClient)

				_, err = schemaClient.SchemaByVersion(ctx, schema.Name, -1)
				require.NoError(t, err)

				// clean up and make sure we properly delete everything
				require.NoError(t, c.Delete(ctx, schema))
				_, err = reconciler.Reconcile(ctx, req)
				require.NoError(t, err)
				require.True(t, apierrors.IsNotFound(c.Get(ctx, key, schema)))

				// make sure we no longer have a schema
				_, err = schemaClient.SchemaByVersion(ctx, schema.Name, -1)
				require.EqualError(t, err, fmt.Sprintf("Subject '%s' not found.", schema.Name))

				return
			}

			// clean up and make sure we properly delete everything
			require.NoError(t, c.Delete(ctx, schema))
			_, err = reconciler.Reconcile(ctx, req)
			require.NoError(t, err)
			require.True(t, apierrors.IsNotFound(c.Get(ctx, key, schema)))
		})
	}
}
