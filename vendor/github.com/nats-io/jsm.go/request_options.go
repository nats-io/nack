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

package jsm

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// RequestOption is a option to configure the NATS related options
type RequestOption func(o *reqoptions)

type reqoptions struct {
	nc          *nats.Conn
	timeout     time.Duration
	ctx         context.Context
	trace       bool
	apiValidate bool
}

func dfltreqoptions() *reqoptions {
	return &reqoptions{
		nc:          Connection(),
		timeout:     timeout,
		trace:       shouldTrace(),
		apiValidate: shouldValidate(),
	}
}

func newreqoptions(opts ...RequestOption) (*reqoptions, error) {
	ropts := dfltreqoptions()

	for _, opt := range opts {
		opt(ropts)
	}

	if ropts.nc == nil {
		return nil, fmt.Errorf("no NATS connection supplied")
	}

	return ropts, nil
}

// WithAPIValidation validates responses sent from the NATS server against a schema
func WithAPIValidation() RequestOption {
	return func(o *reqoptions) {
		o.apiValidate = true
	}
}

// WithTrace enables logging of JSON API requests and responses
func WithTrace() RequestOption {
	return func(o *reqoptions) {
		o.trace = true
	}
}

// WithConnection sets the connection to use
func WithConnection(nc *nats.Conn) RequestOption {
	return func(o *reqoptions) {
		o.nc = nc
	}
}

// WithTimeout sets a timeout for the requests
func WithTimeout(t time.Duration) RequestOption {
	return func(o *reqoptions) {
		o.timeout = t
	}
}

// WithContext sets a context to use for a specific request
func WithContext(ctx context.Context) RequestOption {
	return func(o *reqoptions) {
		o.ctx = ctx
	}
}
