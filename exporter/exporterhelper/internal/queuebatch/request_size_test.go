// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/testdata"
)

// These tests check how logs, metrics, and traces requests use request.SizeCache:
// that a cached size always equals a from-scratch computation over the request's
// current content, across merges and splits in both sizer dimensions.

// sizeCacheSignal describes one telemetry type for the shared size-cache tests.
type sizeCacheSignal struct {
	name string
	// newReq builds a request holding a generated payload of the given size.
	newReq func(count int) request.Request
	// freshSizes returns the from-scratch item count and byte size for a
	// request's current content, bypassing whatever the request has cached.
	freshSizes func(request.Request) (items, bytes int)
	// bytesOf returns the serialized size of a generated payload, used to pick
	// a split threshold for the bytes dimension.
	bytesOf func(count int) int
	// appendPayload adds more data to a request behind its back, without going
	// through merge/split, so a cached size becomes observably stale.
	appendPayload func(r request.Request, count int)
}

func sizeCacheSignals() []sizeCacheSignal {
	return []sizeCacheSignal{
		{
			name:   "logs",
			newReq: func(count int) request.Request { return newLogsRequest(testdata.GenerateLogs(count)) },
			freshSizes: func(r request.Request) (int, int) {
				dst := plog.NewLogs()
				r.(*logsRequest).ld.CopyTo(dst)
				clone := newLogsRequest(dst)
				return clone.ItemsCount(), clone.BytesSize()
			},
			bytesOf: func(count int) int { return logsMarshaler.LogsSize(testdata.GenerateLogs(count)) },
			appendPayload: func(r request.Request, count int) {
				testdata.GenerateLogs(count).ResourceLogs().MoveAndAppendTo(r.(*logsRequest).ld.ResourceLogs())
			},
		},
		{
			name:   "metrics",
			newReq: func(count int) request.Request { return newMetricsRequest(testdata.GenerateMetrics(count)) },
			freshSizes: func(r request.Request) (int, int) {
				dst := pmetric.NewMetrics()
				r.(*metricsRequest).md.CopyTo(dst)
				clone := newMetricsRequest(dst)
				return clone.ItemsCount(), clone.BytesSize()
			},
			bytesOf: func(count int) int { return metricsMarshaler.MetricsSize(testdata.GenerateMetrics(count)) },
			appendPayload: func(r request.Request, count int) {
				testdata.GenerateMetrics(count).ResourceMetrics().MoveAndAppendTo(r.(*metricsRequest).md.ResourceMetrics())
			},
		},
		{
			name:   "traces",
			newReq: func(count int) request.Request { return newTracesRequest(testdata.GenerateTraces(count)) },
			freshSizes: func(r request.Request) (int, int) {
				dst := ptrace.NewTraces()
				r.(*tracesRequest).td.CopyTo(dst)
				clone := newTracesRequest(dst)
				return clone.ItemsCount(), clone.BytesSize()
			},
			bytesOf: func(count int) int { return tracesMarshaler.TracesSize(testdata.GenerateTraces(count)) },
			appendPayload: func(r request.Request, count int) {
				testdata.GenerateTraces(count).ResourceSpans().MoveAndAppendTo(r.(*tracesRequest).td.ResourceSpans())
			},
		},
	}
}

// TestRequestSizesAreCached asserts that the size accessors serve the cache
// instead of recomputing. It appends to the request's payload out of band, which
// merge/split never do, so only a recomputation would observe the new data: a
// size that still reports the pre-append value proves it came from the cache.
func TestRequestSizesAreCached(t *testing.T) {
	for _, sig := range sizeCacheSignals() {
		t.Run(sig.name, func(t *testing.T) {
			req := sig.newReq(5)
			items, bytes := req.ItemsCount(), req.BytesSize()

			sig.appendPayload(req, 3)

			// The append must be big enough to change both dimensions, otherwise
			// this test would pass against a cache that never caches.
			grownItems, grownBytes := sig.freshSizes(req)
			require.Greater(t, grownItems, items)
			require.Greater(t, grownBytes, bytes)

			assert.Equal(t, items, req.ItemsCount(), "ItemsCount must serve the cached value")
			assert.Equal(t, bytes, req.BytesSize(), "BytesSize must serve the cached value")
		})
	}
}

// TestRequestSizeCacheCrossDimension asserts that after a merge that maintains
// only one size dimension, reading the OTHER dimension reflects the merged
// content rather than a value cached before the merge. A cache that fails to
// invalidate the dimension the merge did not maintain returns a stale size here.
func TestRequestSizeCacheCrossDimension(t *testing.T) {
	for _, sig := range sizeCacheSignals() {
		t.Run(sig.name+"/merge_by_items_then_read_bytes", func(t *testing.T) {
			batch := sig.newReq(5)
			// Populate BOTH caches so a missing invalidation would surface as staleness.
			_ = batch.ItemsCount()
			_ = batch.BytesSize()

			res, err := batch.MergeSplit(context.Background(), 0, request.SizerTypeItems, sig.newReq(7))
			require.NoError(t, err)
			merged := res[len(res)-1]

			wantItems, wantBytes := sig.freshSizes(merged)
			assert.Equal(t, wantItems, merged.ItemsCount(), "items after items-merge")
			assert.Equal(t, wantBytes, merged.BytesSize(), "bytes must be recomputed, not stale")
		})

		t.Run(sig.name+"/merge_by_bytes_then_read_items", func(t *testing.T) {
			batch := sig.newReq(5)
			_ = batch.ItemsCount()
			_ = batch.BytesSize()

			res, err := batch.MergeSplit(context.Background(), 0, request.SizerTypeBytes, sig.newReq(7))
			require.NoError(t, err)
			merged := res[len(res)-1]

			wantItems, wantBytes := sig.freshSizes(merged)
			// A bytes-dimension merge does not maintain the item count, so
			// ItemsCount must recompute instead of returning the pre-merge value.
			assert.Equal(t, wantItems, merged.ItemsCount(), "items must be recomputed, not stale")
			assert.Equal(t, wantBytes, merged.BytesSize(), "bytes after bytes-merge")
		})
	}
}

// TestRequestCachedSizeMatchesRecompute exercises long merge/split sequences
// for both sizer dimensions and asserts that every resulting request's cached
// ItemsCount()/BytesSize() equals a from-scratch recomputation over identical
// content. This covers the incremental cache maintenance in both mergeTo and
// split (including the split boundary that produces multiple requests).
func TestRequestCachedSizeMatchesRecompute(t *testing.T) {
	for _, sig := range sizeCacheSignals() {
		cases := []struct {
			name    string
			szt     request.SizerType
			maxSize int
		}{
			{"items_no_split", request.SizerTypeItems, 0},
			{"items_forced_splits", request.SizerTypeItems, 20},
			{"bytes_no_split", request.SizerTypeBytes, 0},
			{"bytes_forced_splits", request.SizerTypeBytes, sig.bytesOf(20)},
		}

		for _, tc := range cases {
			t.Run(sig.name+"/"+tc.name, func(t *testing.T) {
				batch := sig.newReq(3)
				for i := range 50 {
					res, err := batch.MergeSplit(context.Background(), tc.maxSize, tc.szt, sig.newReq(11))
					require.NoError(t, err)
					require.NotEmpty(t, res)
					for _, r := range res {
						wantItems, wantBytes := sig.freshSizes(r)
						assert.Equalf(t, wantItems, r.ItemsCount(), "iter %d: ItemsCount", i)
						assert.Equalf(t, wantBytes, r.BytesSize(), "iter %d: BytesSize", i)
					}
					batch = res[len(res)-1]
				}
			})
		}
	}
}
