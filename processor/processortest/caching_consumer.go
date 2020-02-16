// Copyright 2020, OpenTelemetry Authors
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

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
)

// CachingTraceConsumer is a dummy consumer that simply stores the trace data.
// When testing processors, it can be specified as the next consumer of a processor
// to inspect the outputs of the processor being tested.
type CachingTraceConsumer struct {
	Data consumerdata.TraceData
}

// ConsumeTraceData stores the trace data and returns nil.
func (ctc *CachingTraceConsumer) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	ctc.Data = td
	return nil
}

// CachingMetricsConsumer is a dummy consumer that simply stores the metrics data.
// When testing processors, it can be specified as the next consumer of a processor
// to inspect the outputs of the processor being tested.
type CachingMetricsConsumer struct {
	Data consumerdata.MetricsData
}

// ConsumeMetricData stores the metrics data and returns nil.
func (cmc *CachingMetricsConsumer) ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error {
	cmc.Data = md
	return nil
}
