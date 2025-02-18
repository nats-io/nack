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
	Name              string            `json:"name"`
	Description       string            `json:"description"`
	Subjects          []string          `json:"subjects"`
	Retention         string            `json:"retention"`
	MaxConsumers      int               `json:"maxConsumers"`
	MaxMsgsPerSubject int               `json:"maxMsgsPerSubject"`
	MaxMsgs           int               `json:"maxMsgs"`
	MaxBytes          int               `json:"maxBytes"`
	MaxAge            string            `json:"maxAge"`
	MaxMsgSize        int               `json:"maxMsgSize"`
	Storage           string            `json:"storage"`
	Discard           string            `json:"discard"`
	Replicas          int               `json:"replicas"`
	NoAck             bool              `json:"noAck"`
	DuplicateWindow   string            `json:"duplicateWindow"` // Maps to Duplicates
	Placement         *StreamPlacement  `json:"placement"`
	Mirror            *StreamSource     `json:"mirror"`
	Sources           []*StreamSource   `json:"sources"`
	Compression       string            `json:"compression"`
	SubjectTransform  *SubjectTransform `json:"subjectTransform"`
	RePublish         *RePublish        `json:"republish"`
	Sealed            bool              `json:"sealed"`
	DenyDelete        bool              `json:"denyDelete"`
	DenyPurge         bool              `json:"denyPurge"`
	AllowDirect       bool              `json:"allowDirect"`
	AllowRollup       bool              `json:"allowRollup"` // Maps to RollupAllowed
	MirrorDirect      bool              `json:"mirrorDirect"`
	DiscardPerSubject bool              `json:"discardPerSubject"` // Maps to DiscardNewPer
	FirstSequence     uint64            `json:"firstSequence"`     // Maps to FirstSeq
	Metadata          map[string]string `json:"metadata"`
	ConsumerLimits    *ConsumerLimits   `json:"consumerLimits"`
	BaseStreamConfig
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
