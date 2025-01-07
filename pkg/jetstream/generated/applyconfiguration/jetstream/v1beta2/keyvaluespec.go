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

// KeyValueSpecApplyConfiguration represents a declarative configuration of the KeyValueSpec type for use
// with apply.
type KeyValueSpecApplyConfiguration struct {
	Bucket       *string                            `json:"bucket,omitempty"`
	Description  *string                            `json:"description,omitempty"`
	MaxValueSize *int                               `json:"maxValueSize,omitempty"`
	History      *int                               `json:"history,omitempty"`
	TTL          *string                            `json:"ttl,omitempty"`
	MaxBytes     *int                               `json:"maxBytes,omitempty"`
	Storage      *string                            `json:"storage,omitempty"`
	Replicas     *int                               `json:"replicas,omitempty"`
	Placement    *StreamPlacementApplyConfiguration `json:"placement,omitempty"`
	RePublish    *RePublishApplyConfiguration       `json:"republish,omitempty"`
	Mirror       *StreamSourceApplyConfiguration    `json:"mirror,omitempty"`
	Sources      []*jetstreamv1beta2.StreamSource   `json:"sources,omitempty"`
	Compression  *bool                              `json:"compression,omitempty"`
}

// KeyValueSpecApplyConfiguration constructs a declarative configuration of the KeyValueSpec type for use with
// apply.
func KeyValueSpec() *KeyValueSpecApplyConfiguration {
	return &KeyValueSpecApplyConfiguration{}
}

// WithBucket sets the Bucket field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Bucket field is set to the value of the last call.
func (b *KeyValueSpecApplyConfiguration) WithBucket(value string) *KeyValueSpecApplyConfiguration {
	b.Bucket = &value
	return b
}

// WithDescription sets the Description field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Description field is set to the value of the last call.
func (b *KeyValueSpecApplyConfiguration) WithDescription(value string) *KeyValueSpecApplyConfiguration {
	b.Description = &value
	return b
}

// WithMaxValueSize sets the MaxValueSize field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the MaxValueSize field is set to the value of the last call.
func (b *KeyValueSpecApplyConfiguration) WithMaxValueSize(value int) *KeyValueSpecApplyConfiguration {
	b.MaxValueSize = &value
	return b
}

// WithHistory sets the History field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the History field is set to the value of the last call.
func (b *KeyValueSpecApplyConfiguration) WithHistory(value int) *KeyValueSpecApplyConfiguration {
	b.History = &value
	return b
}

// WithTTL sets the TTL field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the TTL field is set to the value of the last call.
func (b *KeyValueSpecApplyConfiguration) WithTTL(value string) *KeyValueSpecApplyConfiguration {
	b.TTL = &value
	return b
}

// WithMaxBytes sets the MaxBytes field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the MaxBytes field is set to the value of the last call.
func (b *KeyValueSpecApplyConfiguration) WithMaxBytes(value int) *KeyValueSpecApplyConfiguration {
	b.MaxBytes = &value
	return b
}

// WithStorage sets the Storage field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Storage field is set to the value of the last call.
func (b *KeyValueSpecApplyConfiguration) WithStorage(value string) *KeyValueSpecApplyConfiguration {
	b.Storage = &value
	return b
}

// WithReplicas sets the Replicas field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Replicas field is set to the value of the last call.
func (b *KeyValueSpecApplyConfiguration) WithReplicas(value int) *KeyValueSpecApplyConfiguration {
	b.Replicas = &value
	return b
}

// WithPlacement sets the Placement field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Placement field is set to the value of the last call.
func (b *KeyValueSpecApplyConfiguration) WithPlacement(value *StreamPlacementApplyConfiguration) *KeyValueSpecApplyConfiguration {
	b.Placement = value
	return b
}

// WithRePublish sets the RePublish field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the RePublish field is set to the value of the last call.
func (b *KeyValueSpecApplyConfiguration) WithRePublish(value *RePublishApplyConfiguration) *KeyValueSpecApplyConfiguration {
	b.RePublish = value
	return b
}

// WithMirror sets the Mirror field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Mirror field is set to the value of the last call.
func (b *KeyValueSpecApplyConfiguration) WithMirror(value *StreamSourceApplyConfiguration) *KeyValueSpecApplyConfiguration {
	b.Mirror = value
	return b
}

// WithSources adds the given value to the Sources field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Sources field.
func (b *KeyValueSpecApplyConfiguration) WithSources(values ...**jetstreamv1beta2.StreamSource) *KeyValueSpecApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithSources")
		}
		b.Sources = append(b.Sources, *values[i])
	}
	return b
}

// WithCompression sets the Compression field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Compression field is set to the value of the last call.
func (b *KeyValueSpecApplyConfiguration) WithCompression(value bool) *KeyValueSpecApplyConfiguration {
	b.Compression = &value
	return b
}
