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

// ExternalServiceApplyConfiguration represents an declarative configuration of the ExternalService type for use
// with apply.
type ExternalServiceApplyConfiguration struct {
	Enabled *bool `json:"enabled,omitempty"`
}

// ExternalServiceApplyConfiguration constructs an declarative configuration of the ExternalService type for use with
// apply.
func ExternalService() *ExternalServiceApplyConfiguration {
	return &ExternalServiceApplyConfiguration{}
}

// WithEnabled sets the Enabled field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Enabled field is set to the value of the last call.
func (b *ExternalServiceApplyConfiguration) WithEnabled(value bool) *ExternalServiceApplyConfiguration {
	b.Enabled = &value
	return b
}
