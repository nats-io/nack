// Copyright 2025 The NATS Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1beta2

// ConsumerSpecApplyConfiguration represents a declarative configuration of the ConsumerSpec type for use
// with apply.
type ConsumerSpecApplyConfiguration struct {
	AckPolicy          *string                `json:"ackPolicy,omitempty"`
	AckWait            *string                `json:"ackWait,omitempty"`
	BackOff            []string               `json:"backoff,omitempty"`
	Creds              *string                `json:"creds,omitempty"`
	DeliverGroup       *string                `json:"deliverGroup,omitempty"`
	DeliverPolicy      *string                `json:"deliverPolicy,omitempty"`
	DeliverSubject     *string                `json:"deliverSubject,omitempty"`
	Description        *string                `json:"description,omitempty"`
	PreventDelete      *bool                  `json:"preventDelete,omitempty"`
	PreventUpdate      *bool                  `json:"preventUpdate,omitempty"`
	DurableName        *string                `json:"durableName,omitempty"`
	FilterSubject      *string                `json:"filterSubject,omitempty"`
	FilterSubjects     []string               `json:"filterSubjects,omitempty"`
	FlowControl        *bool                  `json:"flowControl,omitempty"`
	HeadersOnly        *bool                  `json:"headersOnly,omitempty"`
	HeartbeatInterval  *string                `json:"heartbeatInterval,omitempty"`
	MaxAckPending      *int                   `json:"maxAckPending,omitempty"`
	MaxDeliver         *int                   `json:"maxDeliver,omitempty"`
	MaxRequestBatch    *int                   `json:"maxRequestBatch,omitempty"`
	MaxRequestExpires  *string                `json:"maxRequestExpires,omitempty"`
	MaxRequestMaxBytes *int                   `json:"maxRequestMaxBytes,omitempty"`
	MaxWaiting         *int                   `json:"maxWaiting,omitempty"`
	MemStorage         *bool                  `json:"memStorage,omitempty"`
	Nkey               *string                `json:"nkey,omitempty"`
	OptStartSeq        *int                   `json:"optStartSeq,omitempty"`
	OptStartTime       *string                `json:"optStartTime,omitempty"`
	RateLimitBps       *int                   `json:"rateLimitBps,omitempty"`
	ReplayPolicy       *string                `json:"replayPolicy,omitempty"`
	Replicas           *int                   `json:"replicas,omitempty"`
	SampleFreq         *string                `json:"sampleFreq,omitempty"`
	Servers            []string               `json:"servers,omitempty"`
	StreamName         *string                `json:"streamName,omitempty"`
	TLS                *TLSApplyConfiguration `json:"tls,omitempty"`
	Account            *string                `json:"account,omitempty"`
	Metadata           map[string]string      `json:"metadata,omitempty"`
}

// ConsumerSpecApplyConfiguration constructs a declarative configuration of the ConsumerSpec type for use with
// apply.
func ConsumerSpec() *ConsumerSpecApplyConfiguration {
	return &ConsumerSpecApplyConfiguration{}
}

// WithAckPolicy sets the AckPolicy field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the AckPolicy field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithAckPolicy(value string) *ConsumerSpecApplyConfiguration {
	b.AckPolicy = &value
	return b
}

// WithAckWait sets the AckWait field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the AckWait field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithAckWait(value string) *ConsumerSpecApplyConfiguration {
	b.AckWait = &value
	return b
}

// WithBackOff adds the given value to the BackOff field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the BackOff field.
func (b *ConsumerSpecApplyConfiguration) WithBackOff(values ...string) *ConsumerSpecApplyConfiguration {
	for i := range values {
		b.BackOff = append(b.BackOff, values[i])
	}
	return b
}

// WithCreds sets the Creds field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Creds field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithCreds(value string) *ConsumerSpecApplyConfiguration {
	b.Creds = &value
	return b
}

// WithDeliverGroup sets the DeliverGroup field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the DeliverGroup field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithDeliverGroup(value string) *ConsumerSpecApplyConfiguration {
	b.DeliverGroup = &value
	return b
}

// WithDeliverPolicy sets the DeliverPolicy field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the DeliverPolicy field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithDeliverPolicy(value string) *ConsumerSpecApplyConfiguration {
	b.DeliverPolicy = &value
	return b
}

// WithDeliverSubject sets the DeliverSubject field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the DeliverSubject field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithDeliverSubject(value string) *ConsumerSpecApplyConfiguration {
	b.DeliverSubject = &value
	return b
}

// WithDescription sets the Description field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Description field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithDescription(value string) *ConsumerSpecApplyConfiguration {
	b.Description = &value
	return b
}

// WithPreventDelete sets the PreventDelete field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the PreventDelete field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithPreventDelete(value bool) *ConsumerSpecApplyConfiguration {
	b.PreventDelete = &value
	return b
}

// WithPreventUpdate sets the PreventUpdate field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the PreventUpdate field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithPreventUpdate(value bool) *ConsumerSpecApplyConfiguration {
	b.PreventUpdate = &value
	return b
}

// WithDurableName sets the DurableName field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the DurableName field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithDurableName(value string) *ConsumerSpecApplyConfiguration {
	b.DurableName = &value
	return b
}

// WithFilterSubject sets the FilterSubject field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the FilterSubject field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithFilterSubject(value string) *ConsumerSpecApplyConfiguration {
	b.FilterSubject = &value
	return b
}

// WithFilterSubjects adds the given value to the FilterSubjects field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the FilterSubjects field.
func (b *ConsumerSpecApplyConfiguration) WithFilterSubjects(values ...string) *ConsumerSpecApplyConfiguration {
	for i := range values {
		b.FilterSubjects = append(b.FilterSubjects, values[i])
	}
	return b
}

// WithFlowControl sets the FlowControl field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the FlowControl field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithFlowControl(value bool) *ConsumerSpecApplyConfiguration {
	b.FlowControl = &value
	return b
}

// WithHeadersOnly sets the HeadersOnly field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the HeadersOnly field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithHeadersOnly(value bool) *ConsumerSpecApplyConfiguration {
	b.HeadersOnly = &value
	return b
}

// WithHeartbeatInterval sets the HeartbeatInterval field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the HeartbeatInterval field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithHeartbeatInterval(value string) *ConsumerSpecApplyConfiguration {
	b.HeartbeatInterval = &value
	return b
}

// WithMaxAckPending sets the MaxAckPending field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the MaxAckPending field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithMaxAckPending(value int) *ConsumerSpecApplyConfiguration {
	b.MaxAckPending = &value
	return b
}

// WithMaxDeliver sets the MaxDeliver field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the MaxDeliver field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithMaxDeliver(value int) *ConsumerSpecApplyConfiguration {
	b.MaxDeliver = &value
	return b
}

// WithMaxRequestBatch sets the MaxRequestBatch field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the MaxRequestBatch field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithMaxRequestBatch(value int) *ConsumerSpecApplyConfiguration {
	b.MaxRequestBatch = &value
	return b
}

// WithMaxRequestExpires sets the MaxRequestExpires field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the MaxRequestExpires field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithMaxRequestExpires(value string) *ConsumerSpecApplyConfiguration {
	b.MaxRequestExpires = &value
	return b
}

// WithMaxRequestMaxBytes sets the MaxRequestMaxBytes field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the MaxRequestMaxBytes field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithMaxRequestMaxBytes(value int) *ConsumerSpecApplyConfiguration {
	b.MaxRequestMaxBytes = &value
	return b
}

// WithMaxWaiting sets the MaxWaiting field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the MaxWaiting field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithMaxWaiting(value int) *ConsumerSpecApplyConfiguration {
	b.MaxWaiting = &value
	return b
}

// WithMemStorage sets the MemStorage field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the MemStorage field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithMemStorage(value bool) *ConsumerSpecApplyConfiguration {
	b.MemStorage = &value
	return b
}

// WithNkey sets the Nkey field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Nkey field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithNkey(value string) *ConsumerSpecApplyConfiguration {
	b.Nkey = &value
	return b
}

// WithOptStartSeq sets the OptStartSeq field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the OptStartSeq field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithOptStartSeq(value int) *ConsumerSpecApplyConfiguration {
	b.OptStartSeq = &value
	return b
}

// WithOptStartTime sets the OptStartTime field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the OptStartTime field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithOptStartTime(value string) *ConsumerSpecApplyConfiguration {
	b.OptStartTime = &value
	return b
}

// WithRateLimitBps sets the RateLimitBps field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the RateLimitBps field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithRateLimitBps(value int) *ConsumerSpecApplyConfiguration {
	b.RateLimitBps = &value
	return b
}

// WithReplayPolicy sets the ReplayPolicy field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ReplayPolicy field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithReplayPolicy(value string) *ConsumerSpecApplyConfiguration {
	b.ReplayPolicy = &value
	return b
}

// WithReplicas sets the Replicas field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Replicas field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithReplicas(value int) *ConsumerSpecApplyConfiguration {
	b.Replicas = &value
	return b
}

// WithSampleFreq sets the SampleFreq field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the SampleFreq field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithSampleFreq(value string) *ConsumerSpecApplyConfiguration {
	b.SampleFreq = &value
	return b
}

// WithServers adds the given value to the Servers field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Servers field.
func (b *ConsumerSpecApplyConfiguration) WithServers(values ...string) *ConsumerSpecApplyConfiguration {
	for i := range values {
		b.Servers = append(b.Servers, values[i])
	}
	return b
}

// WithStreamName sets the StreamName field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the StreamName field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithStreamName(value string) *ConsumerSpecApplyConfiguration {
	b.StreamName = &value
	return b
}

// WithTLS sets the TLS field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the TLS field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithTLS(value *TLSApplyConfiguration) *ConsumerSpecApplyConfiguration {
	b.TLS = value
	return b
}

// WithAccount sets the Account field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Account field is set to the value of the last call.
func (b *ConsumerSpecApplyConfiguration) WithAccount(value string) *ConsumerSpecApplyConfiguration {
	b.Account = &value
	return b
}

// WithMetadata puts the entries into the Metadata field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, the entries provided by each call will be put on the Metadata field,
// overwriting an existing map entries in Metadata field with the same key.
func (b *ConsumerSpecApplyConfiguration) WithMetadata(entries map[string]string) *ConsumerSpecApplyConfiguration {
	if b.Metadata == nil && len(entries) > 0 {
		b.Metadata = make(map[string]string, len(entries))
	}
	for k, v := range entries {
		b.Metadata[k] = v
	}
	return b
}
