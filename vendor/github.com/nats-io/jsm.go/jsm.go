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

// Package jsm provides client helpers for managing and interacting with NATS JetStream
package jsm

//go:generate go run api/gen.go

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/nats-io/jsm.go/api"
)

// standard api responses with error embedded
type jetStreamResponseError interface {
	ToError() error
}

// jetstream iterable responses
type apiIterableResponse interface {
	ItemsTotal() int
	ItemsOffset() int
	ItemsLimit() int
	LastPage() bool
}

// jetstream iterable requests
type apiIterableRequest interface {
	SetOffset(o int)
}

// all types generated using the api/gen.go which includes all
// the jetstream api types.  Validate() will force validator all
// of these on every jsonRequest
type apiValidatable interface {
	Validate(...api.StructValidator) (valid bool, errors []string)
	SchemaType() string
}

// IsErrorResponse checks if the message holds a standard JetStream error
func IsErrorResponse(m *nats.Msg) bool {
	if strings.HasPrefix(string(m.Data), api.ErrPrefix) {
		return true
	}

	resp := api.JSApiResponse{}
	err := json.Unmarshal(m.Data, &resp)
	if err != nil {
		return false
	}

	return resp.IsError()
}

// ParseErrorResponse parses the JetStream response, if it's an error returns an error instance holding the message else nil
func ParseErrorResponse(m *nats.Msg) error {
	if !IsErrorResponse(m) {
		return nil
	}

	d := string(m.Data)
	if strings.HasPrefix(d, api.ErrPrefix) {
		return fmt.Errorf(strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(d, api.ErrPrefix), " '"), "'"))
	}

	resp := api.JSApiResponse{}
	err := json.Unmarshal(m.Data, &resp)
	if err != nil {
		return err
	}

	return resp.ToError()
}

// IsOKResponse checks if the message holds a standard JetStream error
func IsOKResponse(m *nats.Msg) bool {
	if strings.HasPrefix(string(m.Data), api.OK) {
		return true
	}

	resp := api.JSApiResponse{}
	err := json.Unmarshal(m.Data, &resp)
	if err != nil {
		return false
	}

	return !resp.IsError()
}

// IsValidName verifies if n is a valid stream, template or consumer name
func IsValidName(n string) bool {
	if n == "" {
		return false
	}

	return !strings.ContainsAny(n, ">*. ")
}

// APISubject returns API subject with prefix applied
func APISubject(subject string, prefix string, domain string) string {
	if domain != "" {
		return fmt.Sprintf("$JS.%s.API", domain) + strings.TrimPrefix(subject, "$JS.API")
	}

	if prefix == "" {
		return subject
	}

	return prefix + strings.TrimPrefix(subject, "$JS.API")
}

// EventSubject returns Event subject with prefix applied
func EventSubject(subject string, prefix string) string {
	if prefix == "" {
		return subject
	}

	return prefix + strings.TrimPrefix(subject, "$JS.EVENT")
}

// ParsePubAck parses a stream publish response and returns an error if the publish failed or parsing failed
func ParsePubAck(m *nats.Msg) (*api.PubAck, error) {
	if m == nil {
		return nil, fmt.Errorf("no message supplied")
	}

	err := ParseErrorResponse(m)
	if err != nil {
		return nil, err
	}

	res := api.PubAck{}
	err = json.Unmarshal(m.Data, &res)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

// IsNatsError checks if err is a ApiErr matching code
func IsNatsError(err error, code uint16) bool {
	if ae, ok := err.(*api.ApiError); ok {
		return ae.NatsErrorCode() == code
	}

	if ae, ok := err.(api.ApiError); ok {
		return ae.NatsErrorCode() == code
	}

	return false
}

// IsInternalStream indicates if a stream is considered 'internal' by the NATS team,
// that is, it's a backing stream for KV, Object or MQTT state
func IsInternalStream(s string) bool {
	return IsKVBucketStream(s) || IsObjectBucketStream(s) || IsMQTTStateStream(s)
}

// IsKVBucketStream determines if a stream is a KV bucket
func IsKVBucketStream(s string) bool {
	return strings.HasPrefix(s, "KV_")
}

// IsObjectBucketStream determines if a stream is a Object bucket
func IsObjectBucketStream(s string) bool {
	return strings.HasPrefix(s, "OBJ_")
}

// IsMQTTStateStream determines if a stream holds internal MQTT state
func IsMQTTStateStream(s string) bool {
	return strings.HasPrefix(s, "$MQTT_")
}

// LinearBackoffPeriods creates a backoff policy without any jitter suitable for use in a consumer backoff policy
//
// The periods start from min and increase linearly until ~max
func LinearBackoffPeriods(steps uint, min time.Duration, max time.Duration) ([]time.Duration, error) {
	if steps == 0 {
		return nil, fmt.Errorf("steps must be more than 0")
	}
	if min == 0 {
		return nil, fmt.Errorf("minimum retry can not be 0")
	}
	if max == 0 {
		return nil, fmt.Errorf("maximum retry can not be 0")
	}

	if max < min {
		max, min = min, max
	}

	var res []time.Duration

	stepSize := uint(max-min) / steps
	for i := uint(0); i < steps; i += 1 {
		res = append(res, min+time.Duration(i*stepSize).Round(time.Millisecond))
	}

	return res, nil
}
