package zipkin

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

// Sampler functions return if a Zipkin span should be sampled, based on its
// traceID.
type Sampler func(id uint64) bool

func neverSample(_ uint64) bool { return false }

func alwaysSample(_ uint64) bool { return true }

// NewModuloSampler provides a generic type Sampler.
func NewModuloSampler(mod uint64) Sampler {
	if mod < 2 {
		return alwaysSample
	}
	return func(id uint64) bool {
		return (id % mod) == 0
	}
}

// NewBoundarySampler is appropriate for high-traffic instrumentation who
// provision random trace ids, and make the sampling decision only once.
// It defends against nodes in the cluster selecting exactly the same ids.
func NewBoundarySampler(rate float64, salt int64) (Sampler, error) {
	if rate == 0.0 {
		return neverSample, nil
	}
	if rate == 1.0 {
		return alwaysSample, nil
	}
	if rate < 0.0001 || rate > 1 {
		return nil, fmt.Errorf("rate should be 0.0 or between 0.0001 and 1: was %f", rate)
	}

	var (
		boundary = int64(rate * 10000)
		usalt    = uint64(salt)
	)
	return func(id uint64) bool {
		return int64(math.Abs(float64(id^usalt)))%10000 < boundary
	}, nil
}

// NewCountingSampler is appropriate for low-traffic instrumentation or
// those who do not provision random trace ids. It is not appropriate for
// collectors as the sampling decision isn't idempotent (consistent based
// on trace id).
func NewCountingSampler(rate float64) (Sampler, error) {
	if rate == 0.0 {
		return neverSample, nil
	}
	if rate == 1.0 {
		return alwaysSample, nil
	}
	if rate < 0.01 || rate > 1 {
		return nil, fmt.Errorf("rate should be 0.0 or between 0.01 and 1: was %f", rate)
	}
	var (
		i         = 0
		outOf100  = int(rate*100 + math.Copysign(0.5, rate*100)) // for rounding float to int conversion instead of truncation
		decisions = randomBitSet(100, outOf100, rand.New(rand.NewSource(time.Now().UnixNano())))
		mtx       = &sync.Mutex{}
	)

	return func(_ uint64) bool {
		mtx.Lock()
		result := decisions[i]
		i++
		if i == 100 {
			i = 0
		}
		mtx.Unlock()
		return result
	}, nil
}

/**
 * Reservoir sampling algorithm borrowed from Stack Overflow.
 *
 * http://stackoverflow.com/questions/12817946/generate-a-random-bitset-with-n-1s
 */
func randomBitSet(size int, cardinality int, rnd *rand.Rand) []bool {
	result := make([]bool, size)
	chosen := make([]int, cardinality)
	var i int
	for i = 0; i < cardinality; i++ {
		chosen[i] = i
		result[i] = true
	}
	for ; i < size; i++ {
		j := rnd.Intn(i + 1)
		if j < cardinality {
			result[chosen[j]] = false
			result[i] = true
			chosen[j] = i
		}
	}
	return result
}
