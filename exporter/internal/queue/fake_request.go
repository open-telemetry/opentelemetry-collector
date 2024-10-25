// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/internal/queue"

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/internal"
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

func (r *fakeRequest) Merge(_ context.Context,
	_ internal.Request) (internal.Request, error) {
	return nil, errors.New("not implemented")
}

func (r *fakeRequest) MergeSplit(_ context.Context, _ exporterbatcher.MaxSizeConfig,
	_ internal.Request) ([]internal.Request, error) {
	return nil, errors.New("not implemented")
}
