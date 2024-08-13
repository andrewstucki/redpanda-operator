package v1alpha1

import "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha2"

// Topic defines the CRD for Topic resources. See https://docs.redpanda.com/current/manage/kubernetes/manage-topics/.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Topic v1alpha2.Topic

// TopicList contains a list of Topic objects.
// +kubebuilder:object:root=true
type TopicList v1alpha2.TopicList
