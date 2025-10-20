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
	Bucket       string           `json:"bucket,omitempty"`
	Description  string           `json:"description,omitempty"`
	MaxValueSize int              `json:"maxValueSize,omitempty"`
	History      int              `json:"history,omitempty"`
	TTL          string           `json:"ttl,omitempty"`
	MaxBytes     int              `json:"maxBytes,omitempty"`
	Storage      string           `json:"storage,omitempty"`
	Replicas     int              `json:"replicas,omitempty"`
	Placement    *StreamPlacement `json:"placement,omitempty"`
	RePublish    *RePublish       `json:"republish,omitempty"`
	Mirror       *StreamSource    `json:"mirror,omitempty"`
	Sources      []*StreamSource  `json:"sources,omitempty"`
	Compression  bool             `json:"compression,omitempty"`
	BaseStreamConfig
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KeyValueList is a list of Stream resources
type KeyValueList struct {
	k8smeta.TypeMeta `json:",inline"`
	k8smeta.ListMeta `json:"metadata"`

	Items []KeyValue `json:"items"`
}
