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

// ExternalListenerApplyConfiguration represents an declarative configuration of the ExternalListener type for use
// with apply.
type ExternalListenerApplyConfiguration struct {
	Enabled              *bool                          `json:"enabled,omitempty"`
	AuthenticationMethod *string                        `json:"authenticationMethod,omitempty"`
	Port                 *int                           `json:"port,omitempty"`
	TLS                  *ListenerTLSApplyConfiguration `json:"tls,omitempty"`
	AdvertisedPorts      []int                          `json:"advertisedPorts,omitempty"`
	PrefixTemplate       *string                        `json:"prefixTemplate,omitempty"`
	NodePort             *int32                         `json:"nodePort,omitempty"`
}

// ExternalListenerApplyConfiguration constructs an declarative configuration of the ExternalListener type for use with
// apply.
func ExternalListener() *ExternalListenerApplyConfiguration {
	return &ExternalListenerApplyConfiguration{}
}

// WithEnabled sets the Enabled field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Enabled field is set to the value of the last call.
func (b *ExternalListenerApplyConfiguration) WithEnabled(value bool) *ExternalListenerApplyConfiguration {
	b.Enabled = &value
	return b
}

// WithAuthenticationMethod sets the AuthenticationMethod field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the AuthenticationMethod field is set to the value of the last call.
func (b *ExternalListenerApplyConfiguration) WithAuthenticationMethod(value string) *ExternalListenerApplyConfiguration {
	b.AuthenticationMethod = &value
	return b
}

// WithPort sets the Port field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Port field is set to the value of the last call.
func (b *ExternalListenerApplyConfiguration) WithPort(value int) *ExternalListenerApplyConfiguration {
	b.Port = &value
	return b
}

// WithTLS sets the TLS field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the TLS field is set to the value of the last call.
func (b *ExternalListenerApplyConfiguration) WithTLS(value *ListenerTLSApplyConfiguration) *ExternalListenerApplyConfiguration {
	b.TLS = value
	return b
}

// WithAdvertisedPorts adds the given value to the AdvertisedPorts field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the AdvertisedPorts field.
func (b *ExternalListenerApplyConfiguration) WithAdvertisedPorts(values ...int) *ExternalListenerApplyConfiguration {
	for i := range values {
		b.AdvertisedPorts = append(b.AdvertisedPorts, values[i])
	}
	return b
}

// WithPrefixTemplate sets the PrefixTemplate field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the PrefixTemplate field is set to the value of the last call.
func (b *ExternalListenerApplyConfiguration) WithPrefixTemplate(value string) *ExternalListenerApplyConfiguration {
	b.PrefixTemplate = &value
	return b
}

// WithNodePort sets the NodePort field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the NodePort field is set to the value of the last call.
func (b *ExternalListenerApplyConfiguration) WithNodePort(value int32) *ExternalListenerApplyConfiguration {
	b.NodePort = &value
	return b
}
