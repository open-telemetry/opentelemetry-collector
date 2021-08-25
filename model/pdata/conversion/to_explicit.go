// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package conversion

import (
	"sort"

	"go.opentelemetry.io/collector/model/pdata"
)

func sumUint64s(values []uint64) (r uint64) {
	for _, value := range values {
		r += value
	}
	return
}

func (layout *exponentialLayout) scanPositive(buckets pdata.Buckets, boundaries []float64, counts []uint64) {
	bucketNumber := int64(0)

	for position, bound := range boundaries {
		index := layout.mapToIndex(bound)

		for ; bucketNumber < int64(len(buckets.BucketCounts())) && (bucketNumber+buckets.Offset()) < index; bucketNumber++ {
			full := buckets.BucketCounts()[bucketNumber]
			counts[position] += full
		}

		if bucketNumber == int64(len(buckets.BucketCounts())) {
			break
		}

		if bucketNumber+buckets.Offset() == index {
			// The explicit boundary lies between two exponential
			// boundaries.  When there is an exact match, it is
			// the lowerBound.
			lowerBound := layout.lowerBoundary(index)
			upperBound := layout.upperBoundary(index)

			pLow := (bound - lowerBound) / (upperBound - lowerBound)

			cnt := buckets.BucketCounts()[bucketNumber]

			// the 0.5 below rounds the number to the nearest integer.
			low := uint64(pLow*float64(cnt) + 0.5)
			high := cnt - low

			counts[position] += low
			counts[position+1] += high
			bucketNumber++
		}
	}

	for ; bucketNumber < int64(len(buckets.BucketCounts())); bucketNumber++ {
		full := buckets.BucketCounts()[bucketNumber]
		counts[len(counts)-1] += full
	}
}

func (layout *exponentialLayout) scanNegative(buckets pdata.Buckets, boundaries []float64, counts []uint64) {
	bucketNumber := int64(0)

	for position := len(boundaries) - 1; position >= 0; position-- {
		bound := boundaries[position]
		index := layout.mapToIndex(-bound)

		for ; bucketNumber < int64(len(buckets.BucketCounts())) && bucketNumber+buckets.Offset() < index; bucketNumber++ {
			full := buckets.BucketCounts()[bucketNumber]
			counts[position+1] += full
		}

		if bucketNumber == int64(len(buckets.BucketCounts())) {
			break
		}

		if bucketNumber+buckets.Offset() == index {
			lowerBound := -layout.lowerBoundary(index)
			upperBound := -layout.upperBoundary(index)
			pLow := (bound - lowerBound) / (upperBound - lowerBound)

			cnt := buckets.BucketCounts()[bucketNumber]
			low := uint64(pLow*float64(cnt) + 0.5)
			high := cnt - low

			counts[position+1] += low
			counts[position] += high
			bucketNumber++
		}
	}

	for ; bucketNumber < int64(len(buckets.BucketCounts())); bucketNumber++ {
		full := buckets.BucketCounts()[bucketNumber]
		counts[0] += full
	}
}

func toExplicitPoint(input pdata.ExponentialHistogramDataPoint, boundaries []float64) pdata.HistogramDataPoint {
	layout := getExponentialLayout(int(input.Scale()))
	counts := make([]uint64, len(boundaries)+1)

	r := pdata.NewHistogramDataPoint()
	input.Attributes().CopyTo(r.Attributes())
	r.SetTimestamp(input.Timestamp())
	r.SetStartTimestamp(input.StartTimestamp())
	r.SetCount(input.Count())
	r.SetSum(input.Sum())
	r.SetExplicitBounds(boundaries)
	r.SetBucketCounts(counts)
	input.Exemplars().CopyTo(r.Exemplars())

	// Find the zero-crossing boundary
	numBoundaries := len(boundaries)
	zeroCrosser := sort.Search(numBoundaries, func(i int) bool {
		return boundaries[i] > 0
	})
	if zeroCrosser == 0 {
		// No zero crosser (first case) means all boundaries
		// are above zero, the first bucket is (-Inf,
		// boundaries[0]].  Copy the zero bucket and negative
		// range.
		counts[0] += input.ZeroCount()
		counts[0] += sumUint64s(input.Negative().BucketCounts())

		layout.scanPositive(input.Positive(), boundaries, counts)

	} else if zeroCrosser == numBoundaries {
		// No zero crosser (second case) means all boundaries
		// are below zero, the last bucket is (boundaries[lastBucket], +Inf)
		lastBucket := numBoundaries - 1
		counts[lastBucket] += input.ZeroCount()
		counts[lastBucket] += sumUint64s(input.Positive().BucketCounts())

		layout.scanNegative(input.Negative(), boundaries, counts)

	} else if boundaries[zeroCrosser-1] == 0 {
		// We have a boundary at zero as in (-1, 0], (0, 1]
		counts[zeroCrosser-1] += input.ZeroCount()

		layout.scanNegative(input.Negative(), boundaries[0:zeroCrosser-1], counts[0:zeroCrosser])
		layout.scanPositive(input.Positive(), boundaries[zeroCrosser:], counts[zeroCrosser:])
	} else {
		// We have a bucket that spans zero.
		counts[zeroCrosser] += input.ZeroCount()

		layout.scanNegative(input.Negative(), boundaries[0:zeroCrosser], counts[0:zeroCrosser+1])
		layout.scanPositive(input.Positive(), boundaries[zeroCrosser:], counts[zeroCrosser:])
	}
	return r
}
