// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package processor contains interfaces that compose trace/metrics consumers.
package processor

import (
	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
)

// Processor defines the common functions that must be implemented by TraceProcessor
// and MetricsProcessor.
type Processor interface {
	component.Component

	// GetCapabilities must return the capabilities of the processor.
	GetCapabilities() Capabilities
}

// TraceProcessor composes TraceConsumer with some additional processor-specific functions.
type TraceProcessor interface {
	consumer.TraceConsumer
	Processor
}

// MetricsProcessor composes MetricsConsumer with some additional processor-specific functions.
type MetricsProcessor interface {
	consumer.MetricsConsumer
	Processor
}

type DualTypeProcessor interface {
	consumer.TraceConsumer
	consumer.MetricsConsumer
	Processor
}

// Capabilities describes the capabilities of TraceProcessor or MetricsProcessor.
type Capabilities struct {
	// MutatesConsumedData is set to true if ConsumeTraceData or ConsumeMetricsData
	// function of the processor modifies the input TraceData or MetricsData argument.
	// Processors which modify the input data MUST set this flag to true. If the processor
	// does not modify the data it MUST set this flag to false. If the processor creates
	// a copy of the data before modifying then this flag can be safely set to false.
	MutatesConsumedData bool
}
