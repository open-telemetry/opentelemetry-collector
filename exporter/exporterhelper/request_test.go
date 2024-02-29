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

func (r fakeRequest) Export(context.Context) error {
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

func (frc *fakeRequestConverter) requestFromMetricsFunc(_ context.Context, md pmetric.Metrics) (Request, error) {
	return fakeRequest{items: md.DataPointCount(), err: frc.requestError}, frc.metricsError
}

func (frc *fakeRequestConverter) requestFromTracesFunc(_ context.Context, md ptrace.Traces) (Request, error) {
	return fakeRequest{items: md.SpanCount(), err: frc.requestError}, frc.tracesError
}

func (frc *fakeRequestConverter) requestFromLogsFunc(_ context.Context, md plog.Logs) (Request, error) {
	return fakeRequest{items: md.LogRecordCount(), err: frc.requestError}, frc.logsError
}
