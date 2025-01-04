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
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type fakeRequestSink struct {
	requestsCount *atomic.Int64
	itemsCount    *atomic.Int64
}

func newFakeRequestSink() *fakeRequestSink {
	return &fakeRequestSink{
		requestsCount: new(atomic.Int64),
		itemsCount:    new(atomic.Int64),
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
		r.sink.itemsCount.Add(int64(r.items))
	}
	return nil
}

func (r *fakeRequest) ItemsCount() int {
	return r.items
}

func (r *fakeRequest) MergeSplit(_ context.Context, cfg exporterbatcher.MaxSizeConfig, r2 internal.Request) ([]internal.Request, error) {
	if r.mergeErr != nil {
		return nil, r.mergeErr
	}

	maxItems := cfg.MaxSizeItems
	if maxItems == 0 {
		fr2 := r2.(*fakeRequest)
		if fr2.mergeErr != nil {
			return nil, fr2.mergeErr
		}
		return []internal.Request{
			&fakeRequest{
				items:     r.items + fr2.items,
				sink:      r.sink,
				exportErr: fr2.exportErr,
				delay:     r.delay + fr2.delay,
			},
		}, nil
	}

	var fr2 *fakeRequest
	if r2 == nil {
		fr2 = &fakeRequest{sink: r.sink, exportErr: r.exportErr, delay: r.delay}
	} else {
		if r2.(*fakeRequest).mergeErr != nil {
			return nil, r2.(*fakeRequest).mergeErr
		}
		fr2 = r2.(*fakeRequest)
		fr2 = &fakeRequest{items: fr2.items, sink: fr2.sink, exportErr: fr2.exportErr, delay: fr2.delay}
	}
	var res []internal.Request

	// fill fr1 to maxItems if it's not nil

	r = &fakeRequest{items: r.items, sink: r.sink, exportErr: r.exportErr, delay: r.delay}
	if fr2.items <= maxItems-r.items {
		r.items += fr2.items
		if fr2.exportErr != nil {
			r.exportErr = fr2.exportErr
		}
		return []internal.Request{r}, nil
	}
	// if split is needed, we don't propagate exportErr from fr2 to fr1 to test more cases
	fr2.items -= maxItems - r.items
	r.items = maxItems
	res = append(res, r)

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

func RequestFromMetricsFunc(reqErr error) func(context.Context, pmetric.Metrics) (internal.Request, error) {
	return func(_ context.Context, md pmetric.Metrics) (internal.Request, error) {
		return &fakeRequest{items: md.DataPointCount(), exportErr: reqErr}, nil
	}
}

func RequestFromTracesFunc(reqErr error) func(context.Context, ptrace.Traces) (internal.Request, error) {
	return func(_ context.Context, td ptrace.Traces) (internal.Request, error) {
		return &fakeRequest{items: td.SpanCount(), exportErr: reqErr}, nil
	}
}

func RequestFromLogsFunc(reqErr error) func(context.Context, plog.Logs) (internal.Request, error) {
	return func(_ context.Context, ld plog.Logs) (internal.Request, error) {
		return &fakeRequest{items: ld.LogRecordCount(), exportErr: reqErr}, nil
	}
}

func RequestFromProfilesFunc(reqErr error) func(context.Context, pprofile.Profiles) (internal.Request, error) {
	return func(_ context.Context, pd pprofile.Profiles) (internal.Request, error) {
		return &fakeRequest{items: pd.SampleCount(), exportErr: reqErr}, nil
	}
}
