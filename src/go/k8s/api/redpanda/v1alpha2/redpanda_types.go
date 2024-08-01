// Copyright 2024 Redpanda Data, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.md
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0

package v1alpha2

import (
	"encoding/json"
	"fmt"

	helmv2beta2 "github.com/fluxcd/helm-controller/api/v2beta2"
	"github.com/fluxcd/pkg/apis/meta"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/redpanda-data/redpanda-operator/src/go/k8s/api/vectorized/v1alpha1"
)

var RedpandaChartRepository = "https://charts.redpanda.com/"

type ChartRef struct {
	// Specifies the name of the chart to deploy.
	ChartName string `json:"chartName,omitempty"`
	// Defines the version of the Redpanda Helm chart to deploy.
	ChartVersion string `json:"chartVersion,omitempty"`
	// Defines the chart repository to use. Defaults to `redpanda` if not defined.
	HelmRepositoryName string `json:"helmRepositoryName,omitempty"`
	// Specifies the time to wait for any individual Kubernetes operation (like Jobs
	// for hooks) during Helm actions. Defaults to `15m0s`.
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern="^([0-9]+(\\.[0-9]+)?(ms|s|m|h))+$"
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`
	// Defines how to handle upgrades, including failures.
	Upgrade *HelmUpgrade `json:"upgrade,omitempty"`
}

// RedpandaSpec defines the desired state of the Redpanda cluster.
type RedpandaSpec struct {
	// Defines chart details, including the version and repository.
	ChartRef ChartRef `json:"chartRef,omitempty"`
	// Defines the Helm values to use to deploy the cluster.
	ClusterSpec *RedpandaClusterSpec `json:"clusterSpec,omitempty"`
	// Migration flag that adjust Kubernetes core resources with annotation and labels, so
	// flux controller can import resources.
	// Doc: https://docs.redpanda.com/current/upgrade/migrate/kubernetes/operator/
	Migration *Migration `json:"migration,omitempty"`
}

// Migration can configure old Cluster and Console custom resource that will be disabled.
// With Migration the ChartRef and ClusterSpec still need to be correctly configured.
type Migration struct {
	Enabled bool `json:"enabled"`
	// ClusterRef by default will not be able to reach different namespaces, but it can be
	// overwritten by adding ClusterRole and ClusterRoleBinding to operator ServiceAccount.
	ClusterRef v1alpha1.NamespaceNameRef `json:"clusterRef"`

	// ConsoleRef by default will not be able to reach different namespaces, but it can be
	// overwritten by adding ClusterRole and ClusterRoleBinding to operator ServiceAccount.
	ConsoleRef v1alpha1.NamespaceNameRef `json:"consoleRef"`
}

// RedpandaStatus defines the observed state of Redpanda
type RedpandaStatus struct {
	// Specifies the last observed generation.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	meta.ReconcileRequestStatus `json:",inline"`

	// Conditions holds the conditions for the Redpanda.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastAppliedRevision is the revision of the last successfully applied source.
	// +optional
	LastAppliedRevision string `json:"lastAppliedRevision,omitempty"`

	// LastAttemptedRevision is the revision of the last reconciliation attempt.
	// +optional
	LastAttemptedRevision string `json:"lastAttemptedRevision,omitempty"`

	// +optional
	HelmRelease string `json:"helmRelease,omitempty"`

	// +optional
	HelmReleaseReady *bool `json:"helmReleaseReady,omitempty"`

	// +optional
	HelmRepository string `json:"helmRepository,omitempty"`

	// +optional
	HelmRepositoryReady *bool `json:"helmRepositoryReady,omitempty"`

	// +optional
	UpgradeFailures int64 `json:"upgradeFailures,omitempty"`

	// Failures is the reconciliation failure count against the latest desired
	// state. It is reset after a successful reconciliation.
	// +optional
	Failures int64 `json:"failures,omitempty"`

	// +optional
	InstallFailures int64 `json:"installFailures,omitempty"`

	// ManagedDecommissioningNode indicates that a node is currently being
	// decommissioned from the cluster and provides its ordinal number.
	// +optional
	ManagedDecommissioningNode *int32 `json:"decommissioningNode,omitempty"`
}

type RemediationStrategy string

// HelmUpgrade configures the behavior and strategy for Helm chart upgrades.
type HelmUpgrade struct {
	// Specifies the actions to take on upgrade failures. See https://pkg.go.dev/github.com/fluxcd/helm-controller/api/v2beta1#UpgradeRemediation.
	Remediation *helmv2beta2.UpgradeRemediation `json:"remediation,omitempty"`
	// Enables forceful updates during an upgrade.
	Force *bool `json:"force,omitempty"`
	// Specifies whether to preserve user-configured values during an upgrade.
	PreserveValues *bool `json:"preserveValues,omitempty"`
	// Specifies whether to perform cleanup in case of failed upgrades.
	CleanupOnFail *bool `json:"cleanupOnFail,omitempty"`
}

// Redpanda defines the CRD for Redpanda clusters.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=redpandas
// +kubebuilder:resource:shortName=rp
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].status",description=""
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].message",description=""
// +kubebuilder:storageversion
type Redpanda struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Defines the desired state of the Redpanda cluster.
	Spec RedpandaSpec `json:"spec,omitempty"`
	// Represents the current status of the Redpanda cluster.
	Status RedpandaStatus `json:"status,omitempty"`
}

// RedpandaList contains a list of Redpanda objects.
// +kubebuilder:object:root=true
type RedpandaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	// Specifies a list of Redpanda resources.
	Items []Redpanda `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Redpanda{}, &RedpandaList{})
}

// GetHelmRelease returns the namespace and name of the HelmRelease.
func (in *RedpandaStatus) GetHelmRelease() string {
	return in.HelmRelease
}

func (in *Redpanda) GetHelmReleaseName() string {
	return in.Name
}

func (in *Redpanda) GetHelmRepositoryName() string {
	helmRepository := in.Spec.ChartRef.HelmRepositoryName
	if helmRepository == "" {
		helmRepository = "redpanda-repository"
	}
	return helmRepository
}

func (in *Redpanda) ValuesJSON() (*apiextensionsv1.JSON, error) {
	vyaml, err := json.Marshal(in.Spec.ClusterSpec)
	if err != nil {
		return nil, fmt.Errorf("could not convert spec to yaml: %w", err)
	}
	values := &apiextensionsv1.JSON{Raw: vyaml}

	return values, nil
}

// RedpandaReady registers a successful reconciliation of the given HelmRelease.
func RedpandaReady(rp *Redpanda) *Redpanda {
	newCondition := metav1.Condition{
		Type:    meta.ReadyCondition,
		Status:  metav1.ConditionTrue,
		Reason:  "RedpandaClusterDeployed",
		Message: "Redpanda reconciliation succeeded",
	}
	apimeta.SetStatusCondition(rp.GetConditions(), newCondition)
	rp.Status.LastAppliedRevision = rp.Status.LastAttemptedRevision
	return rp
}

// RedpandaNotReady registers a failed reconciliation of the given Redpanda.
func RedpandaNotReady(rp *Redpanda, reason, message string) *Redpanda {
	newCondition := metav1.Condition{
		Type:    meta.ReadyCondition,
		Status:  metav1.ConditionFalse,
		Reason:  reason,
		Message: message,
	}
	apimeta.SetStatusCondition(rp.GetConditions(), newCondition)
	return rp
}

// RedpandaProgressing resets any failures and registers progress toward
// reconciling the given Redpanda by setting the meta.ReadyCondition to
// 'Unknown' for meta.ProgressingReason.
func RedpandaProgressing(rp *Redpanda) *Redpanda {
	rp.Status.Conditions = []metav1.Condition{}
	newCondition := metav1.Condition{
		Type:    meta.ReadyCondition,
		Status:  metav1.ConditionUnknown,
		Reason:  meta.ProgressingReason,
		Message: "Reconciliation in progress",
	}
	apimeta.SetStatusCondition(rp.GetConditions(), newCondition)
	return rp
}

// GetConditions returns the status conditions of the object.
func (in *Redpanda) GetConditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *Redpanda) OwnerShipRefObj() metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: in.APIVersion,
		Kind:       in.Kind,
		Name:       in.Name,
		UID:        in.UID,
	}
}

// GetMigrationConsoleName returns Console custom resource namespace which will be taken out from
// old reconciler, so that underlying resources could be migrated.
func (in *Redpanda) GetMigrationConsoleName() string {
	if in.Spec.Migration == nil {
		return ""
	}
	name := in.Spec.Migration.ConsoleRef.Name
	if name == "" {
		name = in.Name
	}
	return name
}

// GetMigrationConsoleNamespace returns Console custom resource name which will be taken out from
// old reconciler, so that underlying resources could be migrated.
func (in *Redpanda) GetMigrationConsoleNamespace() string {
	if in.Spec.Migration == nil {
		return ""
	}
	namespace := in.Spec.Migration.ConsoleRef.Namespace
	if namespace == "" {
		namespace = in.Namespace
	}
	return namespace
}

// GetMigrationClusterName returns Cluster custom resource namespace which will be taken out from
// old reconciler, so that underlying resources could be migrated.
func (in *Redpanda) GetMigrationClusterName() string {
	if in.Spec.Migration == nil {
		return ""
	}
	name := in.Spec.Migration.ClusterRef.Name
	if name == "" {
		name = in.Name
	}
	return name
}

// GetMigrationClusterNamespace returns Cluster custom resource name which will be taken out from
// old reconciler, so that underlying resources could be migrated.
func (in *Redpanda) GetMigrationClusterNamespace() string {
	if in.Spec.Migration == nil {
		return ""
	}
	namespace := in.Spec.Migration.ClusterRef.Namespace
	if namespace == "" {
		namespace = in.Namespace
	}
	return namespace
}

// ClusterRef represents a reference to a cluster that is being targeted.
type ClusterRef struct {
	// Namespace specifies namespace of the cluster being referenced. If empty, the namespace
	// of the referencing object will be used.
	Namespace string `json:"namespace,omitempty"`
	// Name specifies the name of the cluster being referenced.
	Name string `json:"name,omitempty"`
}

// Redpanda defines the CRD for Redpanda user.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=redpandaUsers
// +kubebuilder:resource:shortName=rpu
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].message",description=""
// +kubebuilder:storageversion
type RedpandaUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Defines the desired state of the Redpanda user.
	Spec RedpandaUserSpec `json:"spec,omitempty"`
	// Represents the current status of the Redpanda user.
	Status RedpandaUserStatus `json:"status,omitempty"`
}

// RedpandaUserStatus defines the observed state of a Redpanda user
type RedpandaUserStatus struct {
	// Specifies the last observed generation.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions holds the conditions for the Redpanda user.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// RedpandaUserSpec defines a user of a Redpanda cluster.
type RedpandaUserSpec struct {
	// ClusterRef is a reference to the cluster where the user should be created.
	ClusterRef ClusterRef `json:"clusterRef,omitempty"`
	// Authentication defines the authentication information for a user.
	// If authentication is not configured, no credentials are generated.
	// +optional
	Authentication RedpandaUserAuthenticationSpec `json:"authentication,omitempty"`
	// Authorization rules defined for this user.
	// +optional
	Authorization RedpandaUserAuthorizationSpec `json:"authorization,omitempty"`
	// Quotas on requests to control the broker resources used by clients. Network
	// bandwidth and request rate quotas can be enforced.
	// +optional
	Quotas RedpandaQuotasSpec `json:"quotas,omitempty"`
	// Template to specify how user secrets are generated.
	// +optional
	Template RedpandaUserTemplateSpec `json:"template,omitempty"`
}

// RedpandaUserTemplateSpec defines the template metadata for a user
type RedpandaUserTemplateSpec struct {
	// Template for RedpandaUser resources. The template allows users
	// to specify how the Secret with password or TLS certificates is generated. 
	Secret ResourceTemplate `json:"secret,omitempty"`
}

// ResourceTemplate specifies additional configuration for a resource.
type ResourceTemplate struct {
	// Metadata specifies additional metadata to associate with a resource.
	Metadata MetadataTemplate `json:"metadata,omitempty"`
}

// MetadataTemplate defines additional metadata to associate with a resource.
type MetadataTemplate struct {
	// Labels specifies the Kubernetes labels to apply to a managed resource.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations specifies the Kubernetes annotations to apply to a managed resource.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// RedpandaQuotasSpec defines the quotas for a user.
type RedpandaQuotasSpec struct {
	// A quota on the maximum bytes per-second that each client group can publish to a broker
	// before the clients in the group are throttled. Defined on a per-broker basis.
	// +optional
	ProducerByteRate *int `json:"producerByteRate,omitempty"`
	// A quota on the maximum bytes per-second that each client group can fetch from a broker
	// before the clients in the group are throttled. Defined on a per-broker basis.
	// +optional
	ConsumerByteRate *int `json:"consumerByteRate,omitempty"`
	// A quota on the maximum CPU utilization of each client group as a percentage of network and I/O threads.
	// +optional
	RequestPercentage *int `json:"requestPercentage,omitempty"`
	// A quota on the rate at which mutations are accepted for the create topics request, the create
	// partitions request and the delete topics request. The rate is accumulated by the number of partitions
	// created or deleted.
	// +optional
	ControllerMutationRate *int `json:"controllerMutationRate,omitempty"`
}

// RedpandaUserAuthenticationSpec defines the authentication mechanism enabled for this Redpanda user.
type RedpandaUserAuthenticationSpec struct {
	// one of: scram-sha-512, tls, or tls-external
	Type string `json:"type,omitempty"`
}

// Authorization rules for this user.
type RedpandaUserAuthorizationSpec struct {
	// one of: simple
	Type string `json:"type,omitempty"`
	// List of ACL rules which should be applied to this user.
	ACLs []RedpandaACLRule `json:"acls,omitempty"`
}

// Defines an ACL rule applied to the given user.
type RedpandaACLRule struct {
	// one of: allow, deny
	Type string `json:"type,omitempty"`
	// Indicates the resource for which given ACL rule applies.
	Resource RedpandaACLResourceSpec `json:"resource,omitempty"`
	// The host from which the action described in the ACL rule is allowed or denied.
	// If not set, it defaults to *, allowing or denying the action from any host.
	Host string `json:"host,omitempty"`
	// List of operations which will be allowed or denied.
	// one or more of [Read, Write, Delete, Alter, Describe, All, IdempotentWrite, ClusterAction, Create, AlterConfigs, DescribeConfigs]
	Operations []string `json:"operations,omitempty"`
}

// Indicates the resource for which given ACL rule applies.
type RedpandaACLResourceSpec struct {
	// one of: topic, group, cluster, transactionalId
	Type string `json:"type,omitempty"`
	// Name of resource for which given ACL rule applies.
	// Can be combined with patternType field to use prefix pattern.
	Name string `json:"name,omitempty"`
	// Describes the pattern used in the resource field. The supported types are literal
	// and prefix. With literal pattern type, the resource field will be used as a definition
	// of a full topic name. With prefix pattern type, the resource name will be used only as
	// a prefix. Default value is literal.
	// 
	// one of: prefix, literal
	PatternType string `json:"patternType,omitempty"`
}
