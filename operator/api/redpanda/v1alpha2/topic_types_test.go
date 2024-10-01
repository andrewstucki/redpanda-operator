// Copyright 2024 Redpanda Data, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.md
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0

package v1alpha2

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/redpanda-data/redpanda-operator/operator/internal/testutils"
)

func TestTopicValidation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	testEnv := testutils.RedpandaTestEnv{}
	cfg, err := testEnv.StartRedpandaTestEnv(false)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	baseTopic := Topic{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "name",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: TopicSpec{
			ClusterSource: &ClusterSource{
				ClusterRef: &ClusterRef{
					Name: "cluster",
				},
			},
		},
	}

	err = AddToScheme(scheme.Scheme)
	require.NoError(t, err)

	c, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	require.NoError(t, err)
	require.NotNil(t, c)

	for name, tt := range map[string]validationTestCase[*Topic]{
		"basic create": {},
		// connection params
		"clusterRef or kafkaApiSpec - no cluster source": {
			mutate: func(topic *Topic) {
				topic.Spec.ClusterSource = nil
			},
			errors: []string{`cluster must be specified if kafkaApiSpec is not`},
		},
		"clusterRef or kafkaApiSpec - none": {
			mutate: func(topic *Topic) {
				topic.Spec.ClusterSource.ClusterRef = nil
			},
			errors: []string{`either clusterref or staticconfiguration must be set`},
		},
		"clusterRef or kafkaApiSpec - admin api spec": {
			mutate: func(topic *Topic) {
				topic.Spec.ClusterSource.ClusterRef = nil
				topic.Spec.ClusterSource.StaticConfiguration = &StaticConfigurationSource{}
			},
			errors: []string{`spec.cluster.staticconfiguration.kafka: required value`},
		},
		"clusterRef or kafkaApiSpec - kafka spec": {
			mutate: func(topic *Topic) {
				topic.Spec.ClusterSource.ClusterRef = nil
				topic.Spec.ClusterSource.StaticConfiguration = &StaticConfigurationSource{
					Kafka: &KafkaAPISpec{
						Brokers: []string{"1.2.3.4:0"},
					},
				}
			},
		},
		"deprecated kafkaApiSpec": {
			mutate: func(topic *Topic) {
				topic.Spec.ClusterSource = nil
				topic.Spec.KafkaAPISpec = &KafkaAPISpec{
					Brokers: []string{"1.2.3.4:0"},
				}
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			runValidationTest(ctx, t, tt, c, &baseTopic)
		})
	}

	t.Run("increase-only partitions", func(t *testing.T) {
		objectCopy := baseTopic.DeepCopy()
		objectCopy.SetName("increase-only")
		objectCopy.Spec.Partitions = ptr.To(3)

		require.NoError(t, c.Create(ctx, objectCopy))

		expectedMessage := "decreasing partitions is not allowed"
		objectCopy.Spec.Partitions = ptr.To(1)

		err := c.Update(ctx, objectCopy)
		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(expectedMessage))

		expectedMessage = "partitions cannot be unset once set"
		objectCopy.Spec.Partitions = nil

		err = c.Update(ctx, objectCopy)
		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(expectedMessage))
	})

	t.Run("decrease-only replication", func(t *testing.T) {
		objectCopy := baseTopic.DeepCopy()
		objectCopy.SetName("decrease-only")
		objectCopy.Spec.ReplicationFactor = ptr.To(1)

		require.NoError(t, c.Create(ctx, objectCopy))

		expectedMessage := "increasing replicationFactor is not allowed"
		objectCopy.Spec.ReplicationFactor = ptr.To(2)

		err := c.Update(ctx, objectCopy)
		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(expectedMessage))

		expectedMessage = "replicationFactor cannot be unset once set"
		objectCopy.Spec.ReplicationFactor = nil

		err = c.Update(ctx, objectCopy)
		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(expectedMessage))
	})
}
