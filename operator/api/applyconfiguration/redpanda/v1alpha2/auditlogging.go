// Copyright 2024 Redpanda Data, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.md
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha2

// AuditLoggingApplyConfiguration represents an declarative configuration of the AuditLogging type for use
// with apply.
type AuditLoggingApplyConfiguration struct {
	Enabled                    *bool    `json:"enabled,omitempty"`
	Listener                   *string  `json:"listener,omitempty"`
	Partitions                 *int     `json:"partitions,omitempty"`
	EnabledEventTypes          []string `json:"enabledEventTypes,omitempty"`
	ExcludedTopics             []string `json:"excludedTopics,omitempty"`
	ExcludedPrincipals         []string `json:"excludedPrincipals,omitempty"`
	ClientMaxBufferSize        *int     `json:"clientMaxBufferSize,omitempty"`
	QueueDrainIntervalMs       *int     `json:"queueDrainIntervalMs,omitempty"`
	QueueMaxBufferSizePerShard *int     `json:"queueMaxBufferSizePerShard,omitempty"`
	ReplicationFactor          *int     `json:"replicationFactor,omitempty"`
}

// AuditLoggingApplyConfiguration constructs an declarative configuration of the AuditLogging type for use with
// apply.
func AuditLogging() *AuditLoggingApplyConfiguration {
	return &AuditLoggingApplyConfiguration{}
}

// WithEnabled sets the Enabled field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Enabled field is set to the value of the last call.
func (b *AuditLoggingApplyConfiguration) WithEnabled(value bool) *AuditLoggingApplyConfiguration {
	b.Enabled = &value
	return b
}

// WithListener sets the Listener field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Listener field is set to the value of the last call.
func (b *AuditLoggingApplyConfiguration) WithListener(value string) *AuditLoggingApplyConfiguration {
	b.Listener = &value
	return b
}

// WithPartitions sets the Partitions field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Partitions field is set to the value of the last call.
func (b *AuditLoggingApplyConfiguration) WithPartitions(value int) *AuditLoggingApplyConfiguration {
	b.Partitions = &value
	return b
}

// WithEnabledEventTypes adds the given value to the EnabledEventTypes field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the EnabledEventTypes field.
func (b *AuditLoggingApplyConfiguration) WithEnabledEventTypes(values ...string) *AuditLoggingApplyConfiguration {
	for i := range values {
		b.EnabledEventTypes = append(b.EnabledEventTypes, values[i])
	}
	return b
}

// WithExcludedTopics adds the given value to the ExcludedTopics field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the ExcludedTopics field.
func (b *AuditLoggingApplyConfiguration) WithExcludedTopics(values ...string) *AuditLoggingApplyConfiguration {
	for i := range values {
		b.ExcludedTopics = append(b.ExcludedTopics, values[i])
	}
	return b
}

// WithExcludedPrincipals adds the given value to the ExcludedPrincipals field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the ExcludedPrincipals field.
func (b *AuditLoggingApplyConfiguration) WithExcludedPrincipals(values ...string) *AuditLoggingApplyConfiguration {
	for i := range values {
		b.ExcludedPrincipals = append(b.ExcludedPrincipals, values[i])
	}
	return b
}

// WithClientMaxBufferSize sets the ClientMaxBufferSize field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ClientMaxBufferSize field is set to the value of the last call.
func (b *AuditLoggingApplyConfiguration) WithClientMaxBufferSize(value int) *AuditLoggingApplyConfiguration {
	b.ClientMaxBufferSize = &value
	return b
}

// WithQueueDrainIntervalMs sets the QueueDrainIntervalMs field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the QueueDrainIntervalMs field is set to the value of the last call.
func (b *AuditLoggingApplyConfiguration) WithQueueDrainIntervalMs(value int) *AuditLoggingApplyConfiguration {
	b.QueueDrainIntervalMs = &value
	return b
}

// WithQueueMaxBufferSizePerShard sets the QueueMaxBufferSizePerShard field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the QueueMaxBufferSizePerShard field is set to the value of the last call.
func (b *AuditLoggingApplyConfiguration) WithQueueMaxBufferSizePerShard(value int) *AuditLoggingApplyConfiguration {
	b.QueueMaxBufferSizePerShard = &value
	return b
}

// WithReplicationFactor sets the ReplicationFactor field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ReplicationFactor field is set to the value of the last call.
func (b *AuditLoggingApplyConfiguration) WithReplicationFactor(value int) *AuditLoggingApplyConfiguration {
	b.ReplicationFactor = &value
	return b
}
