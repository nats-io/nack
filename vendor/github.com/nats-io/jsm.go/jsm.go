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
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/nats-io/jsm.go/api"
)

var timeout = 5 * time.Second
var nc *nats.Conn
var mu sync.Mutex
var trace bool
var validate bool

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
// the jetstream api types.  Validate() will force validate all
// of these on every jsonRequest
type apiValidatable interface {
	Validate() (valid bool, errors []string)
	SchemaType() string
}

// Connect connects to NATS and configures it to use the connection in future interactions with JetStream
// Deprecated: Use Request Options to supply the connection
func Connect(servers string, opts ...nats.Option) (err error) {
	mu.Lock()
	defer mu.Unlock()

	// needed so that interest drops are observed by JetStream to stop
	opts = append(opts, nats.UseOldRequestStyle())

	nc, err = nats.Connect(servers, opts...)

	return err
}

// SetTimeout sets the timeout for requests to JetStream
// Deprecated: Use Request Options to supply the timeout
func SetTimeout(t time.Duration) {
	mu.Lock()
	defer mu.Unlock()

	timeout = t
}

// Validate enables JSON Schema validation of all responses from the Server
func Validate() {
	mu.Lock()
	validate = true
	mu.Unlock()
}

// NoValidate disables Validate()
func NoValidate() {
	mu.Lock()
	validate = true
	mu.Unlock()
}

// Trace produce logs for all interactions with the server
func Trace() {
	mu.Lock()
	trace = true
	mu.Unlock()
}

// NoTrace disables Trace()
func NoTrace() {
	mu.Lock()
	trace = false
	mu.Unlock()
}

func shouldTrace() bool {
	mu.Lock()
	defer mu.Unlock()
	return trace
}

func shouldValidate() bool {
	mu.Lock()
	defer mu.Unlock()
	return validate
}

// SetConnection sets the connection used to perform requests. Will force using old style requests.
// Deprecated: Use Request Options to supply the connection
func SetConnection(c *nats.Conn) {
	mu.Lock()
	defer mu.Unlock()

	c.Opts.UseOldRequestStyle = true

	nc = c
}

// IsJetStreamEnabled determines if JetStream is enabled for the current account
func IsJetStreamEnabled(opts ...RequestOption) bool {
	info, err := JetStreamAccountInfo(opts...)
	if err != nil {
		return false
	}

	if info == nil {
		return false
	}

	return true
}

// IsErrorResponse checks if the message holds a standard JetStream error
// TODO: parse for error response
func IsErrorResponse(m *nats.Msg) bool {
	return strings.HasPrefix(string(m.Data), api.ErrPrefix)
}

// ParseErrorResponse parses the JetStream response, if it's an error returns an error instance holding the message else nil
// TODO: parse json error response
func ParseErrorResponse(m *nats.Msg) error {
	if !IsErrorResponse(m) {
		return nil
	}

	return fmt.Errorf(strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(string(m.Data), api.ErrPrefix), " '"), "'"))
}

// IsOKResponse checks if the message holds a standard JetStream error
// TODO: parse json responses
func IsOKResponse(m *nats.Msg) bool {
	return strings.HasPrefix(string(m.Data), api.OK)
}

// IsKnownStream determines if a Stream is known
func IsKnownStream(stream string, opts ...RequestOption) (bool, error) {
	s, err := LoadStream(stream, opts...)
	if err != nil {
		jserr, ok := err.(api.ApiError)
		if ok {
			if jserr.NotFoundError() {
				return false, nil
			}
		}

		return false, err
	}

	if s.Name() != stream {
		return false, fmt.Errorf("received invalid stream from load")
	}

	return true, nil
}

// IsKnownStreamTemplate determines if a StreamTemplate is known
func IsKnownStreamTemplate(template string, opts ...RequestOption) (bool, error) {
	t, err := LoadStreamTemplate(template, opts...)
	if err != nil {
		jserr, ok := err.(api.ApiError)
		if ok {
			if jserr.NotFoundError() {
				return false, nil
			}
		}

		return false, err
	}

	if t.Name() != template {
		return false, fmt.Errorf("received invalid stream template from load")
	}

	return true, nil
}

// IsKnownConsumer determines if a Consumer is known for a specific Stream
func IsKnownConsumer(stream string, consumer string, opts ...RequestOption) (bool, error) {
	c, err := LoadConsumer(stream, consumer, opts...)
	if err != nil {
		jserr, ok := err.(api.ApiError)
		if ok {
			if jserr.NotFoundError() {
				return false, nil
			}
		}

		return false, err
	}

	if c.Name() != consumer {
		return false, fmt.Errorf("invalid consumer received from load")
	}

	return true, nil
}

// JetStreamAccountInfo retrieves information about the current account limits and more
func JetStreamAccountInfo(opts ...RequestOption) (info *api.JetStreamAccountStats, err error) {
	conn, err := newreqoptions(opts...)
	if err != nil {
		return nil, err
	}

	var resp api.JSApiAccountInfoResponse
	err = jsonRequest(api.JSApiAccountInfo, nil, &resp, conn)
	if err != nil {
		return nil, err
	}

	return resp.JetStreamAccountStats, nil
}

// Streams is a sorted list of all known Streams
func Streams(opts ...RequestOption) (streams []*Stream, err error) {
	conn, err := newreqoptions(opts...)
	if err != nil {
		return nil, err
	}

	var resp api.JSApiStreamListResponse
	err = iterableRequest(api.JSApiStreamList, &api.JSApiStreamListRequest{JSApiIterableRequest: api.JSApiIterableRequest{Offset: 0}}, &resp, conn, func(page interface{}) error {
		apiresp, ok := page.(*api.JSApiStreamListResponse)
		if !ok {
			return fmt.Errorf("invalid response type from iterable request")
		}

		sort.Slice(apiresp.Streams, func(i int, j int) bool {
			return apiresp.Streams[i].Config.Name < apiresp.Streams[j].Config.Name
		})

		for _, s := range apiresp.Streams {
			cfg := &StreamConfig{
				StreamConfig: s.Config,
				conn:         conn,
				ropts:        opts,
			}

			streams = append(streams, streamFromConfig(cfg, s))
		}

		return nil
	})
	if err != nil {
		return streams, err
	}

	return streams, nil
}

// StreamNames is a sorted list of all known Streams
func StreamNames(opts ...RequestOption) (names []string, err error) {
	conn, err := newreqoptions(opts...)
	if err != nil {
		return nil, err
	}

	var resp api.JSApiStreamNamesResponse
	err = iterableRequest(api.JSApiStreamNames, &api.JSApiStreamNamesRequest{JSApiIterableRequest: api.JSApiIterableRequest{Offset: 0}}, &resp, conn, func(page interface{}) error {
		apiresp, ok := page.(*api.JSApiStreamNamesResponse)
		if !ok {
			return fmt.Errorf("invalid response type from iterable request")
		}

		names = append(names, apiresp.Streams...)

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(names)

	return names, nil
}

// StreamNames is a sorted list of all known consumers within a stream
func ConsumerNames(stream string, opts ...RequestOption) (names []string, err error) {
	if !IsValidName(stream) {
		return nil, fmt.Errorf("%q is not a valid stream name", stream)
	}

	conn, err := newreqoptions(opts...)
	if err != nil {
		return nil, err
	}

	var resp api.JSApiConsumerNamesResponse
	err = iterableRequest(fmt.Sprintf(api.JSApiConsumerNamesT, stream), &api.JSApiConsumerNamesRequest{JSApiIterableRequest: api.JSApiIterableRequest{Offset: 0}}, &resp, conn, func(page interface{}) error {
		apiresp, ok := page.(*api.JSApiConsumerNamesResponse)
		if !ok {
			return fmt.Errorf("invalid response type from iterable request")
		}

		names = append(names, apiresp.Consumers...)

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(names)

	return names, nil
}

// StreamTemplateNames is a sorted list of all known StreamTemplates
func StreamTemplateNames(opts ...RequestOption) (templates []string, err error) {
	conn, err := newreqoptions(opts...)
	if err != nil {
		return nil, err
	}

	var resp api.JSApiStreamTemplateNamesResponse
	err = iterableRequest(api.JSApiTemplateNames, &api.JSApiStreamTemplateNamesRequest{JSApiIterableRequest: api.JSApiIterableRequest{Offset: 0}}, &resp, conn, func(page interface{}) error {
		apiresp, ok := page.(*api.JSApiStreamTemplateNamesResponse)
		if !ok {
			return fmt.Errorf("invalid response type from iterable request")
		}

		templates = append(templates, apiresp.Templates...)

		return nil
	})
	if err != nil {
		return templates, err
	}

	sort.Strings(templates)

	return templates, nil
}

// Consumers is a sorted list of all known Consumers within a Stream
func Consumers(stream string, opts ...RequestOption) (consumers []*Consumer, err error) {
	if !IsValidName(stream) {
		return nil, fmt.Errorf("%q is not a valid stream name", stream)
	}

	conn, err := newreqoptions(opts...)
	if err != nil {
		return nil, err
	}

	var cinfo []*api.ConsumerInfo

	var resp api.JSApiConsumerListResponse
	err = iterableRequest(fmt.Sprintf(api.JSApiConsumerListT, stream), &api.JSApiConsumerListRequest{JSApiIterableRequest: api.JSApiIterableRequest{Offset: 0}}, &resp, conn, func(page interface{}) error {
		apiresp, ok := page.(*api.JSApiConsumerListResponse)
		if !ok {
			return fmt.Errorf("invalid response type from iterable request")
		}

		cinfo = append(cinfo, apiresp.Consumers...)
		return nil
	})
	if err != nil {
		return consumers, err
	}

	sort.Slice(cinfo, func(i int, j int) bool {
		return resp.Consumers[i].Name < resp.Consumers[j].Name
	})

	for _, c := range cinfo {
		cfg := &ConsumerCfg{
			ConsumerConfig: c.Config,
			conn:           conn,
			ropts:          opts,
		}

		consumers = append(consumers, consumerFromCfg(c.Stream, c.Name, cfg))
	}

	return consumers, nil
}

// EachStream iterates over all known Streams
func EachStream(cb func(*Stream), opts ...RequestOption) (err error) {
	streams, err := Streams()
	if err != nil {
		return err
	}

	for _, s := range streams {
		cb(s)
	}

	return nil
}

// EachStreamTemplate iterates over all known Stream Templates
func EachStreamTemplate(cb func(*StreamTemplate), opts ...RequestOption) (err error) {
	names, err := StreamTemplateNames(opts...)
	if err != nil {
		return err
	}

	for _, t := range names {
		template, err := LoadStreamTemplate(t, opts...)
		if err != nil {
			return err
		}

		cb(template)
	}

	return nil
}

// Flush flushes the underlying NATS connection
// Deprecated: Use Request Options to supply the connection
func Flush() error {
	nc := Connection()

	if nc == nil {
		return fmt.Errorf("nats connection is not set, use SetConnection()")
	}

	return nc.Flush()
}

// Connection is the active NATS connection being used
// Deprecated: Use Request Options to supply the connection
func Connection() *nats.Conn {
	mu.Lock()
	defer mu.Unlock()

	return nc
}

func iterableRequest(subj string, req apiIterableRequest, response apiIterableResponse, opts *reqoptions, cb func(interface{}) error) (err error) {
	offset := 0
	for {
		req.SetOffset(offset)

		err = jsonRequest(subj, req, response, opts)
		if err != nil {
			return err
		}

		err = cb(response)
		if err != nil {
			return err
		}

		if response.LastPage() {
			break
		}

		offset += response.ItemsLimit()
	}

	return nil
}

func jsonRequest(subj string, req interface{}, response interface{}, opts *reqoptions) (err error) {
	var body []byte

	switch {
	case req == nil:
		body = []byte("")
	default:
		body, err = json.Marshal(req)
		if err != nil {
			return err
		}
	}

	if opts.trace {
		log.Printf(">>> %s\n%s\n\n", subj, string(body))
	}

	msg, err := request(subj, body, opts)
	if err != nil {
		return err
	}

	if opts.trace {
		log.Printf("<<< %s\n%s\n\n", subj, string(msg.Data))
	}

	err = json.Unmarshal(msg.Data, response)
	if err != nil {
		return err
	}

	jsr, ok := response.(jetStreamResponseError)
	if !ok {
		return nil
	}

	if jsr.ToError() != nil {
		return jsr.ToError()
	}

	if !opts.apiValidate {
		return nil
	}

	jv, ok := response.(apiValidatable)
	if ok {
		valid, errs := jv.Validate()
		if valid {
			return nil
		}

		return fmt.Errorf("server response is not a valid %q message: %s", jv.SchemaType(), strings.Join(errs, "\n"))
	} else {
		return fmt.Errorf("validation is required but an unknown response were received")
	}
}

func request(subj string, data []byte, opts *reqoptions) (res *nats.Msg, err error) {
	if opts == nil || opts.nc == nil {
		return nil, fmt.Errorf("nats connection is not set")
	}

	var ctx context.Context
	var cancel func()

	if opts.ctx == nil {
		ctx, cancel = context.WithTimeout(context.Background(), opts.timeout)
		defer cancel()
	} else {
		ctx = opts.ctx
	}

	res, err = opts.nc.RequestWithContext(ctx, subj, data)
	if err != nil {
		return nil, err
	}

	return res, ParseErrorResponse(res)
}

// IsValidName verifies if n is a valid stream, template or consumer name
func IsValidName(n string) bool {
	if n == "" {
		return false
	}

	return !strings.ContainsAny(n, ">*. ")
}
