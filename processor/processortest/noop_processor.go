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

package processortest

import (
	"context"

	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/processor"
)

type noopProcessor int

var _ processor.TraceDataProcessor = (*noopProcessor)(nil)
var _ processor.MetricsDataProcessor = (*noopProcessor)(nil)

func (ns *noopProcessor) ProcessTraceData(ctx context.Context, td data.TraceData) error {
	return nil
}

func (ns *noopProcessor) ProcessMetricsData(ctx context.Context, md data.MetricsData) error {
	return nil
}

// NewNoopTraceDataProcessor creates an TraceDataProcessor that just drops the received data.
func NewNoopTraceDataProcessor() processor.TraceDataProcessor {
	return new(noopProcessor)
}

// NewNoopMetricsDataProcessor creates an MetricsDataProcessor that just drops the received data.
func NewNoopMetricsDataProcessor() processor.MetricsDataProcessor {
	return new(noopProcessor)
}
