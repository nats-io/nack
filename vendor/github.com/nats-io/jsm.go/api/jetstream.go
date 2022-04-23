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
	// ErrPrefix is the ERR prefix response
	ErrPrefix = "-ERR"
)

// Headers for publishing
const (
	// JSMsgId used for tracking duplicates
	JSMsgId = "Nats-Msg-Id"

	// JSExpectedStream only store the message in this stream
	JSExpectedStream = "Nats-Expected-Stream"

	// JSExpectedLastSeq only store the message if stream last sequence matched
	JSExpectedLastSeq = "Nats-Expected-Last-Sequence"

	// JSExpectedLastSubjSeq only stores the message if last sequence for this subject matched
	JSExpectedLastSubjSeq = "Nats-Expected-Last-Subject-Sequence"

	// JSExpectedLastMsgId only stores the message if previous Nats-Msg-Id header value matches this
	JSExpectedLastMsgId = "Nats-Expected-Last-Msg-Id"

	// JSRollup is a header indicating the message being sent should be stored and all past messags should be discarded
	// the value can be either `all` or `subject`
	JSRollup = "Nats-Rollup"

	// JSRollupAll is the value for JSRollup header to replace the entire stream
	JSRollupAll = "all"

	// JSRollupSubject is the value for JSRollup header to replace the a single subject
	JSRollupSubject = "sub"
)

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

type JetStreamTier struct {
	Memory    uint64                 `json:"memory"`
	Store     uint64                 `json:"storage"`
	Streams   int                    `json:"streams"`
	Consumers int                    `json:"consumers"`
	Limits    JetStreamAccountLimits `json:"limits"`
}

// JetStreamAccountStats returns current statistics about the account's JetStream usage.
type JetStreamAccountStats struct {
	JetStreamTier                          // in case tiers are used, reflects totals with limits not set
	Domain        string                   `json:"domain,omitempty"`
	API           JetStreamAPIStats        `json:"api"`
	Tiers         map[string]JetStreamTier `json:"tiers,omitempty"` // indexed by tier name
}

type JetStreamAPIStats struct {
	Total  uint64 `json:"total"`
	Errors uint64 `json:"errors"`
}

type JetStreamAccountLimits struct {
	MaxMemory            int64 `json:"max_memory"`
	MaxStore             int64 `json:"max_storage"`
	MaxStreams           int   `json:"max_streams"`
	MaxConsumers         int   `json:"max_consumers"`
	MaxAckPending        int   `json:"max_ack_pending"`
	MemoryMaxStreamBytes int64 `json:"memory_max_stream_bytes"`
	StoreMaxStreamBytes  int64 `json:"storage_max_stream_bytes"`
	MaxBytesRequired     bool  `json:"max_bytes_required"`
}

type ApiError struct {
	Code        int    `json:"code"`
	ErrCode     uint16 `json:"err_code,omitempty"`
	Description string `json:"description,omitempty"`
}

// Error implements error
func (e ApiError) Error() string {
	switch {
	case e.Description == "" && e.Code == 0:
		return "unknown JetStream Error"
	case e.Description == "" && e.Code > 0:
		return fmt.Sprintf("unknown JetStream %d Error (%d)", e.Code, e.ErrCode)
	default:
		return fmt.Sprintf("%s (%d)", e.Description, e.ErrCode)
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

// NatsErrorCode is the unique nats error code, see `nats errors` command
func (e ApiError) NatsErrorCode() uint16 { return e.ErrCode }

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

// IsNatsErr determines if a error matches ID, if multiple IDs are given if the error matches any of these the function will be true
func IsNatsErr(err error, ids ...uint16) bool {
	if err == nil {
		return false
	}

	ce, ok := err.(ApiError)
	if !ok {
		return false
	}

	for _, id := range ids {
		if ce.ErrCode == id {
			return true
		}
	}

	return false
}
