package v1beta1

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
a	AckPolicy         string   `json:"ackPolicy"`
	AckWait           string   `json:"ackWait"`
	Creds             string   `json:"creds"`
	DeliverGroup      string   `json:"deliverGroup"`
	DeliverPolicy     string   `json:"deliverPolicy"`
	DeliverSubject    string   `json:"deliverSubject"`
	Description       string   `json:"description"`
	DurableName       string   `json:"durableName"`
	FilterSubject     string   `json:"filterSubject"`
	FlowControl       bool     `json:"flowControl"`
	HeartbeatInterval string   `json:"heartbeatInterval"`
	MaxAckPending     int      `json:"maxAckPending"`
	MaxDeliver        int      `json:"maxDeliver"`
	Nkey              string   `json:"nkey"`
	OptStartSeq       int      `json:"optStartSeq"`
	OptStartTime      string   `json:"optStartTime"`
	RateLimitBps      int      `json:"rateLimitBps"`
	ReplayPolicy      string   `json:"replayPolicy"`
	SampleFreq        string   `json:"sampleFreq"`
	Servers           []string `json:"servers"`
	StreamName        string   `json:"streamName"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConsumerList is a list of Consumer resources
type ConsumerList struct {
	k8smeta.TypeMeta `json:",inline"`
	k8smeta.ListMeta `json:"metadata"`

	Items []Consumer `json:"items"`
}
