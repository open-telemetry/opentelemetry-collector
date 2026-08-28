// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xexporterhelper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/testdata"
)

// These tests check how profilesRequest uses request.SizeCache: that a cached
// size always equals a from-scratch computation over the request's current
// content, across merges and splits in both sizer dimensions.

// freshProfilesSizes returns the from-scratch item count and byte size for a
// request's current content, bypassing whatever the request has cached.
func freshProfilesSizes(r Request) (items, bytes int) {
	dst := pprofile.NewProfiles()
	r.(*profilesRequest).pd.CopyTo(dst)
	clone := newProfilesRequest(dst)
	return clone.ItemsCount(), clone.BytesSize()
}

var profilesSizerTypes = []exporterhelper.RequestSizerType{
	exporterhelper.RequestSizerTypeItems,
	exporterhelper.RequestSizerTypeBytes,
}

// TestProfilesRequestSizesAreCached asserts that the size accessors serve the
// cache instead of recomputing. It appends to the request's payload out of band,
// which merge/split never do, so only a recomputation would observe the new data.
func TestProfilesRequestSizesAreCached(t *testing.T) {
	req := newProfilesRequest(testdata.GenerateProfiles(5))
	items, bytes := req.ItemsCount(), req.BytesSize()

	testdata.GenerateProfiles(3).ResourceProfiles().MoveAndAppendTo(req.(*profilesRequest).pd.ResourceProfiles())

	// The append must be big enough to change both dimensions, otherwise this
	// test would pass against a cache that never caches.
	grownItems, grownBytes := freshProfilesSizes(req)
	require.Greater(t, grownItems, items)
	require.Greater(t, grownBytes, bytes)

	assert.Equal(t, items, req.ItemsCount(), "ItemsCount must serve the cached value")
	assert.Equal(t, bytes, req.BytesSize(), "BytesSize must serve the cached value")
}

// TestProfilesRequestSizeCacheCrossDimension asserts that after a merge that
// maintains only one size dimension, reading the OTHER dimension reflects the
// merged content rather than a value cached before the merge.
func TestProfilesRequestSizeCacheCrossDimension(t *testing.T) {
	for _, szt := range profilesSizerTypes {
		t.Run(szt.String(), func(t *testing.T) {
			batch := newProfilesRequest(testdata.GenerateProfiles(5))
			// Populate BOTH caches so a missing invalidation would surface as staleness.
			_ = batch.ItemsCount()
			_ = batch.BytesSize()

			res, err := batch.MergeSplit(context.Background(), 0, szt, newProfilesRequest(testdata.GenerateProfiles(7)))
			require.NoError(t, err)
			merged := res[len(res)-1]

			wantItems, wantBytes := freshProfilesSizes(merged)
			assert.Equal(t, wantItems, merged.ItemsCount(), "items after merge")
			assert.Equal(t, wantBytes, merged.BytesSize(), "bytes after merge")
		})
	}
}

// TestProfilesRequestCachedSizeMatchesRecompute exercises long merge/split
// sequences for both sizer dimensions and checks every resulting request's cached
// sizes against a from-scratch recomputation over identical content.
//
// Both dimensions must match exactly. Keeping the byte size exact is what stops
// split from looping past the point where the request still holds samples: a
// cached size that overestimates never falls below maxSize, so extractProfiles
// eventually returns an empty request and split fails with an error.
func TestProfilesRequestCachedSizeMatchesRecompute(t *testing.T) {
	for _, szt := range profilesSizerTypes {
		cases := []struct {
			name    string
			maxSize int
		}{
			{"no_split", 0},
			{"forced_splits", profilesMarshaler.ProfilesSize(testdata.GenerateProfiles(20))},
		}

		for _, tc := range cases {
			t.Run(szt.String()+"/"+tc.name, func(t *testing.T) {
				maxSize := tc.maxSize
				if szt == exporterhelper.RequestSizerTypeItems && maxSize > 0 {
					maxSize = 20
				}
				batch := newProfilesRequest(testdata.GenerateProfiles(3))
				for i := range 20 {
					res, err := batch.MergeSplit(context.Background(), maxSize, szt,
						newProfilesRequest(testdata.GenerateProfiles(11)))
					require.NoError(t, err)
					require.NotEmpty(t, res)
					for _, r := range res {
						wantItems, wantBytes := freshProfilesSizes(r)
						assert.Equalf(t, wantItems, r.ItemsCount(), "iter %d: ItemsCount", i)
						assert.Equalf(t, wantBytes, r.BytesSize(), "iter %d: BytesSize", i)
					}
					batch = res[len(res)-1]
				}
			})
		}
	}
}
