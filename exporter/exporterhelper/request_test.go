// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"context"
	"sync/atomic"

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
		requestsCount: &atomic.Int64{},
		itemsCount:    &atomic.Int64{},
	}
}

type fakeRequest struct {
	items     int
	exportErr error
	mergeErr  error
	batchID   string
	sink      *fakeRequestSink
}

func (r *fakeRequest) Export(_ context.Context) error {
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

func fakeBatchMergeFunc(_ context.Context, r1 Request, r2 Request) (Request, error) {
	if r1 == nil {
		return r2, nil
	}
	fr1 := r1.(*fakeRequest)
	fr2 := r2.(*fakeRequest)
	if fr2.mergeErr != nil {
		return nil, fr2.mergeErr
	}
	return &fakeRequest{items: fr1.items + fr2.items, sink: fr1.sink}, nil
}

func fakeBatchMergeOrSplitFunc(ctx context.Context, r1 Request, r2 Request, maxItems int) ([]Request, error) {
	if maxItems == 0 {
		r, err := fakeBatchMergeFunc(ctx, r1, r2)
		return []Request{r}, err
	}

	if r2.(*fakeRequest).mergeErr != nil {
		return nil, r2.(*fakeRequest).mergeErr
	}

	fr2 := &fakeRequest{items: r2.(*fakeRequest).items, sink: r2.(*fakeRequest).sink}
	res := []Request{}

	// fill fr1 to maxItems if it's not nil
	if r1 != nil {
		fr1 := &fakeRequest{items: r1.(*fakeRequest).items, sink: r1.(*fakeRequest).sink}
		if fr2.items <= maxItems-fr1.items {
			fr1.items += fr2.items
			return res, nil
		}
		fr2.items -= maxItems - fr1.items
		fr1.items = maxItems
		res = append(res, fr1)
	}

	// split fr2 to maxItems
	for {
		if fr2.items <= maxItems {
			res = append(res, &fakeRequest{items: fr2.items, sink: fr2.sink})
			break
		}
		res = append(res, &fakeRequest{items: maxItems, sink: fr2.sink})
		fr2.items -= maxItems
	}

	return res, nil
}

type fakeRequestConverter struct {
	metricsError error
	tracesError  error
	logsError    error
	requestError error
}

func (c fakeRequestConverter) RequestFromMetrics(_ context.Context, md pmetric.Metrics) (Request, error) {
	return &fakeRequest{items: md.DataPointCount(), exportErr: c.requestError}, c.metricsError
}

func (c fakeRequestConverter) RequestFromTraces(_ context.Context, td ptrace.Traces) (Request, error) {
	return &fakeRequest{items: td.SpanCount(), exportErr: c.requestError}, c.tracesError
}

func (c fakeRequestConverter) RequestFromLogs(_ context.Context, ld plog.Logs) (Request, error) {
	return &fakeRequest{items: ld.LogRecordCount(), exportErr: c.requestError}, c.logsError
}
