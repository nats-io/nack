package v1beta2

import (
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

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
	Description string            `json:"description"`
	TTL         string            `json:"ttl"`
	MaxBytes    int               `json:"maxBytes"`
	Storage     string            `json:"storage"`
	Replicas    int               `json:"replicas"`
	Placement   *StreamPlacement  `json:"placement"`
	Compression bool              `json:"compression"`
	Metadata    map[string]string `json:"metadata"`
	BaseStreamConfig
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ObjectStoreList is a list of Stream resources
type ObjectStoreList struct {
	k8smeta.TypeMeta `json:",inline"`
	k8smeta.ListMeta `json:"metadata"`

	Items []ObjectStore `json:"items"`
}
