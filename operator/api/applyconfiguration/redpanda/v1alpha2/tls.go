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
	v1alpha2 "github.com/redpanda-data/redpanda-operator/operator/api/redpanda/v1alpha2"
)

// TLSApplyConfiguration represents an declarative configuration of the TLS type for use
// with apply.
type TLSApplyConfiguration struct {
	Certs   map[string]*v1alpha2.Certificate `json:"certs,omitempty"`
	Enabled *bool                            `json:"enabled,omitempty"`
}

// TLSApplyConfiguration constructs an declarative configuration of the TLS type for use with
// apply.
func TLS() *TLSApplyConfiguration {
	return &TLSApplyConfiguration{}
}

// WithCerts puts the entries into the Certs field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, the entries provided by each call will be put on the Certs field,
// overwriting an existing map entries in Certs field with the same key.
func (b *TLSApplyConfiguration) WithCerts(entries map[string]*v1alpha2.Certificate) *TLSApplyConfiguration {
	if b.Certs == nil && len(entries) > 0 {
		b.Certs = make(map[string]*v1alpha2.Certificate, len(entries))
	}
	for k, v := range entries {
		b.Certs[k] = v
	}
	return b
}

// WithEnabled sets the Enabled field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Enabled field is set to the value of the last call.
func (b *TLSApplyConfiguration) WithEnabled(value bool) *TLSApplyConfiguration {
	b.Enabled = &value
	return b
}
