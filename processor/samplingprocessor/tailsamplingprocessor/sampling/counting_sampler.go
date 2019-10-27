package sampling

import (
	"math/rand"
	"sync/atomic"
	"time"
)

//DIVISOR : Divisor constant
const DIVISOR = 100

//Sampler : Sampler contract
type Sampler interface {
	isSampled() bool
}

//CountingSampler : Counting sampler is implementation of sampler
type CountingSampler struct {
	samplingRate float32
	randomMap    map[int]int
}

type atomiccounter int64

func (a *atomiccounter) increment() int64 {
	var next int64
	for {
		next = int64(*a) + 1
		if atomic.CompareAndSwapInt64((*int64)(a), int64(*a), next) {
			return next
		}
	}
}

// Initialize the randomMap using random
func (c CountingSampler) init() {
	rand.Seed(time.Now().UnixNano())
	percentage := c.samplingRate * DIVISOR
	rndRange := rand.Perm(DIVISOR)[:int(percentage)]
	for _, s := range rndRange {
		c.randomMap[s]++
	}
}

var counter atomiccounter

// Checks if the counter value falls within the random map
func (c CountingSampler) isSampled() bool {
	value := mod(counter.increment(), DIVISOR)
	if _, ok := c.randomMap[int(value)]; ok {
		return true
	}
	return false
}

//NewCountingSampler : Create instance of counting sampler
func NewCountingSampler(samplingRate float32) CountingSampler {
	sampler := CountingSampler{samplingRate, make(map[int]int)}
	sampler.init()
	return sampler
}

func mod(dividend int64, divisor int64) float32 {
	var result = float32(dividend % divisor)
	if result >= 0 {
		return result
	}
	return float32(divisor) + result
}
