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

package consumerhelper // import "go.opentelemetry.io/collector/consumer/consumerhelper"

import (
	"errors"

	"go.opentelemetry.io/collector/consumer"
)

var errNilFunc = errors.New("nil consumer func")

type baseConsumer struct {
	capabilities consumer.Capabilities
}

// Option applies changes to internalOptions.
type Option func(*baseConsumer)

// WithCapabilities overrides the default GetCapabilities function for a processor.
// The default GetCapabilities function returns mutable capabilities.
func WithCapabilities(capabilities consumer.Capabilities) Option {
	return func(o *baseConsumer) {
		o.capabilities = capabilities
	}
}

// Capabilities implementation of the base Consumer.
func (bs baseConsumer) Capabilities() consumer.Capabilities {
	return bs.capabilities
}

func newBaseConsumer(options ...Option) *baseConsumer {
	bs := &baseConsumer{
		capabilities: consumer.Capabilities{MutatesData: false},
	}

	for _, op := range options {
		op(bs)
	}

	return bs
}
