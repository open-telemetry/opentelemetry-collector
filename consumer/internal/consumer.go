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

package internal // import "go.opentelemetry.io/collector/consumer/internal"

import (
	"errors"
)

// Capabilities describes the capabilities of a Processor.
type Capabilities struct {
	// MutatesData is set to true if Consume* function of the
	// processor modifies the input TraceData or MetricsData argument.
	// Processors which modify the input data MUST set this flag to true. If the processor
	// does not modify the data it MUST set this flag to false. If the processor creates
	// a copy of the data before modifying then this flag can be safely set to false.
	MutatesData bool
}

type BaseConsumer interface {
	Capabilities() Capabilities
}

var ErrNilFunc = errors.New("nil consumer func")

type BaseImpl struct {
	Cap Capabilities
}

// Option to construct new consumers.
type Option func(*BaseImpl)

// WithCapabilities overrides the default GetCapabilities function for a processor.
// The default GetCapabilities function returns mutable capabilities.
func WithCapabilities(capabilities Capabilities) Option {
	return func(o *BaseImpl) {
		o.Cap = capabilities
	}
}

// Capabilities implementation of the base
func (bs BaseImpl) Capabilities() Capabilities {
	return bs.Cap
}

func NewBaseImpl(options ...Option) *BaseImpl {
	bs := &BaseImpl{
		Cap: Capabilities{MutatesData: false},
	}

	for _, op := range options {
		op(bs)
	}

	return bs
}
