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

// ServiceInternalApplyConfiguration represents an declarative configuration of the ServiceInternal type for use
// with apply.
type ServiceInternalApplyConfiguration struct {
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ServiceInternalApplyConfiguration constructs an declarative configuration of the ServiceInternal type for use with
// apply.
func ServiceInternal() *ServiceInternalApplyConfiguration {
	return &ServiceInternalApplyConfiguration{}
}

// WithAnnotations puts the entries into the Annotations field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, the entries provided by each call will be put on the Annotations field,
// overwriting an existing map entries in Annotations field with the same key.
func (b *ServiceInternalApplyConfiguration) WithAnnotations(entries map[string]string) *ServiceInternalApplyConfiguration {
	if b.Annotations == nil && len(entries) > 0 {
		b.Annotations = make(map[string]string, len(entries))
	}
	for k, v := range entries {
		b.Annotations[k] = v
	}
	return b
}
