package v1beta2

import (
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:storageversion

// Stream is a specification for a Stream resource
type ObjectStore struct {
	k8smeta.TypeMeta   `json:",inline"`
	k8smeta.ObjectMeta `json:"metadata,omitempty"`

	Spec   ObjectStoreSpec `json:"spec"`
	Status Status          `json:"status"`
}

func (s *ObjectStore) GetSpec() interface{} {
	return s.Spec
}

// StreamSpec is the spec for a Stream resource
type ObjectStoreSpec struct {
	Bucket      string            `json:"bucket"`
	Description string            `json:"description,omitempty"`
	TTL         string            `json:"ttl,omitempty"`
	MaxBytes    int               `json:"maxBytes,omitempty"`
	Storage     string            `json:"storage,omitempty"`
	Replicas    int               `json:"replicas,omitempty"`
	Placement   *StreamPlacement  `json:"placement,omitempty"`
	Compression bool              `json:"compression,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	BaseStreamConfig
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ObjectStoreList is a list of Stream resources
type ObjectStoreList struct {
	k8smeta.TypeMeta `json:",inline"`
	k8smeta.ListMeta `json:"metadata"`

	Items []ObjectStore `json:"items"`
}
