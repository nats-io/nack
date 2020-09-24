package v1

import (
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StreamTemplate is a specification for a StreamTemplate resource
type StreamTemplate struct {
	k8smeta.TypeMeta   `json:",inline"`
	k8smeta.ObjectMeta `json:"metadata,omitempty"`

	Spec   StreamTemplateSpec `json:"spec"`
	Status Status             `json:"status"`
}

func (s *StreamTemplate) GetSpec() interface{} {
	return s.Spec
}

// StreamTemplateSpec is the spec for a StreamTemplate resource
type StreamTemplateSpec struct {
	StreamSpec `json:",inline"`

	MaxStreams int `json:"maxStreams"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StreamTemplateList is a list of StreamTemplate resources
type StreamTemplateList struct {
	k8smeta.TypeMeta `json:",inline"`
	k8smeta.ListMeta `json:"metadata"`

	Items []StreamTemplate `json:"items"`
}
