// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type fakeRequest struct {
	items int
}

func (r fakeRequest) ItemsCount() int {
	return r.items
}

type fakeRequestConverter struct {
	metricsError error
	tracesError  error
	logsError    error
}

func (c fakeRequestConverter) RequestFromMetrics(_ context.Context, md pmetric.Metrics) (Request, error) {
	return fakeRequest{items: md.DataPointCount()}, c.metricsError
}

func (c fakeRequestConverter) RequestFromTraces(_ context.Context, td ptrace.Traces) (Request, error) {
	return fakeRequest{items: td.SpanCount()}, c.tracesError
}

func (c fakeRequestConverter) RequestFromLogs(_ context.Context, ld plog.Logs) (Request, error) {
	return fakeRequest{items: ld.LogRecordCount()}, c.logsError
}

func newFakeRequestSender(err error) RequestSender {
	return func(_ context.Context, _ Request) error {
		return err
	}
}
