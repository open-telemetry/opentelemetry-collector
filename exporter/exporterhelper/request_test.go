// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"context"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type fakeRequest struct {
	items int
	err   error
}

func (r fakeRequest) Export(_ context.Context) error {
	return r.err
}

func (r fakeRequest) ItemsCount() int {
	return r.items
}

type fakeRequestConverter struct {
	metricsError error
	tracesError  error
	logsError    error
	requestError error
}

func (c fakeRequestConverter) RequestFromMetrics(_ context.Context, md pmetric.Metrics) (Request, error) {
	return fakeRequest{items: md.DataPointCount(), err: c.requestError}, c.metricsError
}

func (c fakeRequestConverter) RequestFromTraces(_ context.Context, td ptrace.Traces) (Request, error) {
	return fakeRequest{items: td.SpanCount(), err: c.requestError}, c.tracesError
}

func (c fakeRequestConverter) RequestFromLogs(_ context.Context, ld plog.Logs) (Request, error) {
	return fakeRequest{items: ld.LogRecordCount(), err: c.requestError}, c.logsError
}
