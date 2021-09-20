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
	Discard         string   `json:"discard"`
	DuplicateWindow string   `json:"duplicateWindow"`
	MaxAge          string   `json:"maxAge"`
	MaxBytes        int      `json:"maxBytes"`
	MaxConsumers    int      `json:"maxConsumers"`
	MaxMsgs         int      `json:"maxMsgs"`
	MaxMsgSize      int      `json:"maxMsgSize"`
	Name            string   `json:"name"`
	NoAck           bool     `json:"noAck"`
	Replicas        int      `json:"replicas"`
	Retention       string   `json:"retention"`
	Storage         string   `json:"storage"`
	Subjects        []string `json:"subjects"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StreamList is a list of Stream resources
type StreamList struct {
	k8smeta.TypeMeta `json:",inline"`
	k8smeta.ListMeta `json:"metadata"`

	Items []Stream `json:"items"`
}
