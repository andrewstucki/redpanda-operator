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
	"fmt"
	"slices"

	"github.com/twmb/franz-go/pkg/sr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func init() {
	SchemeBuilder.Register(&Schema{}, &SchemaList{})
}

// Schema defines the CRD for a Redpanda schema.
// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=schemas
// +kubebuilder:resource:shortName=sc
// +kubebuilder:printcolumn:name="Synced",type="string",JSONPath=`.status.conditions[?(@.type=="Synced")].status`
type Schema struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Defines the desired state of the Redpanda schema.
	Spec SchemaSpec `json:"spec"`
	// Represents the current status of the Redpanda schema.
	// +kubebuilder:default={conditions: {{type: "Synced", status: "Unknown", reason:"Pending", message:"Waiting for controller", lastTransitionTime: "1970-01-01T00:00:00Z"}}}
	Status SchemaStatus `json:"status,omitempty"`
}

func (s *Schema) Matches(schema *sr.SubjectSchema) bool {
	// TODO: actually fill this in

	if s.Name != schema.Subject {
		return false
	}

	// type
	if schemaTypesFromKafka[schema.Type] != s.GetSchemaType() {
		return false
	}

	// text
	if s.Spec.Text != schema.Schema.Schema {
		return false
	}

	// references
	if !compareConvertibleSlices(s.Spec.References, schema.References, func(a SchemaReference, b sr.SchemaReference) bool {
		return false
	}) {
		return false
	}

	// metadata
	if !compareMaps(s.Spec.SchemaMetadata.Properties, schema.SchemaMetadata.Properties) {
		return false
	}
	if !compareMapsFn(s.Spec.SchemaMetadata.Tags, schema.SchemaMetadata.Tags, slices.Equal) {
		return false
	}
	if !slices.Equal(s.Spec.SchemaMetadata.Sensitive, schema.SchemaMetadata.Sensitive) {
		return false
	}

	// rule set
	if !compareConvertibleSlices(s.Spec.SchemaRuleSet.DomainRules, schema.SchemaRuleSet.DomainRules, func(a SchemaRule, b sr.SchemaRule) bool {
		return false
	}) {
		return false
	}
	if !compareConvertibleSlices(s.Spec.SchemaRuleSet.MigrationRules, schema.SchemaRuleSet.MigrationRules, func(a SchemaRule, b sr.SchemaRule) bool {
		return false
	}) {
		return false
	}

	return true
}

func (s *Schema) ToKafka() sr.Schema {
	// TODO: actually fill this in
	return sr.Schema{
		Schema: s.Spec.Text,
	}
}

func (s *Schema) GetSchemaType() SchemaType {
	if s.Spec.Type == nil {
		return SchemaTypeAvro
	}
	return *s.Spec.Type
}

func compareMaps[T, U comparable](a, b map[T]U) bool {
	if len(a) != len(b) {
		return false
	}
	for key := range a {
		if a[key] != b[key] {
			return false
		}
	}
	return true
}

func compareMapsFn[T comparable, U any](a, b map[T]U, fn func(U, U) bool) bool {
	if len(a) != len(b) {
		return false
	}
	for key := range a {
		if !fn(a[key], b[key]) {
			return false
		}
	}
	return true
}

func compareConvertibleSlices[T, U any](a []T, b []U, fn func(T, U) bool) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if !fn(a[i], b[i]) {
			return false
		}
	}
	return true
}

var _ ClusterReferencingObject = (*Schema)(nil)

func (s *Schema) GetClusterSource() *ClusterSource {
	return s.Spec.ClusterSource
}

// SchemaType specifies the type of the given schema.
// +kubebuilder:validation:Enum=avro;protobuf;json
type SchemaType string

const (
	SchemaTypeAvro     SchemaType = "avro"
	SchemaTypeProtobuf SchemaType = "protobuf"
	SchemaTypeJSON     SchemaType = "json"
)

var (
	schemaTypesFromKafka = map[sr.SchemaType]SchemaType{
		sr.TypeAvro:     SchemaTypeAvro,
		sr.TypeProtobuf: SchemaTypeProtobuf,
		sr.TypeJSON:     SchemaTypeJSON,
	}
	schemaTypesToKafka = map[SchemaType]sr.SchemaType{
		SchemaTypeAvro:     sr.TypeAvro,
		SchemaTypeProtobuf: sr.TypeProtobuf,
		SchemaTypeJSON:     sr.TypeJSON,
	}
)

// SchemaRuleKind as an enum representing the kind of schema rule.
//
// +kubebuilder:validation:Enum=transform;condition
type SchemaRuleKind string

const (
	SchemaRuleKindTransform SchemaRuleKind = "transform"
	SchemaRuleKindCondition SchemaRuleKind = "condition"
)

// SchemaRuleMode specifies a schema rule's mode.
//
// Migration rules can be specified for an UPGRADE, DOWNGRADE, or both
// (UPDOWN). Migration rules are used during complex schema evolution.
//
// Domain rules can be specified during serialization (WRITE), deserialization
// (READ) or both (WRITEREAD).
//
// Domain rules can be used to transform the domain values in a message
// payload.
//
// +kubebuilder:validation:Enum=upgrade;downgrade;updown;write;read;writeread
type SchemaRuleMode string

const (
	SchemaRuleModeUpgrade   SchemaRuleMode = "upgrade"
	SchemaRuleModeDowngrade SchemaRuleMode = "downgrade"
	SchemaRuleModeUpdown    SchemaRuleMode = "updown"
	SchemaRuleModeWrite     SchemaRuleMode = "write"
	SchemaRuleModeRead      SchemaRuleMode = "read"
	SchemaRuleModeWriteRead SchemaRuleMode = "writeread"
)

// +kubebuilder:validation:Enum=None;Backward;BackwardTransitive;Forward;ForwardTransitive;Full;FullTransitive
type CompatibilityLevel string

const (
	CompatabilityLevelNone               CompatibilityLevel = "None"
	CompatabilityLevelBackward           CompatibilityLevel = "Backward"
	CompatabilityLevelBackwardTransitive CompatibilityLevel = "BackwardTransitive"
	CompatabilityLevelForward            CompatibilityLevel = "Forward"
	CompatabilityLevelForwardTransitive  CompatibilityLevel = "ForwardTransitive"
	CompatabilityLevelFull               CompatibilityLevel = "Full"
	CompatabilityLevelFullTransitive     CompatibilityLevel = "FullTransitive"
)

func (c CompatibilityLevel) ToKafka() sr.CompatibilityLevel {
	return compatibilityLevelsToKafka[c]
}

var (
	compatibilityLevelsFromKafka = map[sr.CompatibilityLevel]CompatibilityLevel{
		sr.CompatNone:               CompatabilityLevelNone,
		sr.CompatBackward:           CompatabilityLevelBackward,
		sr.CompatBackwardTransitive: CompatabilityLevelBackwardTransitive,
		sr.CompatForward:            CompatabilityLevelForward,
		sr.CompatForwardTransitive:  CompatabilityLevelForwardTransitive,
		sr.CompatFull:               CompatabilityLevelFull,
		sr.CompatFullTransitive:     CompatabilityLevelFullTransitive,
	}
	compatibilityLevelsToKafka = map[CompatibilityLevel]sr.CompatibilityLevel{
		CompatabilityLevelNone:               sr.CompatNone,
		CompatabilityLevelBackward:           sr.CompatBackward,
		CompatabilityLevelBackwardTransitive: sr.CompatBackwardTransitive,
		CompatabilityLevelForward:            sr.CompatForward,
		CompatabilityLevelForwardTransitive:  sr.CompatForwardTransitive,
		CompatabilityLevelFull:               sr.CompatFull,
		CompatabilityLevelFullTransitive:     sr.CompatFullTransitive,
	}
)

// SchemaSpec defines the configuration of a Redpanda schema.
type SchemaSpec struct {
	// ClusterSource is a reference to the cluster hosting the schema registry.
	// It is used in constructing the client created to configure a cluster.
	// +required
	// +kubebuilder:validation:XValidation:message="spec.cluster.staticConfiguration.schemaRegistry: required value",rule=`!has(self.staticConfiguration) || has(self.staticConfiguration.schemaRegistry)`
	ClusterSource *ClusterSource `json:"cluster"`
	// Text is the actual unescaped text of a schema.
	// +required
	Text string `json:"text"`
	// Type is the type of a schema. The default type is avro.
	//
	// +kubebuilder:default=avro
	Type *SchemaType `json:"schemaType,omitempty"`

	// References declares other schemas this schema references. See the
	// docs on SchemaReference for more details.
	References []SchemaReference `json:"references,omitempty"`

	// SchemaMetadata is arbitrary information about the schema.
	SchemaMetadata *SchemaMetadata `json:"metadata,omitempty"`

	// SchemaRuleSet is a set of rules that govern the schema.
	SchemaRuleSet *SchemaRuleSet `json:"ruleSet,omitempty"`

	// CompatibilityLevel sets the compatibility level for the given schema
	// +kubebuilder:default=Backward
	CompatibilityLevel *CompatibilityLevel `json:"compatibilityLevel,omitempty"`
}

func (s *SchemaSpec) MatchesCompatibility(c sr.CompatibilityLevel) bool {
	return compatibilityLevelsFromKafka[c] == s.GetCompatibilityLevel()
}

func (s *SchemaSpec) GetCompatibilityLevel() CompatibilityLevel {
	if s.CompatibilityLevel == nil {
		return CompatabilityLevelBackward
	}
	return *s.CompatibilityLevel
}

// SchemaReference is a way for a one schema to reference another. The
// details for how referencing is done are type specific; for example,
// JSON objects that use the key "$ref" can refer to another schema via
// URL.
type SchemaReference struct {
	Name    string `json:"name"`
	Subject string `json:"subject"`
	Version int    `json:"version"`
}

// SchemaRule specifies integrity constraints or data policies in a
// data contract. These data rules or policies can enforce that a field
// that contains sensitive information must be encrypted, or that a
// message containing an invalid age must be sent to a dead letter
// queue.
type SchemaRule struct {
	// Name is a user-defined name to reference the rule.
	Name string `json:"name"`
	// Doc is an optional description of the rule.
	Doc string `json:"doc,omitempty"`
	// Kind is the type of rule.
	//
	// +kubebuilder:default=transform
	Kind SchemaRuleKind `json:"kind,omitempty"`
	// Mode is the mode of the rule.
	//
	// +kubebuilder:default=upgrade
	Mode *SchemaRuleMode `json:"mode,omitempty"`
	// Type is the type of rule, which invokes a specific rule executor, such as Google Common Expression Language (CEL) or JSONata.
	Type string `json:"type"`
	// Tags to which this rule applies.
	Tags []string `json:"tags"`
	// Optional params for the rule.
	Params map[string]string `json:"params,omitempty"`
	// Expr is the rule expression.
	Expr string `json:"expr"`
	// OnSuccess is an optional action to execute if the rule succeeds, otherwise the built-in action type NONE is used. For UPDOWN and WRITEREAD rules, one can specify two actions separated by commas, such as "NONE,ERROR" for a WRITEREAD rule. In this case NONE applies to WRITE and ERROR applies to READ
	OnSuccess string `json:"onSuccess,omitempty"`
	// OnFailure is an optional action to execute if the rule fails, otherwise the built-in action type NONE is used. See OnSuccess for more details.
	OnFailure string `json:"onFailure,omitempty"`
	// Disabled specifies whether the rule is disabled.
	Disabled bool `json:"disabled,omitempty"`
}

// SchemaRuleSet groups migration rules and domain validation rules.
type SchemaRuleSet struct {
	MigrationRules []SchemaRule `json:"migrationRules,omitempty"`
	DomainRules    []SchemaRule `json:"domainRules,omitempty"`
}

// SchemaMetadata is arbitrary information about the schema or its
// constituent parts, such as whether a field contains sensitive
// information or who created a data contract.
type SchemaMetadata struct {
	Tags       map[string][]string `json:"tags,omitempty"`
	Properties map[string]string   `json:"properties,omitempty"`
	Sensitive  []string            `json:"sensitive,omitempty"`
}

// SchemaStatus defines the observed state of a Redpanda schema.
type SchemaStatus struct {
	// Specifies the last observed generation.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Conditions holds the conditions for the Redpanda schema.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// Versions shows the versions of a given schema
	Versions []int `json:"versions,omitempty"`
}

const (
	SchemaConditionTypeSynced = "Synced"

	SchemaConditionReasonPending              = "Pending"
	SchemaConditionReasonSynced               = "Synced"
	SchemaConditionReasonClusterRefInvalid    = "ClusterRefInvalid"
	SchemaConditionReasonConfigurationInvalid = "ConfigurationInvalid"
	SchemaConditionReasonTerminalClientError  = "TerminalClientError"
	SchemaConditionReasonUnexpectedError      = "UnexpectedError"
)

func SchemaSyncedCondition(name string) metav1.Condition {
	return metav1.Condition{
		Type:    SchemaConditionTypeSynced,
		Status:  metav1.ConditionTrue,
		Reason:  SchemaConditionReasonSynced,
		Message: fmt.Sprintf("Schema %q successfully synced to cluster.", name),
	}
}

func SchemaNotSyncedCondition(reason string, err error) metav1.Condition {
	return metav1.Condition{
		Type:    SchemaConditionTypeSynced,
		Status:  metav1.ConditionFalse,
		Reason:  reason,
		Message: fmt.Sprintf("Error: %v", err),
	}
}

// SchemaList contains a list of Redpanda schema objects.
// +kubebuilder:object:root=true
type SchemaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	// Specifies a list of Redpanda schema resources.
	Items []Schema `json:"items"`
}

func (s *SchemaList) GetItems() []*Schema {
	return mapFn(ptr.To, s.Items)
}
