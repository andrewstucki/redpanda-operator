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

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// ConfigApplyConfiguration represents an declarative configuration of the Config type for use
// with apply.
type ConfigApplyConfiguration struct {
	RPK                  *runtime.RawExtension `json:"rpk,omitempty"`
	Cluster              *runtime.RawExtension `json:"cluster,omitempty"`
	Node                 *runtime.RawExtension `json:"node,omitempty"`
	Tunable              *runtime.RawExtension `json:"tunable,omitempty"`
	SchemaRegistryClient *runtime.RawExtension `json:"schema_registry_client,omitempty"`
	PandaProxyClient     *runtime.RawExtension `json:"pandaproxy_client,omitempty"`
}

// ConfigApplyConfiguration constructs an declarative configuration of the Config type for use with
// apply.
func Config() *ConfigApplyConfiguration {
	return &ConfigApplyConfiguration{}
}

// WithRPK sets the RPK field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the RPK field is set to the value of the last call.
func (b *ConfigApplyConfiguration) WithRPK(value runtime.RawExtension) *ConfigApplyConfiguration {
	b.RPK = &value
	return b
}

// WithCluster sets the Cluster field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Cluster field is set to the value of the last call.
func (b *ConfigApplyConfiguration) WithCluster(value runtime.RawExtension) *ConfigApplyConfiguration {
	b.Cluster = &value
	return b
}

// WithNode sets the Node field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Node field is set to the value of the last call.
func (b *ConfigApplyConfiguration) WithNode(value runtime.RawExtension) *ConfigApplyConfiguration {
	b.Node = &value
	return b
}

// WithTunable sets the Tunable field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Tunable field is set to the value of the last call.
func (b *ConfigApplyConfiguration) WithTunable(value runtime.RawExtension) *ConfigApplyConfiguration {
	b.Tunable = &value
	return b
}

// WithSchemaRegistryClient sets the SchemaRegistryClient field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the SchemaRegistryClient field is set to the value of the last call.
func (b *ConfigApplyConfiguration) WithSchemaRegistryClient(value runtime.RawExtension) *ConfigApplyConfiguration {
	b.SchemaRegistryClient = &value
	return b
}

// WithPandaProxyClient sets the PandaProxyClient field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the PandaProxyClient field is set to the value of the last call.
func (b *ConfigApplyConfiguration) WithPandaProxyClient(value runtime.RawExtension) *ConfigApplyConfiguration {
	b.PandaProxyClient = &value
	return b
}
