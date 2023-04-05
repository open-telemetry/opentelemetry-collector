// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package interceptor // import "go.opentelemetry.io/collector/extension/interceptor"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

// Error instructs the interceptor caller that the processing cannot continue,
// perhaps because the request was rejected by the interceptor. The interceptor
// MAY specify a status code and message to be relayed to the peer.
type Error struct {
	error
	StatusCode int
	Message    string
}

func NewWrappedError(wrapped error, statusCode int, message string) *Error {
	return &Error{
		error:      wrapped,
		StatusCode: statusCode,
		Message:    message,
	}
}

func (e *Error) Unwrap() error {
	return e.error
}

// Interceptor establishes the contract between the port, such as an HTTP or gRPC server,
// and the domain (interceptor implementation). The port MUST provide the interceptor with
// the connection's metadata, such as HTTP headers or gRPC's metadata. The interceptor
// MUST wrap the original error, if any, and MAY specify a status code and/or message to
// be sent to the connection's peer. The interceptor MUST NOT change the incoming metadata.
// Ports MUST abort the processing when an error is returned, as it indicates that the
// interceptor rejected the request.
type Interceptor interface {
	extension.Extension
	Intercept(map[string][]string) error
}

type defaultInterceptor struct {
	InterceptFunc
	component.StartFunc
	component.ShutdownFunc
}

// New creates a new interceptor with the given options.
func New(options ...Option) Interceptor {
	bc := &defaultInterceptor{}

	for _, op := range options {
		op(bc)
	}

	return bc
}

// Option specifies a customization possibility for the interceptor.
type Option func(*defaultInterceptor)

// InterceptFunc defines the signature for the function responsible for
// intercepting the request.
type InterceptFunc func(map[string][]string) error

func (f InterceptFunc) Intercept(md map[string][]string) error {
	if f == nil {
		return nil
	}
	return f(md)
}

// WithIntercept specifies which function to use to perform the interception.
func WithIntercept(interceptFunc InterceptFunc) Option {
	return func(o *defaultInterceptor) {
		o.InterceptFunc = interceptFunc
	}
}
