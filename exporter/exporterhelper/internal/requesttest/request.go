// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package requesttest // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type errorPartial struct {
	fr *FakeRequest
}

func (e errorPartial) Error() string {
	return fmt.Sprintf("items: %d", e.fr.Items)
}

type FakeRequest struct {
	Items          int
	Bytes          int
	Partial        int
	MergeErr       error
	MergeErrResult []request.Request
	Delay          time.Duration
}

func (r *FakeRequest) OnError(err error) request.Request {
	var pErr errorPartial
	if errors.As(err, &pErr) {
		return pErr.fr
	}
	return r
}

func (r *FakeRequest) ItemsCount() int {
	return r.Items
}

func (r *FakeRequest) BytesSize() int {
	return r.Bytes
}

func (r *FakeRequest) MergeSplit(_ context.Context, maxSize int, szt request.SizerType, r2 request.Request) ([]request.Request, error) {
	if r.MergeErr != nil {
		return r.MergeErrResult, r.MergeErr
	}

	if r2 != nil {
		fr2 := r2.(*FakeRequest)
		if fr2.MergeErr != nil {
			return fr2.MergeErrResult, fr2.MergeErr
		}
		fr2.mergeTo(r)
	}

	if maxSize == 0 {
		return []request.Request{r}, nil
	}

	var res []request.Request
	switch szt {
	case request.SizerTypeItems:
		for r.Items != 0 {
			if r.Items <= maxSize {
				res = append(res, r)
				break
			}
			res = append(res, &FakeRequest{Items: maxSize, Bytes: -1, Delay: r.Delay})
			r.Items -= maxSize
			r.Bytes = -1
		}
	case request.SizerTypeBytes:
		for r.Bytes != 0 {
			if r.Bytes <= maxSize {
				res = append(res, r)
				break
			}
			res = append(res, &FakeRequest{Items: -1, Bytes: maxSize, Delay: r.Delay})
			r.Items = -1
			r.Bytes -= maxSize
		}
	}

	return res, nil
}

func (r *FakeRequest) mergeTo(dst *FakeRequest) {
	dst.Items += r.Items
	dst.Bytes += r.Bytes
	dst.Delay += r.Delay
}

func RequestFromMetricsFunc(err error) func(context.Context, pmetric.Metrics) (request.Request, error) {
	return func(_ context.Context, md pmetric.Metrics) (request.Request, error) {
		return &FakeRequest{Items: md.DataPointCount()}, err
	}
}

func RequestFromTracesFunc(err error) func(context.Context, ptrace.Traces) (request.Request, error) {
	return func(_ context.Context, td ptrace.Traces) (request.Request, error) {
		return &FakeRequest{Items: td.SpanCount()}, err
	}
}

func RequestFromLogsFunc(err error) func(context.Context, plog.Logs) (request.Request, error) {
	return func(_ context.Context, ld plog.Logs) (request.Request, error) {
		return &FakeRequest{Items: ld.LogRecordCount()}, err
	}
}
