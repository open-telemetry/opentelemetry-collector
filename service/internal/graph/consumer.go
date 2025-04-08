// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph // import "go.opentelemetry.io/collector/service/internal/graph"

import (
	"context"

	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// baseConsumer redeclared here since not public in consumer package. May consider to make that public.
type baseConsumer interface {
	Capabilities() consumer.Capabilities
}

type consumerNode interface {
	getConsumer() baseConsumer
}

type obsConsumerTraces struct {
	consumer.Traces
	itemCounter metric.Int64Counter
}

func (c obsConsumerTraces) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	c.itemCounter.Add(ctx, int64(td.ResourceSpans().Len()))
	return c.Traces.ConsumeTraces(ctx, td)
}

type obsConsumerMetrics struct {
	consumer.Metrics
	itemCounter metric.Int64Counter
}

func (c obsConsumerMetrics) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	c.itemCounter.Add(ctx, int64(md.ResourceMetrics().Len()))
	return c.Metrics.ConsumeMetrics(ctx, md)
}

type obsConsumerLogs struct {
	consumer.Logs
	itemCounter metric.Int64Counter
}

func (c obsConsumerLogs) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	c.itemCounter.Add(ctx, int64(ld.ResourceLogs().Len()))
	return c.Logs.ConsumeLogs(ctx, ld)
}

type obsConsumerProfiles struct {
	xconsumer.Profiles
	itemCounter metric.Int64Counter
}

func (c obsConsumerProfiles) ConsumeProfiles(ctx context.Context, pd pprofile.Profiles) error {
	c.itemCounter.Add(ctx, int64(pd.ResourceProfiles().Len()))
	return c.Profiles.ConsumeProfiles(ctx, pd)
}
