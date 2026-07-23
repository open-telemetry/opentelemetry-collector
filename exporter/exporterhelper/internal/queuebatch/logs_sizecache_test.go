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
	"go.opentelemetry.io/collector/pdata/testdata"
)

// cloneLogs returns an independent deep copy, so a freshly-constructed request
// over it recomputes size from scratch (the ground truth for the cached value).
func cloneLogs(src plog.Logs) plog.Logs {
	dst := plog.NewLogs()
	src.CopyTo(dst)
	return dst
}

// freshSizes returns the from-scratch item count and byte size for a request's
// current content, bypassing whatever the request has cached.
func freshSizes(r request.Request) (items, bytes int) {
	clone := newLogsRequest(cloneLogs(r.(*logsRequest).ld))
	return clone.ItemsCount(), clone.BytesSize()
}

// TestLogsRequestSizeCacheCrossDimension guards the correctness concern from
// #12636: after a merge that maintains only one size dimension, reading the
// OTHER dimension must reflect the merged content, not a value cached before
// the merge. A cache that fails to invalidate the unused dimension (as in the
// earlier #12641 attempt) would return a stale size here.
func TestLogsRequestSizeCacheCrossDimension(t *testing.T) {
	t.Run("merge_by_items_then_read_bytes", func(t *testing.T) {
		batch := newLogsRequest(testdata.GenerateLogs(5))
		// Populate BOTH caches so a missing invalidation would surface as staleness.
		_ = batch.ItemsCount()
		_ = batch.BytesSize()

		res, err := batch.MergeSplit(context.Background(), 0, request.SizerTypeItems,
			newLogsRequest(testdata.GenerateLogs(7)))
		require.NoError(t, err)
		merged := res[len(res)-1]

		wantItems, wantBytes := freshSizes(merged)
		assert.Equal(t, wantItems, merged.ItemsCount(), "items after items-merge")
		assert.Equal(t, wantBytes, merged.BytesSize(), "bytes must be recomputed, not stale")
	})

	t.Run("merge_by_bytes_then_read_items", func(t *testing.T) {
		batch := newLogsRequest(testdata.GenerateLogs(5))
		_ = batch.ItemsCount()
		_ = batch.BytesSize()

		res, err := batch.MergeSplit(context.Background(), 0, request.SizerTypeBytes,
			newLogsRequest(testdata.GenerateLogs(7)))
		require.NoError(t, err)
		merged := res[len(res)-1]

		wantItems, wantBytes := freshSizes(merged)
		// This is the direction the earlier #12641 attempt got wrong: ItemsCount
		// stayed frozen at the pre-merge value after a bytes-dimension merge.
		assert.Equal(t, wantItems, merged.ItemsCount(), "items must be recomputed, not stale")
		assert.Equal(t, wantBytes, merged.BytesSize(), "bytes after bytes-merge")
	})
}

// TestLogsRequestCachedSizeMatchesRecompute exercises long merge/split sequences
// for both sizer dimensions and asserts that every resulting request's cached
// ItemsCount()/BytesSize() equals a from-scratch recomputation over identical
// content. This covers the incremental cache maintenance in both mergeTo and
// split (including the split boundary that produces multiple requests).
func TestLogsRequestCachedSizeMatchesRecompute(t *testing.T) {
	cases := []struct {
		name    string
		szt     request.SizerType
		maxSize int
	}{
		{"items_no_split", request.SizerTypeItems, 0},
		{"items_forced_splits", request.SizerTypeItems, 20},
		{"bytes_no_split", request.SizerTypeBytes, 0},
		{"bytes_forced_splits", request.SizerTypeBytes, logsMarshaler.LogsSize(testdata.GenerateLogs(20))},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			batch := newLogsRequest(testdata.GenerateLogs(3))
			for i := range 50 {
				res, err := batch.MergeSplit(context.Background(), tc.maxSize, tc.szt,
					newLogsRequest(testdata.GenerateLogs(11)))
				require.NoError(t, err)
				require.NotEmpty(t, res)
				for _, r := range res {
					wantItems, wantBytes := freshSizes(r)
					assert.Equalf(t, wantItems, r.ItemsCount(), "iter %d: ItemsCount", i)
					assert.Equalf(t, wantBytes, r.BytesSize(), "iter %d: BytesSize", i)
				}
				batch = res[len(res)-1]
			}
		})
	}
}
