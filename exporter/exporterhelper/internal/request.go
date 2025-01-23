// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"

	"go.opentelemetry.io/collector/exporter/internal"
	"go.opentelemetry.io/collector/exporter/internal/requesttest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func RequestFromMetricsFunc(reqErr error) func(context.Context, pmetric.Metrics) (internal.Request, error) {
	return func(_ context.Context, md pmetric.Metrics) (internal.Request, error) {
		return &requesttest.FakeRequest{Items: md.DataPointCount(), ExportErr: reqErr}, nil
	}
}

func RequestFromTracesFunc(reqErr error) func(context.Context, ptrace.Traces) (internal.Request, error) {
	return func(_ context.Context, td ptrace.Traces) (internal.Request, error) {
		return &requesttest.FakeRequest{Items: td.SpanCount(), ExportErr: reqErr}, nil
	}
}

func RequestFromLogsFunc(reqErr error) func(context.Context, plog.Logs) (internal.Request, error) {
	return func(_ context.Context, ld plog.Logs) (internal.Request, error) {
		return &requesttest.FakeRequest{Items: ld.LogRecordCount(), ExportErr: reqErr}, nil
	}
}
