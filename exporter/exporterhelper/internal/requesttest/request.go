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
	Items    int
	Partial  int
	MergeErr error
	Delay    time.Duration
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

func (r *FakeRequest) MergeSplit(_ context.Context, maxSize int, _ request.SizerType, r2 request.Request) ([]request.Request, error) {
	if r.MergeErr != nil {
		return nil, r.MergeErr
	}

	maxItems := maxSize
	if maxItems == 0 {
		if r2 != nil {
			fr2 := r2.(*FakeRequest)
			if fr2.MergeErr != nil {
				return nil, fr2.MergeErr
			}
			fr2.mergeTo(r)
		}
		return []request.Request{r}, nil
	}

	var fr2 *FakeRequest
	if r2 == nil {
		fr2 = &FakeRequest{Delay: r.Delay}
	} else {
		if r2.(*FakeRequest).MergeErr != nil {
			return nil, r2.(*FakeRequest).MergeErr
		}
		fr2 = r2.(*FakeRequest)
	}

	var res []request.Request
	// No split, then just simple merge
	if r.Items+fr2.Items <= maxItems {
		fr2.mergeTo(r)
		return []request.Request{r}, nil
	}

	// if split is needed, we don't propagate ExportErr from fr2 to fr1 to test more cases
	fr2.Items -= maxItems - r.Items
	r.Items = maxItems
	res = append(res, r)

	// split fr2 to maxItems
	for {
		if fr2.Items <= maxItems {
			res = append(res, &FakeRequest{Items: fr2.Items, Delay: fr2.Delay})
			break
		}
		res = append(res, &FakeRequest{Items: maxItems, Delay: fr2.Delay})
		fr2.Items -= maxItems
	}

	return res, nil
}

func (r *FakeRequest) mergeTo(dst *FakeRequest) {
	dst.Items += r.Items
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
