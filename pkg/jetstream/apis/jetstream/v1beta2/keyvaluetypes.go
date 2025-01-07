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
	Account       string           `json:"account"`
	Compression   bool             `json:"compression"`
	Creds         string           `json:"creds"`
	Description   string           `json:"description"`
	History       int              `json:"history"`
	MaxBytes      int              `json:"maxBytes"`
	MaxValueSize  int              `json:"maxValueSize"`
	Mirror        *StreamSource    `json:"mirror"`
	Name          string           `json:"name"`
	Nkey          string           `json:"nkey"`
	Placement     *StreamPlacement `json:"placement"`
	PreventDelete bool             `json:"preventDelete"`
	PreventUpdate bool             `json:"preventUpdate"`
	Replicas      int              `json:"replicas"`
	Republish     *RePublish       `json:"republish"`
	Servers       []string         `json:"servers"`
	Sources       []*StreamSource  `json:"sources"`
	Storage       string           `json:"storage"`
	TLS           TLS              `json:"tls"`
	TTL           string           `json:"ttl"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KeyValueList is a list of Stream resources
type KeyValueList struct {
	k8smeta.TypeMeta `json:",inline"`
	k8smeta.ListMeta `json:"metadata"`

	Items []KeyValue `json:"items"`
}
