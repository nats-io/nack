package v1beta2

import (
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:storageversion

// Consumer is a specification for a Consumer resource
type Consumer struct {
	k8smeta.TypeMeta   `json:",inline"`
	k8smeta.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConsumerSpec `json:"spec"`
	Status Status       `json:"status"`
}

func (c *Consumer) GetSpec() interface{} {
	return c.Spec
}

// ConsumerSpec is the spec for a Consumer resource
type ConsumerSpec struct {
	Description        string            `json:"description,omitempty"`
	AckPolicy          string            `json:"ackPolicy,omitempty"`
	AckWait            string            `json:"ackWait,omitempty"`
	DeliverPolicy      string            `json:"deliverPolicy,omitempty"`
	DeliverSubject     string            `json:"deliverSubject,omitempty"`
	DeliverGroup       string            `json:"deliverGroup,omitempty"`
	DurableName        string            `json:"durableName,omitempty"` // Maps to Durable
	FilterSubject      string            `json:"filterSubject,omitempty"`
	FilterSubjects     []string          `json:"filterSubjects,omitempty"`
	FlowControl        bool              `json:"flowControl,omitempty"`
	HeartbeatInterval  string            `json:"heartbeatInterval,omitempty"` // Maps to IdleHeartbeat
	MaxAckPending      int               `json:"maxAckPending,omitempty"`
	MaxDeliver         int               `json:"maxDeliver,omitempty"`
	BackOff            []string          `json:"backoff,omitempty"`
	MaxWaiting         int               `json:"maxWaiting,omitempty"`
	OptStartSeq        int               `json:"optStartSeq,omitempty"`
	OptStartTime       string            `json:"optStartTime,omitempty"`
	RateLimitBps       int               `json:"rateLimitBps,omitempty"` // Maps to RateLimit
	ReplayPolicy       string            `json:"replayPolicy,omitempty"`
	SampleFreq         string            `json:"sampleFreq,omitempty"` // Maps to SampleFrequency
	HeadersOnly        bool              `json:"headersOnly,omitempty"`
	MaxRequestBatch    int               `json:"maxRequestBatch,omitempty"`
	MaxRequestExpires  string            `json:"maxRequestExpires,omitempty"`
	MaxRequestMaxBytes int               `json:"maxRequestMaxBytes,omitempty"`
	InactiveThreshold  string            `json:"inactiveThreshold,omitempty"`
	Replicas           int               `json:"replicas,omitempty"`
	MemStorage         bool              `json:"memStorage,omitempty"` // Maps to MemoryStorage
	Metadata           map[string]string `json:"metadata,omitempty"`
	PauseUntil         string            `json:"pauseUntil,omitempty"`     // RFC3339 timestamp for pausing consumer
	PriorityPolicy     string            `json:"priorityPolicy,omitempty"` // Priority policy: none, pinned_client, overflow, prioritized
	PinnedTTL          string            `json:"pinnedTtl,omitempty"`      // Duration for pinned client timeout
	PriorityGroups     []string          `json:"priorityGroups,omitempty"` // List of priority groups

	StreamName       string `json:"streamName,omitempty"`
	BaseStreamConfig `json:"baseStreamConfig"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConsumerList is a list of Consumer resources
type ConsumerList struct {
	k8smeta.TypeMeta `json:",inline"`
	k8smeta.ListMeta `json:"metadata"`

	Items []Consumer `json:"items"`
}
