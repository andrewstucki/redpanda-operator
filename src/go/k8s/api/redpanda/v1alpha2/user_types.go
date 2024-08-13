package v1alpha2

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// ClusterRef represents a reference to a cluster that is being targeted.
type ClusterRef struct {
	// Name specifies the name of the cluster being referenced.
	// +kubebuilder:validation:Required
	Name string `json:"name"`
}

// User defines the CRD for Redpanda user.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=users
// +kubebuilder:resource:shortName=rpu
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].message",description=""
// +kubebuilder:storageversion
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Defines the desired state of the Redpanda user.
	Spec UserSpec `json:"spec,omitempty"`
	// Represents the current status of the Redpanda user.
	Status UserStatus `json:"status,omitempty"`
}

// UserList contains a list of Redpanda user objects.
// +kubebuilder:object:root=true
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	// Specifies a list of Redpanda user resources.
	Items []User `json:"items"`
}

// UserStatus defines the observed state of a Redpanda user
type UserStatus struct {
	// Specifies the last observed generation.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Conditions holds the conditions for the Redpanda user.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// ClusterRef is a reference to the cluster where the user was created.
	// +optional
	ClusterRef *ClusterRef `json:"clusterRef,omitempty"`
}

// UserSpec defines a user of a Redpanda cluster.
type UserSpec struct {
	// ClusterRef is a reference to the cluster where the user should be created.
	// This takes precedence over KafkaAPISpec
	// +optional
	ClusterRef *ClusterRef `json:"clusterRef,omitempty"`
	// KafkaAPISpec is the configuration information for communicating with a Redpanda cluster.
	// +optional
	KafkaAPISpec *KafkaAPISpec `json:"kafkaApiSpec,omitempty"`
	// Authentication defines the authentication information for a user.
	// If authentication is not configured, no credentials are generated.
	// +optional
	Authentication UserAuthenticationSpec `json:"authentication,omitempty"`
	// Authorization rules defined for this user.
	// +optional
	Authorization UserAuthorizationSpec `json:"authorization,omitempty"`
	// Quotas on requests to control the broker resources used by clients. Network
	// bandwidth and request rate quotas can be enforced.
	// +optional
	Quotas QuotasSpec `json:"quotas,omitempty"`
	// Template to specify how user secrets are generated.
	// +optional
	Template UserTemplateSpec `json:"template,omitempty"`
}

// UserTemplateSpec defines the template metadata for a user
type UserTemplateSpec struct {
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

// QuotasSpec defines the quotas for a user.
type QuotasSpec struct {
	// A quota on the maximum bytes per-second that each client group can publish to a broker
	// before the clients in the group are throttled. Defined on a per-broker basis.
	// +optional
	ProducerByteRate *int `json:"producerByteRate,omitempty"`
	// A quota on the maximum bytes per-second that each client group can fetch from a broker
	// before the clients in the group are throttled. Defined on a per-broker basis.
	// +optional
	ConsumerByteRate *int `json:"consumerByteRate,omitempty"`
	// A quota on the rate at which mutations are accepted for the create topics request, the create
	// partitions request and the delete topics request. The rate is accumulated by the number of partitions
	// created or deleted.
	// +optional
	ControllerMutationRate *int `json:"controllerMutationRate,omitempty"`
}

// UserAuthenticationSpec defines the authentication mechanism enabled for this Redpanda user.
type UserAuthenticationSpec struct {
	// +kubebuilder:validation:Enum=scram-sha-512;tls;tls-external
	// +kubebuilder:validation:Required
	Type string `json:"type"`
	// Password for SCRAM based authentication
	Password *Password `json:"password,omitempty"`
}

type Password struct {
	// +kubebuilder:validation:Required
	ValueFrom PasswordSource `json:"valueFrom"`
}

type PasswordSource struct {
	// +kubebuilder:validation:Required
	SecretKeyRef SecretKeyRef `json:"secretKeyRef"`
}

// Authorization rules for this user.
type UserAuthorizationSpec struct {
	// +kubebuilder:validation:Enum=simple
	// +kubebuilder:default=simple
	// +kubebuilder:validation:Required
	Type string `json:"type,omitempty"`
	// List of ACL rules which should be applied to this user.
	ACLs []ACLRule `json:"acls,omitempty"`
}

// Defines an ACL rule applied to the given user.
type ACLRule struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=allow;deny
	Type string `json:"type"`
	// Indicates the resource for which given ACL rule applies.
	// +kubebuilder:validation:Required
	Resource ACLResourceSpec `json:"resource"`
	// The host from which the action described in the ACL rule is allowed or denied.
	// If not set, it defaults to *, allowing or denying the action from any host.
	// +kubebuilder:default:*
	Host string `json:"host,omitempty"`
	// List of operations which will be allowed or denied.
	// +kubebuilder:validation:UniqueItems=true
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:Enum=Read;Write;Delete;Alter;Describe;All;IdempotentWrite;ClusterAction;Create;AlterConfigs;DescribeConfigs
	Operations []string `json:"operations"`
}

// Indicates the resource for which given ACL rule applies.
type ACLResourceSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=topic;group;cluster;transactionalId
	Type string `json:"type"`
	// Name of resource for which given ACL rule applies.
	// Can be combined with patternType field to use prefix pattern.
	Name string `json:"name,omitempty"`
	// Describes the pattern used in the resource field. The supported types are literal
	// and prefix. With literal pattern type, the resource field will be used as a definition
	// of a full topic name. With prefix pattern type, the resource name will be used only as
	// a prefix. Default value is literal.
	//
	// +kubebuilder:validation:Enum=prefix;literal
	// +kubebuilder:default=literal
	PatternType string `json:"patternType,omitempty"`
}
