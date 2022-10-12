package v1beta2

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
	Account           string                `json:"account"`
	Creds             string                `json:"creds"`
	Description       string                `json:"description"`
	Discard           string                `json:"discard"`
	DuplicateWindow   string                `json:"duplicateWindow"`
	MaxAge            string                `json:"maxAge"`
	MaxBytes          int                   `json:"maxBytes"`
	MaxConsumers      int                   `json:"maxConsumers"`
	MaxMsgs           int                   `json:"maxMsgs"`
	MaxMsgSize        int                   `json:"maxMsgSize"`
	MaxMsgsPerSubject int                   `json:"maxMsgsPerSubject"`
	Mirror            *StreamSource         `json:"mirror"`
	Name              string                `json:"name"`
	Nkey              string                `json:"nkey"`
	NoAck             bool                  `json:"noAck"`
	Placement         *StreamPlacement      `json:"placement"`
	Replicas          int                   `json:"replicas"`
	Republish         *StreamSubjectMapping `json:"republish"`
	Retention         string                `json:"retention"`
	Servers           []string              `json:"servers"`
	Sources           []*StreamSource       `json:"sources"`
	Storage           string                `json:"storage"`
	Subjects          []string              `json:"subjects"`
	TLS               TLS                   `json:"tls"`
}

type StreamPlacement struct {
	Cluster string   `json:"cluster"`
	Tags    []string `json:"tags"`
}

type StreamSource struct {
	Name          string `json:"name"`
	OptStartSeq   int    `json:"optStartSeq"`
	OptStartTime  string `json:"optStartTime"`
	FilterSubject string `json:"filterSubject"`

	ExternalAPIPrefix     string `json:"externalApiPrefix"`
	ExternalDeliverPrefix string `json:"externalDeliverPrefix"`
}

type StreamSubjectMapping struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StreamList is a list of Stream resources
type StreamList struct {
	k8smeta.TypeMeta `json:",inline"`
	k8smeta.ListMeta `json:"metadata"`

	Items []Stream `json:"items"`
}
