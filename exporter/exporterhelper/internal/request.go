// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/internal"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type fakeRequestSink struct {
	requestsCount *atomic.Uint64
	itemsCount    *atomic.Uint64
}

func newFakeRequestSink() *fakeRequestSink {
	return &fakeRequestSink{
		requestsCount: &atomic.Uint64{},
		itemsCount:    &atomic.Uint64{},
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
		r.sink.itemsCount.Add(uint64(r.items))
	}
	return nil
}

func (r *fakeRequest) ItemsCount() int {
	return r.items
}

func fakeBatchMergeFunc(_ context.Context, r1 internal.Request, r2 internal.Request) (internal.Request, error) {
	if r1 == nil {
		return r2, nil
	}
	fr1 := r1.(*fakeRequest)
	fr2 := r2.(*fakeRequest)
	if fr2.mergeErr != nil {
		return nil, fr2.mergeErr
	}
	return &fakeRequest{
		items:     fr1.items + fr2.items,
		sink:      fr1.sink,
		exportErr: fr2.exportErr,
		delay:     fr1.delay + fr2.delay,
	}, nil
}

func fakeBatchMergeSplitFunc(ctx context.Context, cfg exporterbatcher.MaxSizeConfig, r1 internal.Request, r2 internal.Request) ([]internal.Request, error) {
	maxItems := cfg.MaxSizeItems
	if maxItems == 0 {
		r, err := fakeBatchMergeFunc(ctx, r1, r2)
		return []internal.Request{r}, err
	}

	if r2.(*fakeRequest).mergeErr != nil {
		return nil, r2.(*fakeRequest).mergeErr
	}

	fr2 := r2.(*fakeRequest)
	fr2 = &fakeRequest{items: fr2.items, sink: fr2.sink, exportErr: fr2.exportErr, delay: fr2.delay}
	var res []internal.Request

	// fill fr1 to maxItems if it's not nil
	if r1 != nil {
		fr1 := r1.(*fakeRequest)
		fr1 = &fakeRequest{items: fr1.items, sink: fr1.sink, exportErr: fr1.exportErr, delay: fr1.delay}
		if fr2.items <= maxItems-fr1.items {
			fr1.items += fr2.items
			if fr2.exportErr != nil {
				fr1.exportErr = fr2.exportErr
			}
			return []internal.Request{fr1}, nil
		}
		// if split is needed, we don't propagate exportErr from fr2 to fr1 to test more cases
		fr2.items -= maxItems - fr1.items
		fr1.items = maxItems
		res = append(res, fr1)
	}

	// split fr2 to maxItems
	for {
		if fr2.items <= maxItems {
			res = append(res, &fakeRequest{items: fr2.items, sink: fr2.sink, exportErr: fr2.exportErr, delay: fr2.delay})
			break
		}
		res = append(res, &fakeRequest{items: maxItems, sink: fr2.sink, exportErr: fr2.exportErr, delay: fr2.delay})
		fr2.items -= maxItems
	}

	return res, nil
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
