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

package receiverhelper

import (
	"context"

	"go.opentelemetry.io/collector/component"
)

// Start specifies the function invoked when the receiver is being started.
type Start func(context.Context, component.Host) error

// Shutdown specifies the function invoked when the receiver is being shutdown.
type Shutdown func(context.Context) error

// Option apply changes to internal options.
type Option func(*baseReceiver)

// WithStart overrides the default Start function for a receiver.
// The default shutdown function does nothing and always returns nil.
func WithStart(start Start) Option {
	return func(o *baseReceiver) {
		o.start = start
	}
}

// WithShutdown overrides the default Shutdown function for a receiver.
// The default shutdown function does nothing and always returns nil.
func WithShutdown(shutdown Shutdown) Option {
	return func(o *baseReceiver) {
		o.shutdown = shutdown
	}
}

type baseReceiver struct {
	fullName string
	start    Start
	shutdown Shutdown
}

// Construct the internalOptions from multiple Option.
func newBaseReceiver(fullName string, options ...Option) baseReceiver {
	br := baseReceiver{fullName: fullName}

	for _, op := range options {
		op(&br)
	}

	return br
}

// Start the receiver, invoked during service start.
func (br *baseReceiver) Start(ctx context.Context, host component.Host) error {
	if br.start != nil {
		return br.start(ctx, host)
	}
	return nil
}

// Shutdown the receiver, invoked during service shutdown.
func (br *baseReceiver) Shutdown(ctx context.Context) error {
	if br.shutdown != nil {
		return br.shutdown(ctx)
	}
	return nil
}
