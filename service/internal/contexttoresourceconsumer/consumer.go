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
	auth := extractFromContext(ctx)
	td = pdata.TracesWithAuth(td, auth)
	return tc.traces.ConsumeTraces(ctx, td)
}

func (tc wrappedConsumer) consumeMetrics(ctx context.Context, md pdata.Metrics) error {
	auth := extractFromContext(ctx)
	md = pdata.MetricsWithAuth(md, auth)
	return tc.metrics.ConsumeMetrics(ctx, md)
}

func (tc wrappedConsumer) consumeLogs(ctx context.Context, ld pdata.Logs) error {
	auth := extractFromContext(ctx)
	ld = pdata.LogsWithAuth(ld, auth)
	return tc.logs.ConsumeLogs(ctx, ld)
}

func extractFromContext(ctx context.Context) *pdata.Auth {
	raw, _ := auth.RawFromContext(ctx)
	subject, _ := auth.SubjectFromContext(ctx)
	groups, _ := auth.GroupsFromContext(ctx)

	// raw and subject are required, no need to check the groups here
	if len(raw) == 0 && len(subject) == 0 {
		return nil
	}

	return &pdata.Auth{
		Raw:     raw,
		Subject: subject,
		Groups:  groups,
	}
}
