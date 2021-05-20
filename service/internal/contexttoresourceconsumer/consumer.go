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

// Package contexttoresourceconsumer contains implementations of Traces/Metrics/Logs consumers
// placing important information stored from the context.Context into the each data point's
// pdata.Resource attributes.
package contexttoresourceconsumer

import (
	"context"
	"strings"

	"go.opentelemetry.io/collector/auth"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerhelper"
	"go.opentelemetry.io/collector/consumer/pdata"
)

type wrappedConsumer struct {
	traces  consumer.Traces
	metrics consumer.Metrics
	logs    consumer.Logs
}

func NewTraces(toConsume consumer.Traces) (consumer.Traces, error) {
	tc := &wrappedConsumer{
		traces: toConsume,
	}
	return consumerhelper.NewTraces(
		tc.consumeTraces,
		consumerhelper.WithCapabilities(consumer.Capabilities{MutatesData: true}),
	)
}

func NewMetrics(toConsume consumer.Metrics) (consumer.Metrics, error) {
	tc := &wrappedConsumer{
		metrics: toConsume,
	}
	return consumerhelper.NewMetrics(
		tc.consumeMetrics,
		consumerhelper.WithCapabilities(consumer.Capabilities{MutatesData: true}),
	)
}

func NewLogs(toConsume consumer.Logs) (consumer.Logs, error) {
	tc := &wrappedConsumer{
		logs: toConsume,
	}
	return consumerhelper.NewLogs(
		tc.consumeLogs,
		consumerhelper.WithCapabilities(consumer.Capabilities{MutatesData: true}),
	)
}

func (tc wrappedConsumer) consumeTraces(ctx context.Context, td pdata.Traces) error {
	raw, subject, groups := extractFromContext(ctx)

	for i := 0; i < td.ResourceSpans().Len(); i++ {
		rs := td.ResourceSpans().At(i).Resource()
		injectIntoResource(rs, raw, subject, groups)
	}

	return tc.traces.ConsumeTraces(ctx, td)
}

func (tc wrappedConsumer) consumeMetrics(ctx context.Context, md pdata.Metrics) error {
	raw, subject, groups := extractFromContext(ctx)

	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rs := md.ResourceMetrics().At(i).Resource()
		injectIntoResource(rs, raw, subject, groups)
	}

	return tc.metrics.ConsumeMetrics(ctx, md)
}

func (tc wrappedConsumer) consumeLogs(ctx context.Context, ld pdata.Logs) error {
	raw, subject, groups := extractFromContext(ctx)

	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		rs := ld.ResourceLogs().At(i).Resource()
		injectIntoResource(rs, raw, subject, groups)
	}

	return tc.logs.ConsumeLogs(ctx, ld)
}

func extractFromContext(ctx context.Context) (raw, subject, groups string) {
	raw, _ = auth.RawFromContext(ctx)
	subject, _ = auth.SubjectFromContext(ctx)
	groupsSlice, _ := auth.GroupsFromContext(ctx)
	groups = strings.Join(groupsSlice, ",")
	return
}

func injectIntoResource(rs pdata.Resource, raw, subject, groups string) {
	if len(raw) > 0 {
		rs.Attributes().UpsertString(auth.RawAttributeKey, raw)
	}

	if len(subject) > 0 {
		rs.Attributes().UpsertString(auth.SubjectAttributeKey, subject)
	}

	if len(groups) > 0 {
		rs.Attributes().UpsertString(auth.MembershipsAttributeKey, groups)
	}
}
