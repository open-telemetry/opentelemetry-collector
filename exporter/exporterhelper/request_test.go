// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"context"
	"encoding/json"
	"errors"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type fakeRequest struct {
	Items          int
	exportCallback func(req Request) error
}

func (r fakeRequest) Export(_ context.Context) error {
	if r.exportCallback == nil {
		return nil
	}
	return r.exportCallback(r)
}

func (r fakeRequest) ItemsCount() int {
	return r.Items
}

type fakeRequestConverter struct {
	metricsError   error
	tracesError    error
	logsError      error
	exportCallback func(req Request) error
}

func (c fakeRequestConverter) RequestFromMetrics(_ context.Context, md pmetric.Metrics) (Request, error) {
	return c.fakeRequest(md.DataPointCount()), c.metricsError
}

func (c fakeRequestConverter) RequestFromTraces(_ context.Context, td ptrace.Traces) (Request, error) {
	return c.fakeRequest(td.SpanCount()), c.tracesError
}

func (c fakeRequestConverter) RequestFromLogs(_ context.Context, ld plog.Logs) (Request, error) {
	return c.fakeRequest(ld.LogRecordCount()), c.logsError
}

func (c fakeRequestConverter) fakeRequest(items int) Request {
	return fakeRequest{Items: items, exportCallback: c.exportCallback}
}

func (c fakeRequestConverter) requestMarshalerFunc() RequestMarshaler {
	return func(req Request) ([]byte, error) {
		r, ok := req.(fakeRequest)
		if !ok {
			return nil, errors.New("invalid request type")
		}
		return json.Marshal(r)
	}
}

func (c fakeRequestConverter) requestUnmarshalerFunc() RequestUnmarshaler {
	return func(bytes []byte) (Request, error) {
		var r fakeRequest
		if err := json.Unmarshal(bytes, &r); err != nil {
			return nil, err
		}
		r.exportCallback = c.exportCallback
		return r, nil
	}
}
