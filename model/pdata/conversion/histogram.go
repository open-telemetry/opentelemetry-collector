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

// buckets and boundaries are the positive range
func (layout *exponentialLayout) scanPositive(buckets pdata.Buckets, boundaries []float64, counts []uint64) {
	bucketNumber := int64(0)

	for position, bound := range boundaries {
		index := layout.mapToBinIndex(bound)

		for ; bucketNumber < int64(len(buckets.BucketCounts())) && bucketNumber+buckets.Offset() < index; bucketNumber++ {
			one := buckets.BucketCounts()[bucketNumber]
			counts[position] += one
		}

		// The explicit boundary lies between two exponential
		// boundaries, except for the exact match case (TODO).
		lowerBound := layout.lowerBoundary(index)
		upperBound := layout.upperBoundary(index)

		pLow := 1 - (bound-lowerBound)/(upperBound-lowerBound)
		bucketNumber++
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

	} else {
		// True zero-crossing bucket.
		counts[zeroCrosser] += input.ZeroCount()

	}
	return r
}
