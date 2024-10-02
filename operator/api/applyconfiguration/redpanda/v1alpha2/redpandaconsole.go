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

// RedpandaConsoleApplyConfiguration represents an declarative configuration of the RedpandaConsole type for use
// with apply.
type RedpandaConsoleApplyConfiguration struct {
	Enabled                      *bool                               `json:"enabled,omitempty"`
	ReplicaCount                 *int                                `json:"replicaCount,omitempty"`
	NameOverride                 *string                             `json:"nameOverride,omitempty"`
	FullNameOverride             *string                             `json:"fullnameOverride,omitempty"`
	CommonLabels                 map[string]string                   `json:"commonLabels,omitempty"`
	PriorityClassName            *string                             `json:"priorityClassName,omitempty"`
	Image                        *runtime.RawExtension               `json:"image,omitempty"`
	ImagePullSecrets             []*runtime.RawExtension             `json:"imagePullSecrets,omitempty"`
	ServiceAccount               *runtime.RawExtension               `json:"serviceAccount,omitempty"`
	Annotations                  *runtime.RawExtension               `json:"annotations,omitempty"`
	PodAnnotations               *runtime.RawExtension               `json:"podAnnotations,omitempty"`
	PodLabels                    *runtime.RawExtension               `json:"podLabels,omitempty"`
	PodSecurityContext           *runtime.RawExtension               `json:"podSecurityContext,omitempty"`
	SecurityContext              *runtime.RawExtension               `json:"securityContext,omitempty"`
	Service                      *runtime.RawExtension               `json:"service,omitempty"`
	Ingress                      *runtime.RawExtension               `json:"ingress,omitempty"`
	Resources                    *runtime.RawExtension               `json:"resources,omitempty"`
	Autoscaling                  *runtime.RawExtension               `json:"autoscaling,omitempty"`
	NodeSelector                 *runtime.RawExtension               `json:"nodeSelector,omitempty"`
	Tolerations                  []*runtime.RawExtension             `json:"tolerations,omitempty"`
	Affinity                     *runtime.RawExtension               `json:"affinity,omitempty"`
	TopologySpreadConstraints    *runtime.RawExtension               `json:"topologySpreadConstraints,omitempty"`
	ExtraEnv                     []*runtime.RawExtension             `json:"extraEnv,omitempty"`
	ExtraEnvFrom                 []*runtime.RawExtension             `json:"extraEnvFrom,omitempty"`
	ExtraVolumes                 []*runtime.RawExtension             `json:"extraVolumes,omitempty"`
	ExtraVolumeMounts            []*runtime.RawExtension             `json:"extraVolumeMounts,omitempty"`
	ExtraContainers              []*runtime.RawExtension             `json:"extraContainers,omitempty"`
	InitContainers               *runtime.RawExtension               `json:"initContainers,omitempty"`
	SecretMounts                 []*runtime.RawExtension             `json:"secretMounts,omitempty"`
	DeprecatedConfigMap          *ConsoleCreateObjApplyConfiguration `json:"configmap,omitempty"`
	ConfigMap                    *ConsoleCreateObjApplyConfiguration `json:"configMap,omitempty"`
	Secret                       *runtime.RawExtension               `json:"secret,omitempty"`
	Deployment                   *runtime.RawExtension               `json:"deployment,omitempty"`
	Console                      *runtime.RawExtension               `json:"console,omitempty"`
	Strategy                     *runtime.RawExtension               `json:"strategy,omitempty"`
	Enterprise                   *runtime.RawExtension               `json:"enterprise,omitempty"`
	AutomountServiceAccountToken *bool                               `json:"automountServiceAccountToken,omitempty"`
	ReadinessProbe               *ReadinessProbeApplyConfiguration   `json:"readinessProbe,omitempty"`
	LivenessProbe                *LivenessProbeApplyConfiguration    `json:"livenessProbe,omitempty"`
	Tests                        *EnablableApplyConfiguration        `json:"tests,omitempty"`
}

// RedpandaConsoleApplyConfiguration constructs an declarative configuration of the RedpandaConsole type for use with
// apply.
func RedpandaConsole() *RedpandaConsoleApplyConfiguration {
	return &RedpandaConsoleApplyConfiguration{}
}

// WithEnabled sets the Enabled field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Enabled field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithEnabled(value bool) *RedpandaConsoleApplyConfiguration {
	b.Enabled = &value
	return b
}

// WithReplicaCount sets the ReplicaCount field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ReplicaCount field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithReplicaCount(value int) *RedpandaConsoleApplyConfiguration {
	b.ReplicaCount = &value
	return b
}

// WithNameOverride sets the NameOverride field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the NameOverride field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithNameOverride(value string) *RedpandaConsoleApplyConfiguration {
	b.NameOverride = &value
	return b
}

// WithFullNameOverride sets the FullNameOverride field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the FullNameOverride field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithFullNameOverride(value string) *RedpandaConsoleApplyConfiguration {
	b.FullNameOverride = &value
	return b
}

// WithCommonLabels puts the entries into the CommonLabels field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, the entries provided by each call will be put on the CommonLabels field,
// overwriting an existing map entries in CommonLabels field with the same key.
func (b *RedpandaConsoleApplyConfiguration) WithCommonLabels(entries map[string]string) *RedpandaConsoleApplyConfiguration {
	if b.CommonLabels == nil && len(entries) > 0 {
		b.CommonLabels = make(map[string]string, len(entries))
	}
	for k, v := range entries {
		b.CommonLabels[k] = v
	}
	return b
}

// WithPriorityClassName sets the PriorityClassName field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the PriorityClassName field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithPriorityClassName(value string) *RedpandaConsoleApplyConfiguration {
	b.PriorityClassName = &value
	return b
}

// WithImage sets the Image field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Image field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithImage(value runtime.RawExtension) *RedpandaConsoleApplyConfiguration {
	b.Image = &value
	return b
}

// WithImagePullSecrets adds the given value to the ImagePullSecrets field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the ImagePullSecrets field.
func (b *RedpandaConsoleApplyConfiguration) WithImagePullSecrets(values ...*runtime.RawExtension) *RedpandaConsoleApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithImagePullSecrets")
		}
		b.ImagePullSecrets = append(b.ImagePullSecrets, values[i])
	}
	return b
}

// WithServiceAccount sets the ServiceAccount field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ServiceAccount field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithServiceAccount(value runtime.RawExtension) *RedpandaConsoleApplyConfiguration {
	b.ServiceAccount = &value
	return b
}

// WithAnnotations sets the Annotations field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Annotations field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithAnnotations(value runtime.RawExtension) *RedpandaConsoleApplyConfiguration {
	b.Annotations = &value
	return b
}

// WithPodAnnotations sets the PodAnnotations field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the PodAnnotations field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithPodAnnotations(value runtime.RawExtension) *RedpandaConsoleApplyConfiguration {
	b.PodAnnotations = &value
	return b
}

// WithPodLabels sets the PodLabels field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the PodLabels field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithPodLabels(value runtime.RawExtension) *RedpandaConsoleApplyConfiguration {
	b.PodLabels = &value
	return b
}

// WithPodSecurityContext sets the PodSecurityContext field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the PodSecurityContext field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithPodSecurityContext(value runtime.RawExtension) *RedpandaConsoleApplyConfiguration {
	b.PodSecurityContext = &value
	return b
}

// WithSecurityContext sets the SecurityContext field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the SecurityContext field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithSecurityContext(value runtime.RawExtension) *RedpandaConsoleApplyConfiguration {
	b.SecurityContext = &value
	return b
}

// WithService sets the Service field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Service field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithService(value runtime.RawExtension) *RedpandaConsoleApplyConfiguration {
	b.Service = &value
	return b
}

// WithIngress sets the Ingress field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Ingress field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithIngress(value runtime.RawExtension) *RedpandaConsoleApplyConfiguration {
	b.Ingress = &value
	return b
}

// WithResources sets the Resources field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Resources field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithResources(value runtime.RawExtension) *RedpandaConsoleApplyConfiguration {
	b.Resources = &value
	return b
}

// WithAutoscaling sets the Autoscaling field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Autoscaling field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithAutoscaling(value runtime.RawExtension) *RedpandaConsoleApplyConfiguration {
	b.Autoscaling = &value
	return b
}

// WithNodeSelector sets the NodeSelector field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the NodeSelector field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithNodeSelector(value runtime.RawExtension) *RedpandaConsoleApplyConfiguration {
	b.NodeSelector = &value
	return b
}

// WithTolerations adds the given value to the Tolerations field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Tolerations field.
func (b *RedpandaConsoleApplyConfiguration) WithTolerations(values ...*runtime.RawExtension) *RedpandaConsoleApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithTolerations")
		}
		b.Tolerations = append(b.Tolerations, values[i])
	}
	return b
}

// WithAffinity sets the Affinity field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Affinity field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithAffinity(value runtime.RawExtension) *RedpandaConsoleApplyConfiguration {
	b.Affinity = &value
	return b
}

// WithTopologySpreadConstraints sets the TopologySpreadConstraints field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the TopologySpreadConstraints field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithTopologySpreadConstraints(value runtime.RawExtension) *RedpandaConsoleApplyConfiguration {
	b.TopologySpreadConstraints = &value
	return b
}

// WithExtraEnv adds the given value to the ExtraEnv field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the ExtraEnv field.
func (b *RedpandaConsoleApplyConfiguration) WithExtraEnv(values ...*runtime.RawExtension) *RedpandaConsoleApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithExtraEnv")
		}
		b.ExtraEnv = append(b.ExtraEnv, values[i])
	}
	return b
}

// WithExtraEnvFrom adds the given value to the ExtraEnvFrom field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the ExtraEnvFrom field.
func (b *RedpandaConsoleApplyConfiguration) WithExtraEnvFrom(values ...*runtime.RawExtension) *RedpandaConsoleApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithExtraEnvFrom")
		}
		b.ExtraEnvFrom = append(b.ExtraEnvFrom, values[i])
	}
	return b
}

// WithExtraVolumes adds the given value to the ExtraVolumes field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the ExtraVolumes field.
func (b *RedpandaConsoleApplyConfiguration) WithExtraVolumes(values ...*runtime.RawExtension) *RedpandaConsoleApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithExtraVolumes")
		}
		b.ExtraVolumes = append(b.ExtraVolumes, values[i])
	}
	return b
}

// WithExtraVolumeMounts adds the given value to the ExtraVolumeMounts field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the ExtraVolumeMounts field.
func (b *RedpandaConsoleApplyConfiguration) WithExtraVolumeMounts(values ...*runtime.RawExtension) *RedpandaConsoleApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithExtraVolumeMounts")
		}
		b.ExtraVolumeMounts = append(b.ExtraVolumeMounts, values[i])
	}
	return b
}

// WithExtraContainers adds the given value to the ExtraContainers field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the ExtraContainers field.
func (b *RedpandaConsoleApplyConfiguration) WithExtraContainers(values ...*runtime.RawExtension) *RedpandaConsoleApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithExtraContainers")
		}
		b.ExtraContainers = append(b.ExtraContainers, values[i])
	}
	return b
}

// WithInitContainers sets the InitContainers field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the InitContainers field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithInitContainers(value runtime.RawExtension) *RedpandaConsoleApplyConfiguration {
	b.InitContainers = &value
	return b
}

// WithSecretMounts adds the given value to the SecretMounts field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the SecretMounts field.
func (b *RedpandaConsoleApplyConfiguration) WithSecretMounts(values ...*runtime.RawExtension) *RedpandaConsoleApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithSecretMounts")
		}
		b.SecretMounts = append(b.SecretMounts, values[i])
	}
	return b
}

// WithDeprecatedConfigMap sets the DeprecatedConfigMap field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the DeprecatedConfigMap field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithDeprecatedConfigMap(value *ConsoleCreateObjApplyConfiguration) *RedpandaConsoleApplyConfiguration {
	b.DeprecatedConfigMap = value
	return b
}

// WithConfigMap sets the ConfigMap field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ConfigMap field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithConfigMap(value *ConsoleCreateObjApplyConfiguration) *RedpandaConsoleApplyConfiguration {
	b.ConfigMap = value
	return b
}

// WithSecret sets the Secret field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Secret field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithSecret(value runtime.RawExtension) *RedpandaConsoleApplyConfiguration {
	b.Secret = &value
	return b
}

// WithDeployment sets the Deployment field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Deployment field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithDeployment(value runtime.RawExtension) *RedpandaConsoleApplyConfiguration {
	b.Deployment = &value
	return b
}

// WithConsole sets the Console field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Console field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithConsole(value runtime.RawExtension) *RedpandaConsoleApplyConfiguration {
	b.Console = &value
	return b
}

// WithStrategy sets the Strategy field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Strategy field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithStrategy(value runtime.RawExtension) *RedpandaConsoleApplyConfiguration {
	b.Strategy = &value
	return b
}

// WithEnterprise sets the Enterprise field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Enterprise field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithEnterprise(value runtime.RawExtension) *RedpandaConsoleApplyConfiguration {
	b.Enterprise = &value
	return b
}

// WithAutomountServiceAccountToken sets the AutomountServiceAccountToken field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the AutomountServiceAccountToken field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithAutomountServiceAccountToken(value bool) *RedpandaConsoleApplyConfiguration {
	b.AutomountServiceAccountToken = &value
	return b
}

// WithReadinessProbe sets the ReadinessProbe field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ReadinessProbe field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithReadinessProbe(value *ReadinessProbeApplyConfiguration) *RedpandaConsoleApplyConfiguration {
	b.ReadinessProbe = value
	return b
}

// WithLivenessProbe sets the LivenessProbe field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the LivenessProbe field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithLivenessProbe(value *LivenessProbeApplyConfiguration) *RedpandaConsoleApplyConfiguration {
	b.LivenessProbe = value
	return b
}

// WithTests sets the Tests field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Tests field is set to the value of the last call.
func (b *RedpandaConsoleApplyConfiguration) WithTests(value *EnablableApplyConfiguration) *RedpandaConsoleApplyConfiguration {
	b.Tests = value
	return b
}
