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
	"slices"
	"testing"
	"time"

	redpandav1alpha2 "github.com/redpanda-data/redpanda-operator/operator/api/redpanda/v1alpha2"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestTopicController(t *testing.T) { // nolint:funlen // These tests have clear subtests.
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	timeoutOption := kgo.RetryTimeout(1 * time.Millisecond)
	environment := InitializeResourceReconcilerTest(t, ctx, &TopicReconciler{
		extraOptions: []kgo.Opt{timeoutOption},
	})

	baseTopic := &redpandav1alpha2.Topic{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: metav1.NamespaceDefault,
		},
		Spec: redpandav1alpha2.TopicSpec{
			ClusterSource:     environment.ClusterSourceValid,
			Partitions:        ptr.To(3),
			ReplicationFactor: ptr.To(1),
		},
	}

	fetchTopicMetadata := func(t *testing.T, o *redpandav1alpha2.Topic) kmsg.MetadataResponseTopic {
		client, err := environment.Factory.KafkaClient(ctx, o, timeoutOption)
		require.NoError(t, err)

		metaRequest := kmsg.NewPtrMetadataRequest()
		metaRequest.Topics = []kmsg.MetadataRequestTopic{{
			Topic: kmsg.StringPtr(o.GetTopicName()),
		}}
		response, err := metaRequest.RequestWith(ctx, client)
		require.NoError(t, err)

		return response.Topics[0]
	}

	expectTopicReconciles := func(t *testing.T, topic *redpandav1alpha2.Topic, create bool) {
		key := client.ObjectKeyFromObject(topic)
		req := ctrl.Request{NamespacedName: key}

		if create {
			require.NoError(t, environment.Factory.Client.Create(ctx, topic))
		} else {
			require.NoError(t, environment.Factory.Client.Update(ctx, topic))
		}

		_, err := environment.Reconciler.Reconcile(ctx, req)
		require.NoError(t, err)

		metadata := fetchTopicMetadata(t, topic)

		require.Len(t, metadata.Partitions, topic.Spec.GetPartitions())
		require.Len(t, metadata.Partitions[0].Replicas, topic.Spec.GetReplicationFactor())

		require.NoError(t, environment.Factory.Client.Get(ctx, key, topic))

		require.Contains(t, topic.Finalizers, FinalizerKey)

		require.Len(t, topic.Status.Conditions, 1)
		condition := topic.Status.Conditions[0]
		require.Equal(t, redpandav1alpha2.ResourceConditionTypeSynced, condition.Type)
		require.Equal(t, metav1.ConditionTrue, condition.Status)
		require.Equal(t, redpandav1alpha2.ResourceConditionReasonSynced, condition.Reason)
	}

	expectTopicConfigurationMatches := func(t *testing.T, topic *redpandav1alpha2.Topic, keys ...string) {
		client, err := environment.Factory.KafkaClient(ctx, topic, timeoutOption)
		require.NoError(t, err)

		adminClient := kadm.NewClient(client)
		configs, err := adminClient.DescribeTopicConfigs(ctx, topic.GetTopicName())
		require.NoError(t, err)

		config, err := configs.On(topic.GetTopicName(), nil)
		require.NoError(t, err)

		for _, c := range config.Configs {
			if !slices.Contains(keys, c.Key) {
				continue
			}
			value, exist := topic.Spec.AdditionalConfig[c.Key]
			require.True(t, exist)

			require.NotNil(t, c.Value)
			require.Equal(t, *value, *c.Value)
		}
	}

	t.Run("create_topic", func(t *testing.T) {
		topic := baseTopic.DeepCopy()
		topic.Name = "create-topic"

		expectTopicReconciles(t, topic, true)
	})

	t.Run("overwrite_topic", func(t *testing.T) {
		topic := baseTopic.DeepCopy()
		topic.Name = "overwrite-topic"
		topic.Spec.OverwriteTopicName = ptr.To("different_name_with_underscore")

		expectTopicReconciles(t, topic, true)
	})

	t.Run("create_topic_that_already_exist", func(t *testing.T) {
		topic := baseTopic.DeepCopy()
		topic.Name = "create-already-existent-test-topic"

		kafkaClient, err := environment.Factory.KafkaClient(ctx, topic)
		require.NoError(t, err)
		adminClient := kadm.NewClient(kafkaClient)

		_, err = adminClient.CreateTopic(ctx, -1, -1, nil, topic.Name)
		require.NoError(t, err)

		expectTopicReconciles(t, topic, true)
	})

	t.Run("add_partition", func(t *testing.T) {
		topic := baseTopic.DeepCopy()
		topic.Name = "partition-count-change"
		topic.Spec.AdditionalConfig = map[string]*string{
			"segment.bytes": ptr.To("7654321"),
		}

		expectTopicReconciles(t, topic, true)

		topic.Spec.Partitions = ptr.To(6)

		expectTopicReconciles(t, topic, false)
	})

	t.Run("delete_a_key`s_config_value", func(t *testing.T) {
		testPropertyKey := "max.message.bytes"
		testPropertyValue := "87678987"

		topic := baseTopic.DeepCopy()
		topic.Name = "remote-topic-property"
		topic.Spec.AdditionalConfig = map[string]*string{
			"segment.bytes": ptr.To("7654321"),
			testPropertyKey: &testPropertyValue,
		}

		expectTopicReconciles(t, topic, true)

		delete(topic.Spec.AdditionalConfig, testPropertyKey)

		expectTopicReconciles(t, topic, false)
	})

	t.Run("both_tiered_storage_property", func(t *testing.T) {
		// Redpanda fails to set both remote.read and write when passed
		// at the same time, so we issue first the set request for write,
		// then the rest of the requests.
		// See https://github.com/redpanda-data/redpanda/issues/9191 and
		// https://github.com/redpanda-data/redpanda/issues/4499
		topic := baseTopic.DeepCopy()
		topic.Name = "both-tiered-storage-conf" // nolint:gosec // this is not credentials
		topic.Spec.AdditionalConfig = map[string]*string{
			"redpanda.remote.read":  ptr.To("true"),
			"redpanda.remote.write": ptr.To("true"),
			"segment.bytes":         ptr.To("7654321"),
		}

		expectTopicReconciles(t, topic, true)
		expectTopicConfigurationMatches(t, topic, "redpanda.remote.read", "redpanda.remote.write")

		topic.Spec.AdditionalConfig["redpanda.remote.read"] = ptr.To("false")
		topic.Spec.AdditionalConfig["redpanda.remote.write"] = ptr.To("false")

		expectTopicReconciles(t, topic, false)
		expectTopicConfigurationMatches(t, topic)
	})
}
