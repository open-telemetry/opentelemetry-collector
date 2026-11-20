// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package reclaim evicts cold startup file-backed binary pages on Linux after
// Collector initialization completes.
package reclaim // import "go.opentelemetry.io/collector/service/internal/reclaim"

import (
	"cmp"
	"math"
	"slices"
)

// Result records summary counts from a cold binary page reclaim attempt.
type Result struct {
	RangesAdvised int
	BytesAdvised  uint64
	SkippedReason string
}

// interval represents a half-open byte range [start, end).
type interval struct {
	start uint64
	end   uint64
}

func alignDown(v, pageSize uint64) uint64 {
	if pageSize == 0 {
		return v
	}
	return v &^ (pageSize - 1)
}

func alignUp(v, pageSize uint64) (uint64, bool) {
	if pageSize == 0 {
		return v, true
	}
	rem := v & (pageSize - 1)
	if rem == 0 {
		return v, true
	}
	add := pageSize - rem
	if v > math.MaxUint64-add {
		return 0, false
	}
	return v + add, true
}

func mergeIntervals(in []interval) []interval {
	var valid []interval
	for _, iv := range in {
		if iv.end > iv.start {
			valid = append(valid, iv)
		}
	}
	if len(valid) <= 1 {
		return valid
	}
	slices.SortFunc(valid, func(a, b interval) int {
		if c := cmp.Compare(a.start, b.start); c != 0 {
			return c
		}
		return cmp.Compare(a.end, b.end)
	})
	merged := []interval{valid[0]}
	for _, iv := range valid[1:] {
		last := &merged[len(merged)-1]
		if iv.start <= last.end {
			if iv.end > last.end {
				last.end = iv.end
			}
		} else {
			merged = append(merged, iv)
		}
	}
	return merged
}

// subtractIntervals removes all excluded intervals from base, returning sorted,
// non-overlapping remaining intervals.
func subtractIntervals(base, excluded []interval) []interval {
	cur := mergeIntervals(base)
	for _, ex := range excluded {
		if ex.end <= ex.start {
			continue
		}
		var next []interval
		for _, iv := range cur {
			if ex.end <= iv.start || ex.start >= iv.end {
				next = append(next, iv)
				continue
			}
			if ex.start > iv.start {
				next = append(next, interval{start: iv.start, end: ex.start})
			}
			if ex.end < iv.end {
				next = append(next, interval{start: ex.end, end: iv.end})
			}
		}
		cur = next
	}
	return cur
}

func intersectInterval(a, b interval) (interval, bool) {
	s := max(a.start, b.start)
	e := min(a.end, b.end)
	if e <= s {
		return interval{}, false
	}
	return interval{start: s, end: e}, true
}
