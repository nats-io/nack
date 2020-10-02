package v1beta1

import (
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Stream is a specification for a Stream resource
type Stream struct {
	k8smeta.TypeMeta   `json:",inline"`
	k8smeta.ObjectMeta `json:"metadata,omitempty"`

	Spec   StreamSpec `json:"spec"`
	Status Status     `json:"status"`
}

func (s *Stream) GetSpec() interface{} {
	return s.Spec
}

// StreamSpec is the spec for a Stream resource
type StreamSpec struct {
	Name            string   `json:"name"`
	Subjects        []string `json:"subjects"`
	Retention       string   `json:"retention"`
	MaxConsumers    int      `json:"maxConsumers"`
	MaxMsgs         int      `json:"maxMsgs"`
	MaxBytes        int      `json:"maxBytes"`
	MaxAge          string   `json:"maxAge"`
	MaxMsgSize      int      `json:"maxMsgSize"`
	Storage         string   `json:"storage"`
	Replicas        int      `json:"replicas"`
	NoAck           bool     `json:"noAck"`
	Discard         string   `json:"discard"`
	DuplicateWindow string   `json:"duplicateWindow"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StreamList is a list of Stream resources
type StreamList struct {
	k8smeta.TypeMeta `json:",inline"`
	k8smeta.ListMeta `json:"metadata"`

	Items []Stream `json:"items"`
}
