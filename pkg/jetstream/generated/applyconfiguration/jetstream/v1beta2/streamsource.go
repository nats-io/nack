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

import (
	jetstreamv1beta2 "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
)

// StreamSourceApplyConfiguration represents a declarative configuration of the StreamSource type for use
// with apply.
type StreamSourceApplyConfiguration struct {
	Name                  *string                              `json:"name,omitempty"`
	OptStartSeq           *int                                 `json:"optStartSeq,omitempty"`
	OptStartTime          *string                              `json:"optStartTime,omitempty"`
	FilterSubject         *string                              `json:"filterSubject,omitempty"`
	ExternalAPIPrefix     *string                              `json:"externalApiPrefix,omitempty"`
	ExternalDeliverPrefix *string                              `json:"externalDeliverPrefix,omitempty"`
	SubjectTransforms     []*jetstreamv1beta2.SubjectTransform `json:"subjectTransforms,omitempty"`
}

// StreamSourceApplyConfiguration constructs a declarative configuration of the StreamSource type for use with
// apply.
func StreamSource() *StreamSourceApplyConfiguration {
	return &StreamSourceApplyConfiguration{}
}

// WithName sets the Name field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Name field is set to the value of the last call.
func (b *StreamSourceApplyConfiguration) WithName(value string) *StreamSourceApplyConfiguration {
	b.Name = &value
	return b
}

// WithOptStartSeq sets the OptStartSeq field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the OptStartSeq field is set to the value of the last call.
func (b *StreamSourceApplyConfiguration) WithOptStartSeq(value int) *StreamSourceApplyConfiguration {
	b.OptStartSeq = &value
	return b
}

// WithOptStartTime sets the OptStartTime field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the OptStartTime field is set to the value of the last call.
func (b *StreamSourceApplyConfiguration) WithOptStartTime(value string) *StreamSourceApplyConfiguration {
	b.OptStartTime = &value
	return b
}

// WithFilterSubject sets the FilterSubject field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the FilterSubject field is set to the value of the last call.
func (b *StreamSourceApplyConfiguration) WithFilterSubject(value string) *StreamSourceApplyConfiguration {
	b.FilterSubject = &value
	return b
}

// WithExternalAPIPrefix sets the ExternalAPIPrefix field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ExternalAPIPrefix field is set to the value of the last call.
func (b *StreamSourceApplyConfiguration) WithExternalAPIPrefix(value string) *StreamSourceApplyConfiguration {
	b.ExternalAPIPrefix = &value
	return b
}

// WithExternalDeliverPrefix sets the ExternalDeliverPrefix field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ExternalDeliverPrefix field is set to the value of the last call.
func (b *StreamSourceApplyConfiguration) WithExternalDeliverPrefix(value string) *StreamSourceApplyConfiguration {
	b.ExternalDeliverPrefix = &value
	return b
}

// WithSubjectTransforms adds the given value to the SubjectTransforms field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the SubjectTransforms field.
func (b *StreamSourceApplyConfiguration) WithSubjectTransforms(values ...**jetstreamv1beta2.SubjectTransform) *StreamSourceApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithSubjectTransforms")
		}
		b.SubjectTransforms = append(b.SubjectTransforms, *values[i])
	}
	return b
}
