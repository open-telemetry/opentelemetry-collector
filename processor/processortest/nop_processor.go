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

package processortest

import (
	"context"

	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/processor"
)

type nopProcessor struct {
	nextTraceProcessor   consumer.TraceConsumer
	nextMetricsProcessor consumer.MetricsConsumer
}

var _ processor.TraceProcessor = (*nopProcessor)(nil)
var _ processor.MetricsProcessor = (*nopProcessor)(nil)

func (np *nopProcessor) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	return np.nextTraceProcessor.ConsumeTraceData(ctx, td)
}

func (np *nopProcessor) ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error {
	return np.nextMetricsProcessor.ConsumeMetricsData(ctx, md)
}

func (np *nopProcessor) GetCapabilities() processor.Capabilities {
	return processor.Capabilities{MutatesConsumedData: false}
}

// NewNopTraceProcessor creates an TraceProcessor that just pass the received data to the nextTraceProcessor.
func NewNopTraceProcessor(nextTraceProcessor consumer.TraceConsumer) consumer.TraceConsumer {
	return &nopProcessor{nextTraceProcessor: nextTraceProcessor}
}

// NewNopMetricsProcessor creates an MetricsProcessor that just pass the received data to the nextMetricsProcessor.
func NewNopMetricsProcessor(nextMetricsProcessor consumer.MetricsConsumer) consumer.MetricsConsumer {
	return &nopProcessor{nextMetricsProcessor: nextMetricsProcessor}
}
