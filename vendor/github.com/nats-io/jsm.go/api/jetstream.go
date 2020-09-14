// Copyright 2020 The NATS Authors
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

package api

import (
	"encoding/json"
	"fmt"
)

// Subjects used by the JetStream API
const (
	JSAuditAdvisory  = "$JS.EVENT.ADVISORY.API"
	JSMetricPrefix   = "$JS.EVENT.METRIC"
	JSAdvisoryPrefix = "$JS.EVENT.ADVISORY"
	JSApiAccountInfo = "$JS.API.INFO"
)

// Responses to requests sent to a server from a client.
const (
	// OK response
	OK = "+OK"
	// ERR prefix response
	ErrPrefix = "-ERR"
)

type DiscardPolicy int

const (
	DiscardOld DiscardPolicy = iota
	DiscardNew
)

func (p DiscardPolicy) String() string {
	switch p {
	case DiscardOld:
		return "Old"
	case DiscardNew:
		return "New"
	default:
		return "Unknown Discard Policy"
	}
}

func (p *DiscardPolicy) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case jsonString("old"):
		*p = DiscardOld
	case jsonString("new"):
		*p = DiscardNew
	default:
		return fmt.Errorf("can not unmarshal %q", data)
	}

	return nil
}

func (p DiscardPolicy) MarshalJSON() ([]byte, error) {
	switch p {
	case DiscardOld:
		return json.Marshal("old")
	case DiscardNew:
		return json.Marshal("new")
	default:
		return nil, fmt.Errorf("unknown discard policy %v", p)
	}
}

type StorageType int

const (
	MemoryStorage StorageType = iota
	FileStorage
)

func (t StorageType) String() string {
	switch t {
	case MemoryStorage:
		return "Memory"
	case FileStorage:
		return "File"
	default:
		return "Unknown Storage Type"
	}
}

func (t *StorageType) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case jsonString("memory"):
		*t = MemoryStorage
	case jsonString("file"):
		*t = FileStorage
	default:
		return fmt.Errorf("can not unmarshal %q", data)
	}

	return nil
}

func (t StorageType) MarshalJSON() ([]byte, error) {
	switch t {
	case MemoryStorage:
		return json.Marshal("memory")
	case FileStorage:
		return json.Marshal("file")
	default:
		return nil, fmt.Errorf("unknown storage type %q", t)
	}
}

type RetentionPolicy int

const (
	LimitsPolicy RetentionPolicy = iota
	InterestPolicy
	WorkQueuePolicy
)

func (p RetentionPolicy) String() string {
	switch p {
	case LimitsPolicy:
		return "Limits"
	case InterestPolicy:
		return "Interest"
	case WorkQueuePolicy:
		return "WorkQueue"
	default:
		return "Unknown Retention Policy"
	}
}

func (p *RetentionPolicy) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case jsonString("limits"):
		*p = LimitsPolicy
	case jsonString("interest"):
		*p = InterestPolicy
	case jsonString("workqueue"):
		*p = WorkQueuePolicy
	default:
		return fmt.Errorf("can not unmarshal %q", data)
	}

	return nil
}

func (p RetentionPolicy) MarshalJSON() ([]byte, error) {
	switch p {
	case LimitsPolicy:
		return json.Marshal("limits")
	case InterestPolicy:
		return json.Marshal("interest")
	case WorkQueuePolicy:
		return json.Marshal("workqueue")
	default:
		return nil, fmt.Errorf("unknown retention policy %q", p)
	}
}

type JSApiIterableRequest struct {
	Offset int `json:"offset"`
}

func (i *JSApiIterableRequest) SetOffset(o int) { i.Offset = o }

type JSApiIterableResponse struct {
	Total  int `json:"total"`
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

func (i JSApiIterableResponse) ItemsTotal() int  { return i.Total }
func (i JSApiIterableResponse) ItemsOffset() int { return i.Offset }
func (i JSApiIterableResponse) ItemsLimit() int  { return i.Limit }
func (i JSApiIterableResponse) LastPage() bool {
	return i.Offset+i.Limit >= i.Total
}

// io.nats.jetstream.api.v1.account_info_response
type JSApiAccountInfoResponse struct {
	JSApiResponse
	*JetStreamAccountStats
}

type JetStreamAccountStats struct {
	Memory  uint64                 `json:"memory"`
	Store   uint64                 `json:"storage"`
	Streams int                    `json:"streams"`
	Limits  JetStreamAccountLimits `json:"limits"`
}

type JetStreamAccountLimits struct {
	MaxMemory    int64 `json:"max_memory"`
	MaxStore     int64 `json:"max_storage"`
	MaxStreams   int   `json:"max_streams"`
	MaxConsumers int   `json:"max_consumers"`
}

type ApiError struct {
	Code        int    `json:"code"`
	Description string `json:"description,omitempty"`
}

// Error implements error
func (e ApiError) Error() string {
	switch {
	case e.Description == "" && e.Code == 0:
		return "unknown JetStream Error"
	case e.Description == "" && e.Code > 0:
		return fmt.Sprintf("unknown JetStream %d Error", e.Code)
	default:
		return e.Description
	}
}

// NotFoundError is true when the error is one about a resource not found
func (e ApiError) NotFoundError() bool { return e.Code == 404 }

// ServerError is true when the server returns a 5xx error code
func (e ApiError) ServerError() bool { return e.Code >= 500 && e.Code < 600 }

// UserError is true when the server returns a 4xx error code
func (e ApiError) UserError() bool { return e.Code >= 400 && e.Code < 500 }

// ErrorCode is the JetStream error code
func (e ApiError) ErrorCode() int { return e.Code }

type JSApiResponse struct {
	Type  string    `json:"type"`
	Error *ApiError `json:"error,omitempty"`
}

// ToError extracts a standard error from a JetStream response
func (r JSApiResponse) ToError() error {
	if r.Error == nil {
		return nil
	}

	return *r.Error
}

// IsError determines if a standard JetStream API response is a error
func (r JSApiResponse) IsError() bool {
	return r.Error != nil
}
