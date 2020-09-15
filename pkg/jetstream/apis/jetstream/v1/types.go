package v1

import (
	k8sapi "k8s.io/api/core/v1"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Stream is a specification for a Stream resource
type Stream struct {
	k8smeta.TypeMeta   `json:",inline"`
	k8smeta.ObjectMeta `json:"metadata,omitempty"`

	Spec   StreamSpec   `json:"spec"`
	Status StreamStatus `json:"status"`
}

// StreamSpec is the spec for a Stream resource
type StreamSpec struct {
	Servers  []string `json:"servers"`
	Name     string   `json:"name"`
	Subjects []string `json:"subjects"`
	Storage  string   `json:"storage"`
	MaxAge   string   `json:"maxAge"`
}

// StreamStatus is the status for a Stream resource
type StreamStatus struct {
	ObservedGeneration int64             `json:"observedGeneration"`
	Conditions         []StreamCondition `json:"conditions"`
}

type StreamCondition struct {
	Type               string                 `json:"type"`
	Status             k8sapi.ConditionStatus `json:"status"`
	Reason             string                 `json:"reason"`
	Message            string                 `json:"message"`
	LastTransitionTime string                 `json:"lastTransitionTime"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StreamList is a list of Stream resources
type StreamList struct {
	k8smeta.TypeMeta `json:",inline"`
	k8smeta.ListMeta `json:"metadata"`

	Items []Stream `json:"items"`
}
