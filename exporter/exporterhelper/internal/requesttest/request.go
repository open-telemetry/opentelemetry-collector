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

func (r *FakeRequest) MergeSplit(_ context.Context, maxSizePerSizer map[request.SizerType]int64, r2 request.Request) ([]request.Request, error) {
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

	if r.Items == 0 {
		return []request.Request{r}, nil
	}

	// Calculate how many items to take per batch.
	itemsToTake := r.Items
	hasLimits := false

	for szt, limit := range maxSizePerSizer {
		if limit <= 0 {
			continue
		}
		hasLimits = true
		switch szt {
		case request.SizerTypeItems:
			if int(limit) < itemsToTake {
				itemsToTake = int(limit)
			}
		case request.SizerTypeBytes:
			if r.Bytes > 0 {
				// Assume uniform distribution of bytes across items.
				avgBytesPerItem := float64(r.Bytes) / float64(r.Items)
				maxItemsByBytes := int(float64(limit) / avgBytesPerItem)
				if maxItemsByBytes < itemsToTake {
					itemsToTake = maxItemsByBytes
				}
			}
		}
	}

	if !hasLimits {
		return []request.Request{r}, nil
	}

	if itemsToTake <= 0 {
		itemsToTake = 1 // Guarantee progress
	}

	var res []request.Request
	avgBytesPerItem := float64(r.Bytes) / float64(r.Items)

	for r.Items > 0 {
		take := itemsToTake
		if take > r.Items {
			take = r.Items
		}

		bytes := int(float64(take) * avgBytesPerItem)

		res = append(res, &FakeRequest{
			Items: take,
			Bytes: bytes,
			Delay: r.Delay,
		})

		r.Items -= take
		r.Bytes -= bytes
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
