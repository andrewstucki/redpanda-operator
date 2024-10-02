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

// MonitoringApplyConfiguration represents an declarative configuration of the Monitoring type for use
// with apply.
type MonitoringApplyConfiguration struct {
	Enabled        *bool                 `json:"enabled,omitempty"`
	Labels         map[string]string     `json:"labels,omitempty"`
	ScrapeInterval *string               `json:"scrapeInterval,omitempty"`
	TLSConfig      *runtime.RawExtension `json:"tlsConfig,omitempty"`
	EnableHTTP2    *bool                 `json:"enableHttp2,omitempty"`
}

// MonitoringApplyConfiguration constructs an declarative configuration of the Monitoring type for use with
// apply.
func Monitoring() *MonitoringApplyConfiguration {
	return &MonitoringApplyConfiguration{}
}

// WithEnabled sets the Enabled field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Enabled field is set to the value of the last call.
func (b *MonitoringApplyConfiguration) WithEnabled(value bool) *MonitoringApplyConfiguration {
	b.Enabled = &value
	return b
}

// WithLabels puts the entries into the Labels field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, the entries provided by each call will be put on the Labels field,
// overwriting an existing map entries in Labels field with the same key.
func (b *MonitoringApplyConfiguration) WithLabels(entries map[string]string) *MonitoringApplyConfiguration {
	if b.Labels == nil && len(entries) > 0 {
		b.Labels = make(map[string]string, len(entries))
	}
	for k, v := range entries {
		b.Labels[k] = v
	}
	return b
}

// WithScrapeInterval sets the ScrapeInterval field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ScrapeInterval field is set to the value of the last call.
func (b *MonitoringApplyConfiguration) WithScrapeInterval(value string) *MonitoringApplyConfiguration {
	b.ScrapeInterval = &value
	return b
}

// WithTLSConfig sets the TLSConfig field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the TLSConfig field is set to the value of the last call.
func (b *MonitoringApplyConfiguration) WithTLSConfig(value runtime.RawExtension) *MonitoringApplyConfiguration {
	b.TLSConfig = &value
	return b
}

// WithEnableHTTP2 sets the EnableHTTP2 field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the EnableHTTP2 field is set to the value of the last call.
func (b *MonitoringApplyConfiguration) WithEnableHTTP2(value bool) *MonitoringApplyConfiguration {
	b.EnableHTTP2 = &value
	return b
}
