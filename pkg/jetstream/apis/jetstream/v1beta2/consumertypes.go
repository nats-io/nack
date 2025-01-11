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
	AckPolicy          string            `json:"ackPolicy"`
	AckWait            string            `json:"ackWait"`
	BackOff            []string          `json:"backoff"`
	DeliverGroup       string            `json:"deliverGroup"`
	DeliverPolicy      string            `json:"deliverPolicy"`
	DeliverSubject     string            `json:"deliverSubject"`
	Description        string            `json:"description"`
	PreventDelete      bool              `json:"preventDelete"`
	PreventUpdate      bool              `json:"preventUpdate"`
	DurableName        string            `json:"durableName"`
	FilterSubject      string            `json:"filterSubject"`
	FilterSubjects     []string          `json:"filterSubjects"`
	FlowControl        bool              `json:"flowControl"`
	HeadersOnly        bool              `json:"headersOnly"`
	HeartbeatInterval  string            `json:"heartbeatInterval"`
	MaxAckPending      int               `json:"maxAckPending"`
	MaxDeliver         int               `json:"maxDeliver"`
	MaxRequestBatch    int               `json:"maxRequestBatch"`
	MaxRequestExpires  string            `json:"maxRequestExpires"`
	MaxRequestMaxBytes int               `json:"maxRequestMaxBytes"`
	MaxWaiting         int               `json:"maxWaiting"`
	MemStorage         bool              `json:"memStorage"`
	Name               string            `json:"name"`
	OptStartSeq        int               `json:"optStartSeq"`
	OptStartTime       string            `json:"optStartTime"`
	RateLimitBps       int               `json:"rateLimitBps"`
	ReplayPolicy       string            `json:"replayPolicy"`
	Replicas           int               `json:"replicas"`
	SampleFreq         string            `json:"sampleFreq"`
	StreamName         string            `json:"streamName"`
	Metadata           map[string]string `json:"metadata"`
	BaseStreamConfig
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConsumerList is a list of Consumer resources
type ConsumerList struct {
	k8smeta.TypeMeta `json:",inline"`
	k8smeta.ListMeta `json:"metadata"`

	Items []Consumer `json:"items"`
}
