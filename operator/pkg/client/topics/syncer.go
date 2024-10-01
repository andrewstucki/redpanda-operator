// Copyright 2024 Redpanda Data, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.md
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0

package topics

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	redpandav1alpha2 "github.com/redpanda-data/redpanda-operator/operator/api/redpanda/v1alpha2"
	"github.com/redpanda-data/redpanda-operator/operator/pkg/collections"
	"github.com/redpanda-data/redpanda-operator/operator/pkg/functional"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
)

const (
	remoteReadKey  = "redpanda.remote.read"
	remoteWriteKey = "redpanda.remote.write"
	replicationKey = "replication.factor"
)

var (
	ErrEmptyTopicConfigDescription = errors.New("topic config description response is empty")
	ErrScaleDownPartitionCount     = errors.New("unable to scale down number of partition in topic")
)

// Syncer synchronizes Topics for the given object to Redpanda.
type Syncer struct {
	client *kgo.Client
}

// NewSyncer initializes a Syncer.
func NewSyncer(client *kgo.Client) *Syncer {
	return &Syncer{
		client: client,
	}
}

// Sync synchronizes the topic in Redpanda.
func (s *Syncer) Sync(ctx context.Context, o *redpandav1alpha2.Topic) error {
	config, configs, err := s.getTopicConfig(ctx, o)
	if err != nil {
		return err
	}

	if config == nil {
		return s.createTopic(ctx, o)
	}

	replicas, err := s.syncPartitions(ctx, o, config)
	if err != nil {
		return err
	}

	return s.syncConfigs(ctx, o, replicas, configs)
}

func (s *Syncer) syncPartitions(ctx context.Context, o *redpandav1alpha2.Topic, config *kmsg.MetadataResponseTopic) (int, error) {
	requestedPartitions := o.Spec.GetPartitions()
	actualPartitions := len(config.Partitions)

	// if we have no partitions, then the replication factor
	// will just be whatever the spec says, but if we already
	// have a partition with a replication factor, then use it
	// instead
	replicas := o.Spec.GetReplicationFactor()
	if len(config.Partitions) != 0 {
		replicas = len(config.Partitions[0].Replicas)
	}

	if requestedPartitions == actualPartitions {
		return replicas, nil
	}

	if requestedPartitions < actualPartitions {
		return 0, ErrScaleDownPartitionCount
	}

	partitionRequest := kmsg.NewCreatePartitionsRequest()
	partitionRequest.Topics = []kmsg.CreatePartitionsRequestTopic{{
		Topic: o.GetName(),
		Count: int32(requestedPartitions),
	}}

	partitionResponse, err := partitionRequest.RequestWith(ctx, s.client)
	if err != nil {
		return 0, err
	}

	for _, result := range partitionResponse.Topics {
		if err := checkError(result.ErrorMessage, result.ErrorCode); err != nil {
			return 0, err
		}
	}

	return replicas, nil
}

func (s *Syncer) syncConfigs(ctx context.Context, o *redpandav1alpha2.Topic, replicas int, configs []kmsg.DescribeConfigsResponseResourceConfig) error {
	doAlterations := func(alterations ...kmsg.IncrementalAlterConfigsRequestResourceConfig) error {
		alterConfigsRequest := kmsg.NewPtrIncrementalAlterConfigsRequest()
		alterConfigsRequest.Resources = []kmsg.IncrementalAlterConfigsRequestResource{{
			ResourceType: kmsg.ConfigResourceTypeTopic,
			ResourceName: o.GetTopicName(),
			Configs:      alterations,
		}}

		alterConfigsResponse, err := alterConfigsRequest.RequestWith(ctx, s.client)
		if err != nil {
			return err
		}

		for _, resource := range alterConfigsResponse.Resources {
			if err := checkError(resource.ErrorMessage, resource.ErrorCode); err != nil {
				return err
			}
		}

		return nil
	}

	toDelete := collections.NewSet[string]()

	// filter out all of the non-default configs other than cleanup.policy
	for _, conf := range configs {
		if conf.Source != kmsg.ConfigSourceDefaultConfig && conf.Value != nil && conf.Name != "cleanup.policy" {
			toDelete.Add(conf.Name)
		}
	}

	remoteWriteValue := o.Spec.AdditionalConfig[remoteWriteKey]
	remoteWrite := remoteWriteValue != nil
	remoteRead := o.Spec.AdditionalConfig[remoteReadKey] != nil
	writeFirst := remoteRead && remoteWrite
	if writeFirst {
		// we need to set write first if read is also specified
		err := doAlterations(kmsg.IncrementalAlterConfigsRequestResourceConfig{
			Name:  remoteWriteKey,
			Value: remoteWriteValue,
			Op:    kmsg.IncrementalAlterConfigOpSet,
		})
		if err != nil {
			return err
		}
	}

	additionalConfigs := functional.RejectKVs(func(k string, v *string) bool {
		toDelete.Delete(k)
		// filter out any key we already wrote
		return k == remoteReadKey && writeFirst
	}, o.Spec.AdditionalConfig)

	alterations := functional.MapKV(func(k string, v *string) kmsg.IncrementalAlterConfigsRequestResourceConfig {
		return kmsg.IncrementalAlterConfigsRequestResourceConfig{
			Name:  k,
			Value: v,
			Op:    kmsg.IncrementalAlterConfigOpSet,
		}
	}, additionalConfigs)

	for _, deletion := range toDelete.Values() {
		alterations = append(alterations, kmsg.IncrementalAlterConfigsRequestResourceConfig{
			Name: deletion,
			Op:   kmsg.IncrementalAlterConfigOpDelete,
		})
	}

	if replicas != o.Spec.GetReplicationFactor() {
		factor := strconv.Itoa(o.Spec.GetReplicationFactor())
		alterations = append(alterations, kmsg.IncrementalAlterConfigsRequestResourceConfig{
			Name:  replicationKey,
			Value: kmsg.StringPtr(factor),
		})
	}

	if len(alterations) == 0 {
		return nil
	}

	return doAlterations(alterations...)
}

func (s *Syncer) createTopic(ctx context.Context, o *redpandav1alpha2.Topic) error {
	req := kmsg.NewCreateTopicsRequest()
	req.Topics = []kmsg.CreateTopicsRequestTopic{{
		Topic:             o.GetTopicName(),
		NumPartitions:     int32(o.Spec.GetPartitions()),
		ReplicationFactor: int16(o.Spec.GetReplicationFactor()),
		Configs: functional.MapKV(func(k string, v *string) kmsg.CreateTopicsRequestTopicConfig {
			return kmsg.CreateTopicsRequestTopicConfig{
				Name:  k,
				Value: v,
			}
		}, o.Spec.AdditionalConfig),
	}}

	response, err := req.RequestWith(ctx, s.client)
	if err != nil {
		return err
	}

	if len(response.Topics) != 1 {
		return ErrEmptyTopicConfigDescription
	}

	for _, result := range response.Topics {
		if err := checkError(result.ErrorMessage, result.ErrorCode); err != nil {
			return err
		}
	}

	return nil
}

func (s *Syncer) getTopicConfig(ctx context.Context, o *redpandav1alpha2.Topic) (*kmsg.MetadataResponseTopic, []kmsg.DescribeConfigsResponseResourceConfig, error) {
	configRequest := &kmsg.DescribeConfigsRequest{
		Resources: []kmsg.DescribeConfigsRequestResource{{
			ResourceType: kmsg.ConfigResourceTypeTopic,
			ResourceName: o.GetTopicName(),
		}},
	}

	configResponse, err := configRequest.RequestWith(ctx, s.client)
	if err != nil {
		if errors.Is(err, kerr.UnknownTopicOrPartition) {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	if len(configResponse.Resources) != 1 {
		return nil, nil, ErrEmptyTopicConfigDescription
	}

	for _, result := range configResponse.Resources {
		if result.ErrorCode == kerr.UnknownTopicOrPartition.Code {
			return nil, nil, nil
		}
		if err := checkError(result.ErrorMessage, result.ErrorCode); err != nil {
			return nil, nil, err
		}
	}

	configs := configResponse.Resources[0].Configs

	metadataRequest := &kmsg.MetadataRequest{
		Topics: []kmsg.MetadataRequestTopic{{
			Topic: kmsg.StringPtr(o.GetTopicName()),
		}},
	}

	metadataResponse, err := metadataRequest.RequestWith(ctx, s.client)
	if err != nil {
		if errors.Is(err, kerr.UnknownTopicOrPartition) {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	if len(metadataResponse.Topics) != 1 {
		return nil, nil, ErrEmptyTopicConfigDescription
	}

	for _, result := range metadataResponse.Topics {
		if result.ErrorCode == kerr.UnknownTopicOrPartition.Code {
			return nil, nil, nil
		}
		if err := checkError(nil, result.ErrorCode); err != nil {
			return nil, nil, err
		}
	}

	topic := metadataResponse.Topics[0]

	return &topic, configs, nil
}

// Delete removes the topic from Redpanda.
func (s *Syncer) Delete(ctx context.Context, o *redpandav1alpha2.Topic) error {
	deletionRequest := kmsg.NewDeleteTopicsRequest()
	deletionRequest.TopicNames = []string{o.GetTopicName()}
	deletionRequest.Topics = []kmsg.DeleteTopicsRequestTopic{{
		Topic: kmsg.StringPtr(o.GetTopicName()),
	}}

	deletionResponse, err := deletionRequest.RequestWith(ctx, s.client)
	if err != nil {
		return err
	}

	if len(deletionResponse.Topics) == 0 {
		return ErrEmptyTopicConfigDescription
	}

	for _, result := range deletionResponse.Topics {
		if err := checkError(nil, result.ErrorCode); err != nil {
			return err
		}
	}

	return nil
}

func checkError(message *string, code int16) error {
	var errMessage string
	if message != nil {
		errMessage = "Error: " + *message + "; "
	}

	if code != 0 {
		return fmt.Errorf("%s%w", errMessage, kerr.ErrorForCode(code))
	}

	return nil
}
