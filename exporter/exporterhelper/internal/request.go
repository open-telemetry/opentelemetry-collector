// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/collector/exporter/internal"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type fakeRequestSink struct {
	requestsCount *atomic.Int64
	itemsCount    *atomic.Int64
}

func newFakeRequestSink() *fakeRequestSink {
	return &fakeRequestSink{
		requestsCount: new(atomic.Int64),
		itemsCount:    new(atomic.Int64),
	}
}

type fakeRequest struct {
	items     int
	exportErr error
	mergeErr  error
	delay     time.Duration
	sink      *fakeRequestSink
}

func (r *fakeRequest) Export(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(r.delay):
	}
	if r.exportErr != nil {
		return r.exportErr
	}
	if r.sink != nil {
		r.sink.requestsCount.Add(1)
		r.sink.itemsCount.Add(int64(r.items))
	}
	return nil
}

func (r *fakeRequest) ItemsCount() int {
	return r.items
}

type FakeRequestConverter struct {
	MetricsError error
	TracesError  error
	LogsError    error
	RequestError error
}

func (frc *FakeRequestConverter) RequestFromMetricsFunc(_ context.Context, md pmetric.Metrics) (internal.Request, error) {
	return &fakeRequest{items: md.DataPointCount(), exportErr: frc.RequestError}, frc.MetricsError
}

func (frc *FakeRequestConverter) RequestFromTracesFunc(_ context.Context, md ptrace.Traces) (internal.Request, error) {
	return &fakeRequest{items: md.SpanCount(), exportErr: frc.RequestError}, frc.TracesError
}

func (frc *FakeRequestConverter) RequestFromLogsFunc(_ context.Context, md plog.Logs) (internal.Request, error) {
	return &fakeRequest{items: md.LogRecordCount(), exportErr: frc.RequestError}, frc.LogsError
}
