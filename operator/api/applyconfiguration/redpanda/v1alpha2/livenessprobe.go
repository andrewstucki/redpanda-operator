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

// LivenessProbeApplyConfiguration represents an declarative configuration of the LivenessProbe type for use
// with apply.
type LivenessProbeApplyConfiguration struct {
	FailureThreshold    *int `json:"failureThreshold,omitempty"`
	InitialDelaySeconds *int `json:"initialDelaySeconds,omitempty"`
	PeriodSeconds       *int `json:"periodSeconds,omitempty"`
	TimeoutSeconds      *int `json:"timeoutSeconds,omitempty"`
	SuccessThreshold    *int `json:"successThreshold,omitempty"`
}

// LivenessProbeApplyConfiguration constructs an declarative configuration of the LivenessProbe type for use with
// apply.
func LivenessProbe() *LivenessProbeApplyConfiguration {
	return &LivenessProbeApplyConfiguration{}
}

// WithFailureThreshold sets the FailureThreshold field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the FailureThreshold field is set to the value of the last call.
func (b *LivenessProbeApplyConfiguration) WithFailureThreshold(value int) *LivenessProbeApplyConfiguration {
	b.FailureThreshold = &value
	return b
}

// WithInitialDelaySeconds sets the InitialDelaySeconds field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the InitialDelaySeconds field is set to the value of the last call.
func (b *LivenessProbeApplyConfiguration) WithInitialDelaySeconds(value int) *LivenessProbeApplyConfiguration {
	b.InitialDelaySeconds = &value
	return b
}

// WithPeriodSeconds sets the PeriodSeconds field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the PeriodSeconds field is set to the value of the last call.
func (b *LivenessProbeApplyConfiguration) WithPeriodSeconds(value int) *LivenessProbeApplyConfiguration {
	b.PeriodSeconds = &value
	return b
}

// WithTimeoutSeconds sets the TimeoutSeconds field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the TimeoutSeconds field is set to the value of the last call.
func (b *LivenessProbeApplyConfiguration) WithTimeoutSeconds(value int) *LivenessProbeApplyConfiguration {
	b.TimeoutSeconds = &value
	return b
}

// WithSuccessThreshold sets the SuccessThreshold field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the SuccessThreshold field is set to the value of the last call.
func (b *LivenessProbeApplyConfiguration) WithSuccessThreshold(value int) *LivenessProbeApplyConfiguration {
	b.SuccessThreshold = &value
	return b
}
