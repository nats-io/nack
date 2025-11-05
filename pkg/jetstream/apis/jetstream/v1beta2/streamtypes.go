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
	Name                   string            `json:"name"`
	Description            string            `json:"description,omitempty"`
	Subjects               []string          `json:"subjects,omitempty"`
	Retention              string            `json:"retention,omitempty"`
	MaxConsumers           int               `json:"maxConsumers,omitempty"`
	MaxMsgsPerSubject      int               `json:"maxMsgsPerSubject,omitempty"`
	MaxMsgs                int               `json:"maxMsgs,omitempty"`
	MaxBytes               int               `json:"maxBytes,omitempty"`
	MaxAge                 string            `json:"maxAge,omitempty"`
	MaxMsgSize             int               `json:"maxMsgSize,omitempty"`
	Storage                string            `json:"storage,omitempty"`
	Discard                string            `json:"discard,omitempty"`
	Replicas               int               `json:"replicas,omitempty"`
	NoAck                  bool              `json:"noAck,omitempty"`
	DuplicateWindow        string            `json:"duplicateWindow,omitempty"` // Maps to Duplicates
	Placement              *StreamPlacement  `json:"placement,omitempty"`
	Mirror                 *StreamSource     `json:"mirror,omitempty"`
	Sources                []*StreamSource   `json:"sources,omitempty"`
	Compression            string            `json:"compression,omitempty"`
	SubjectTransform       *SubjectTransform `json:"subjectTransform,omitempty"`
	RePublish              *RePublish        `json:"republish,omitempty"`
	Sealed                 bool              `json:"sealed,omitempty"`
	DenyDelete             bool              `json:"denyDelete,omitempty"`
	DenyPurge              bool              `json:"denyPurge,omitempty"`
	AllowDirect            bool              `json:"allowDirect,omitempty"`
	AllowRollup            bool              `json:"allowRollup,omitempty"` // Maps to RollupAllowed
	MirrorDirect           bool              `json:"mirrorDirect,omitempty"`
	DiscardPerSubject      bool              `json:"discardPerSubject,omitempty"` // Maps to DiscardNewPer
	FirstSequence          uint64            `json:"firstSequence,omitempty"`     // Maps to FirstSeq
	Metadata               map[string]string `json:"metadata,omitempty"`
	ConsumerLimits         *ConsumerLimits   `json:"consumerLimits,omitempty"`
	AllowMsgTTL            bool              `json:"allowMsgTtl,omitempty"`
	SubjectDeleteMarkerTTL string            `json:"subjectDeleteMarkerTtl,omitempty"`
	AllowMsgCounter        bool              `json:"allowMsgCounter,omitempty"`
	AllowAtomicPublish     bool              `json:"allowAtomicPublish,omitempty"`
	AllowMsgSchedules      bool              `json:"allowMsgSchedules,omitempty"`
	PersistMode            string            `json:"persistMode,omitempty"`
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
	OptStartSeq   int    `json:"optStartSeq,omitempty"`
	OptStartTime  string `json:"optStartTime,omitempty"`
	FilterSubject string `json:"filterSubject,omitempty"`

	ExternalAPIPrefix     string `json:"externalApiPrefix,omitempty"`
	ExternalDeliverPrefix string `json:"externalDeliverPrefix,omitempty"`

	SubjectTransforms []*SubjectTransform `json:"subjectTransforms,omitempty"`
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
