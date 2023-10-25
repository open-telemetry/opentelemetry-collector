// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"context"
	"encoding/json"
	"errors"

	"go.opentelemetry.io/collector/exporter/exporterhelper/request"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type fakeRequest struct {
	Items int
	Err   error
}

func (r fakeRequest) Export(_ context.Context) error {
	return r.Err
}

func (r fakeRequest) ItemsCount() int {
	return r.Items
}

type fakeRequestConverter struct {
	metricsError error
	tracesError  error
	logsError    error
	requestError error
}

func (c fakeRequestConverter) RequestFromMetrics(_ context.Context, md pmetric.Metrics) (request.Request, error) {
	return fakeRequest{Items: md.DataPointCount(), Err: c.requestError}, c.metricsError
}

func (c fakeRequestConverter) RequestFromTraces(_ context.Context, td ptrace.Traces) (request.Request, error) {
	return fakeRequest{Items: td.SpanCount(), Err: c.requestError}, c.tracesError
}

func (c fakeRequestConverter) RequestFromLogs(_ context.Context, ld plog.Logs) (request.Request, error) {
	return fakeRequest{Items: ld.LogRecordCount(), Err: c.requestError}, c.logsError
}

func fakeRequestMarshaler(req request.Request) ([]byte, error) {
	r, ok := req.(fakeRequest)
	if !ok {
		return nil, errors.New("invalid request type")
	}
	return json.Marshal(r)
}

func fakeRequestUnmarshaler(bytes []byte) (request.Request, error) {
	var r fakeRequest
	if err := json.Unmarshal(bytes, &r); err != nil {
		return nil, err
	}
	return r, nil
}
