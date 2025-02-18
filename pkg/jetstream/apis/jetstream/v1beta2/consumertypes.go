package v1beta2

import (
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

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
	Description        string            `json:"description"`
	AckPolicy          string            `json:"ackPolicy"`
	AckWait            string            `json:"ackWait"`
	DeliverPolicy      string            `json:"deliverPolicy"`
	DeliverSubject     string            `json:"deliverSubject"`
	DeliverGroup       string            `json:"deliverGroup"`
	DurableName        string            `json:"durableName"` // Maps to Durable
	FilterSubject      string            `json:"filterSubject"`
	FilterSubjects     []string          `json:"filterSubjects"`
	FlowControl        bool              `json:"flowControl"`
	HeartbeatInterval  string            `json:"heartbeatInterval"` // Maps to Heartbeat
	MaxAckPending      int               `json:"maxAckPending"`
	MaxDeliver         int               `json:"maxDeliver"`
	BackOff            []string          `json:"backoff"`
	MaxWaiting         int               `json:"maxWaiting"`
	OptStartSeq        int               `json:"optStartSeq"`
	OptStartTime       string            `json:"optStartTime"`
	RateLimitBps       int               `json:"rateLimitBps"` // Maps to RateLimit
	ReplayPolicy       string            `json:"replayPolicy"`
	SampleFreq         string            `json:"sampleFreq"` // Maps to SampleFrequency
	HeadersOnly        bool              `json:"headersOnly"`
	MaxRequestBatch    int               `json:"maxRequestBatch"`
	MaxRequestExpires  string            `json:"maxRequestExpires"`
	MaxRequestMaxBytes int               `json:"maxRequestMaxBytes"`
	InactiveThreshold  string            `json:"inactiveThreshold"`
	Replicas           int               `json:"replicas"`
	MemStorage         bool              `json:"memStorage"` // Maps to MemoryStorage
	Metadata           map[string]string `json:"metadata"`

	StreamName string `json:"streamName"`
	BaseStreamConfig
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConsumerList is a list of Consumer resources
type ConsumerList struct {
	k8smeta.TypeMeta `json:",inline"`
	k8smeta.ListMeta `json:"metadata"`

	Items []Consumer `json:"items"`
}
