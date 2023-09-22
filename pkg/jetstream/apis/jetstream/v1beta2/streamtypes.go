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
	Account           string            `json:"account"`
	AllowDirect       bool              `json:"allowDirect"`
	AllowRollup       bool              `json:"allowRollup"`
	Creds             string            `json:"creds"`
	DenyDelete        bool              `json:"denyDelete"`
	Description       string            `json:"description"`
	DiscardPerSubject bool              `json:"discardPerSubject"`
	PreventDelete     bool              `json:"preventDelete"`
	PreventUpdate     bool              `json:"preventUpdate"`
	Discard           string            `json:"discard"`
	DuplicateWindow   string            `json:"duplicateWindow"`
	MaxAge            string            `json:"maxAge"`
	MaxBytes          int               `json:"maxBytes"`
	MaxConsumers      int               `json:"maxConsumers"`
	MaxMsgs           int               `json:"maxMsgs"`
	MaxMsgSize        int               `json:"maxMsgSize"`
	MaxMsgsPerSubject int               `json:"maxMsgsPerSubject"`
	Mirror            *StreamSource     `json:"mirror"`
	Name              string            `json:"name"`
	Nkey              string            `json:"nkey"`
	NoAck             bool              `json:"noAck"`
	Placement         *StreamPlacement  `json:"placement"`
	Replicas          int               `json:"replicas"`
	Republish         *RePublish        `json:"republish"`
	SubjectTransform  *SubjectTransform `json:"subjectTransform"`
	FirstSequence     uint64            `json:"firstSequence"`
	Compression       string            `json:"compression"`
	Retention         string            `json:"retention"`
	Servers           []string          `json:"servers"`
	Sources           []*StreamSource   `json:"sources"`
	Storage           string            `json:"storage"`
	Subjects          []string          `json:"subjects"`
	TLS               TLS               `json:"tls"`
}

type SubjectTransform struct {
	Source string `json:"source"`
	Dest   string `json:"dest"`
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

	SubjectTransforms []*SubjectTransform `json:"subjectTransforms"`
}

type RePublish struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	HeadersOnly bool   `json:"headers_only,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StreamList is a list of Stream resources
type StreamList struct {
	k8smeta.TypeMeta `json:",inline"`
	k8smeta.ListMeta `json:"metadata"`

	Items []Stream `json:"items"`
}
