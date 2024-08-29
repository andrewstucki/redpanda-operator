// Copyright 2022 Redpanda Data, Inc.
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
	v1alpha2 "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha2"
)

// UserAuthenticationSpecApplyConfiguration represents an declarative configuration of the UserAuthenticationSpec type for use
// with apply.
type UserAuthenticationSpecApplyConfiguration struct {
	Type     *v1alpha2.SASLMechanism     `json:"type,omitempty"`
	Password *PasswordApplyConfiguration `json:"password,omitempty"`
}

// UserAuthenticationSpecApplyConfiguration constructs an declarative configuration of the UserAuthenticationSpec type for use with
// apply.
func UserAuthenticationSpec() *UserAuthenticationSpecApplyConfiguration {
	return &UserAuthenticationSpecApplyConfiguration{}
}

// WithType sets the Type field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Type field is set to the value of the last call.
func (b *UserAuthenticationSpecApplyConfiguration) WithType(value v1alpha2.SASLMechanism) *UserAuthenticationSpecApplyConfiguration {
	b.Type = &value
	return b
}

// WithPassword sets the Password field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Password field is set to the value of the last call.
func (b *UserAuthenticationSpecApplyConfiguration) WithPassword(value *PasswordApplyConfiguration) *UserAuthenticationSpecApplyConfiguration {
	b.Password = value
	return b
}
