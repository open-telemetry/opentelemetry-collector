// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package requesttest // import "go.opentelemetry.io/collector/exporter/internal/requesttest"

import (
	"context"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/internal"
)

type Sink struct {
	requestsCount *atomic.Int64
	itemsCount    *atomic.Int64
}

func (s *Sink) RequestsCount() int64 {
	return s.requestsCount.Load()
}

func (s *Sink) ItemsCount() int64 {
	return s.itemsCount.Load()
}

func NewSink() *Sink {
	return &Sink{
		requestsCount: new(atomic.Int64),
		itemsCount:    new(atomic.Int64),
	}
}

type FakeRequest struct {
	Items     int
	Sink      *Sink
	ExportErr error
	MergeErr  error
	Delay     time.Duration
}

func (r *FakeRequest) Export(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(r.Delay):
	}
	if r.ExportErr != nil {
		return r.ExportErr
	}
	if r.Sink != nil {
		r.Sink.requestsCount.Add(1)
		r.Sink.itemsCount.Add(int64(r.Items))
	}
	return nil
}

func (r *FakeRequest) ItemsCount() int {
	return r.Items
}

func (r *FakeRequest) MergeSplit(_ context.Context, cfg exporterbatcher.MaxSizeConfig, r2 internal.Request) ([]internal.Request, error) {
	if r.MergeErr != nil {
		return nil, r.MergeErr
	}

	maxItems := cfg.MaxSizeItems
	if maxItems == 0 {
		fr2 := r2.(*FakeRequest)
		if fr2.MergeErr != nil {
			return nil, fr2.MergeErr
		}
		return []internal.Request{
			&FakeRequest{
				Items:     r.Items + fr2.Items,
				Sink:      r.Sink,
				ExportErr: fr2.ExportErr,
				Delay:     r.Delay + fr2.Delay,
			},
		}, nil
	}

	var fr2 *FakeRequest
	if r2 == nil {
		fr2 = &FakeRequest{Sink: r.Sink, ExportErr: r.ExportErr, Delay: r.Delay}
	} else {
		if r2.(*FakeRequest).MergeErr != nil {
			return nil, r2.(*FakeRequest).MergeErr
		}
		fr2 = r2.(*FakeRequest)
		fr2 = &FakeRequest{Items: fr2.Items, Sink: fr2.Sink, ExportErr: fr2.ExportErr, Delay: fr2.Delay}
	}
	var res []internal.Request

	// fill fr1 to maxItems if it's not nil

	r = &FakeRequest{Items: r.Items, Sink: r.Sink, ExportErr: r.ExportErr, Delay: r.Delay}
	if fr2.Items <= maxItems-r.Items {
		r.Items += fr2.Items
		if fr2.ExportErr != nil {
			r.ExportErr = fr2.ExportErr
		}
		return []internal.Request{r}, nil
	}
	// if split is needed, we don't propagate ExportErr from fr2 to fr1 to test more cases
	fr2.Items -= maxItems - r.Items
	r.Items = maxItems
	res = append(res, r)

	// split fr2 to maxItems
	for {
		if fr2.Items <= maxItems {
			res = append(res, &FakeRequest{Items: fr2.Items, Sink: fr2.Sink, ExportErr: fr2.ExportErr, Delay: fr2.Delay})
			break
		}
		res = append(res, &FakeRequest{Items: maxItems, Sink: fr2.Sink, ExportErr: fr2.ExportErr, Delay: fr2.Delay})
		fr2.Items -= maxItems
	}

	return res, nil
}
