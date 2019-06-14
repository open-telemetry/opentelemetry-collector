// Copyright 2019, OpenCensus Authors
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

package processor

import (
	"github.com/census-instrumentation/opencensus-service/consumer"
)

// TraceProcessor composes TraceConsumer with some additional processor-specific functions.
type TraceProcessor interface {
	consumer.TraceConsumer

	// TODO: Add processor specific functions.
}

// MetricsProcessor composes MetricsConsumer with some additional processor-specific functions.
type MetricsProcessor interface {
	consumer.MetricsConsumer

	// TODO: Add processor specific functions.
}

// Processor is a data consumer.
type Processor interface {
	consumer.DataConsumer
}
