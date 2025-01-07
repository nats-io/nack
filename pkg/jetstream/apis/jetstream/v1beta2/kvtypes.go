package v1beta2

import (
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Stream is a specification for a Stream resource
type KeyValue struct {
	k8smeta.TypeMeta   `json:",inline"`
	k8smeta.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeyValueSpec `json:"spec"`
	Status Status       `json:"status"`
}

func (s *KeyValue) GetSpec() interface{} {
	return s.Spec
}

// StreamSpec is the spec for a Stream resource
type KeyValueSpec struct {
	Bucket       string           `json:"bucket"`
	Description  string           `json:"description"`
	MaxValueSize int              `json:"maxValueSize"`
	History      int              `json:"history"`
	TTL          string           `json:"ttl"`
	MaxBytes     int              `json:"maxBytes"`
	Storage      string           `json:"storage"`
	Replicas     int              `json:"replicas"`
	Placement    *StreamPlacement `json:"placement"`
	RePublish    *RePublish       `json:"republish"`
	Mirror       *StreamSource    `json:"mirror"`
	Sources      []*StreamSource  `json:"sources"`
	Compression  bool             `json:"compression"`
	BaseStreamConfig
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KeyValueList is a list of Stream resources
type KeyValueList struct {
	k8smeta.TypeMeta `json:",inline"`
	k8smeta.ListMeta `json:"metadata"`

	Items []KeyValue `json:"items"`
}
