// Copyright  The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package aggregateprocessor

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test that traceIDs are distributed equally by jumpHash
func TestJumpHashDistribution(t *testing.T) {
	nBuckets := 3
	nTraceIDs := 10000
	deviationMargin := 0.9

	// Generate a bunch of random traceIDs (uint64s)
	randomTraceIDs := make([]uint64, nTraceIDs)
	for i := 0; i < nTraceIDs; i++ {
		randomTraceIDs[i] = rand.Uint64()
	}

	bucketCounts := make([]int, nBuckets)
	for i := 0; i < nBuckets; i++ {
		bucketCounts[i] = 0
	}

	for _, traceID := range randomTraceIDs {
		bucketCounts[jumpHash(traceID, nBuckets)]++
	}

	for _, bucketCount := range bucketCounts {
		assert.Greater(t, float64(bucketCount), deviationMargin*float64(nTraceIDs/nBuckets))
	}
}

func TestJumpHashIdempotence(t *testing.T) {
	nBuckets := 3

	randomTraceID := rand.Uint64()
	first := jumpHash(randomTraceID, nBuckets)
	second := jumpHash(randomTraceID, nBuckets)
	assert.Equal(t, first, second)
}
