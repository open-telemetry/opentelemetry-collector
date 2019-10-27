package sampling

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIsSampled(t *testing.T) {
	// Do Not Sample
	var sampler Sampler = NewCountingSampler(0.0)
	assert.Equal(t, getSampleBucketSize(sampler, 100), 0)
	assert.Equal(t, getSampleBucketSize(sampler, 1000), 0)

	// Sample 10%
	sampler = NewCountingSampler(0.1)
	assert.Equal(t, getSampleBucketSize(sampler, 100), 10)
	assert.Equal(t, getSampleBucketSize(sampler, 1000), 100)

	// Sample 20%
	sampler = NewCountingSampler(0.2)
	assert.Equal(t, getSampleBucketSize(sampler, 100), 20)
	assert.Equal(t, getSampleBucketSize(sampler, 1000), 200)

	// Sample 50%
	sampler = NewCountingSampler(0.5)
	assert.Equal(t, getSampleBucketSize(sampler, 100), 50)
	assert.Equal(t, getSampleBucketSize(sampler, 1000), 500)

	// Sample 100%
	sampler = NewCountingSampler(1.0)
	assert.Equal(t, getSampleBucketSize(sampler, 100), 100)
	assert.Equal(t, getSampleBucketSize(sampler, 1000), 1000)
}

func getSampleBucketSize(sampler Sampler, traceCount int) int {
	j := 0
	for i := 0; i < traceCount; i++ {
		if sampler.isSampled() {
			j++
		}
	}
	return j
}
