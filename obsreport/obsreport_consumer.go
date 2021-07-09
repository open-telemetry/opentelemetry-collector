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

package obsreport

import (
	"context"

	"go.opencensus.io/tag"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
	"go.opentelemetry.io/collector/model/pdata"
)

// NewMetrics wraps a metrics consumer and adds pipeline tags.
func NewMetrics(pipeline string, mc consumer.Metrics) consumer.Metrics {
	return metricsConsumer{next: mc, pipeline: pipeline}
}

type metricsConsumer struct {
	next     consumer.Metrics
	pipeline string
}

var _ consumer.Metrics = (*metricsConsumer)(nil)

// ConsumeMetrics exports the pdata.Metrics to the next consumer after tagging.
func (c metricsConsumer) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	ctx, _ = tag.New(ctx,
		tag.Upsert(obsmetrics.TagKeyPipeline, c.pipeline, tag.WithTTL(tag.TTLNoPropagation)))
	return c.next.ConsumeMetrics(ctx, md)
}

func (c metricsConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// NewTraces wraps a traces consumer and adds pipeline tags.
func NewTraces(pipeline string, tc consumer.Traces) consumer.Traces {
	return tracesConsumer{next: tc, pipeline: pipeline}
}

type tracesConsumer struct {
	next     consumer.Traces
	pipeline string
}

var _ consumer.Traces = (*tracesConsumer)(nil)

// ConsumeTraces exports the pdata.Traces to the next consumer after tagging.
func (c tracesConsumer) ConsumeTraces(ctx context.Context, md pdata.Traces) error {
	ctx, _ = tag.New(ctx,
		tag.Upsert(obsmetrics.TagKeyPipeline, c.pipeline, tag.WithTTL(tag.TTLNoPropagation)))
	return c.next.ConsumeTraces(ctx, md)
}

func (c tracesConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// NewLogs wraps a logs consumer and adds pipeline tags.
func NewLogs(pipeline string, lc consumer.Logs) consumer.Logs {
	return logsConsumer{next: lc, pipeline: pipeline}
}

type logsConsumer struct {
	next     consumer.Logs
	pipeline string
}

var _ consumer.Logs = (*logsConsumer)(nil)

// ConsumeLogs exports the pdata.Logs to the next consumer after tagging.
func (c logsConsumer) ConsumeLogs(ctx context.Context, md pdata.Logs) error {
	ctx, _ = tag.New(ctx,
		tag.Upsert(obsmetrics.TagKeyPipeline, c.pipeline, tag.WithTTL(tag.TTLNoPropagation)))
	return c.next.ConsumeLogs(ctx, md)
}

func (c logsConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}
