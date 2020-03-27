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
