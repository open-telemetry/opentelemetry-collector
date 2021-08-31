package histogram

import (
	"sync"

	"go.opentelemetry.io/collector/model/pdata"
)

type halfRange struct {
	offset int64
	counts []uint64
}

type Exponential struct {
	lock     sync.Mutex
	mapping  *exponentialMapping
	positive halfRange
	negative halfRange
	zero     uint64
	total    uint64
	sum      float64
}

const (
	initialCap     = 16
	halfCap        = initialCap / 2
	capRaiseFactor = 2
)

func NewExponential(scale int) *Exponential {
	return &Exponential{
		mapping: getExponentialMapping(scale),
	}
}

func (expo *Exponential) Update(value float64, count uint64) {
	expo.lock.Lock()
	defer expo.lock.Unlock()

	expo.total += count
	expo.sum += value * float64(count)

	if value == 0 {
		expo.zero++
	} else if value > 0 {
		expo.positive.update(expo.mapping.MapToIndex(value), count)
	} else {
		expo.negative.update(expo.mapping.MapToIndex(-value), count)
	}
}

func (hr *halfRange) update(index int64, count uint64) {
	if hr.counts == nil {
		// Initialize with the initial capacity centered on
		// the first index.
		hr.counts = make([]uint64, initialCap)
		hr.offset = index - halfCap
		hr.counts[halfCap-1] = count
		return
	}
	if index < hr.offset {
		// Grow the counts array to the left.
		newLen := len(hr.counts) * capRaiseFactor
		shift := newLen - len(hr.counts)

		tmp := make([]uint64, newLen)
		copy(tmp[shift:], hr.counts)
		hr.offset -= int64(shift)
		hr.counts = tmp
	} else if index >= hr.offset+int64(len(hr.counts)) {
		// Grow the counts array to the right.
		newLen := len(hr.counts) * capRaiseFactor

		tmp := make([]uint64, newLen)
		copy(tmp[:len(hr.counts)], hr.counts)
		hr.counts = tmp
	}

	hr.counts[index-hr.offset] += count
}

func (expo *Exponential) MoveToDataPoint(point pdata.ExponentialHistogramDataPoint) {
	expo.lock.Lock()
	defer expo.lock.Unlock()

	point.SetZeroCount(expo.zero)
	point.SetCount(expo.total)
	point.SetSum(expo.sum)
	expo.zero = 0
	expo.total = 0
	expo.sum = 0

	expo.positive.moveTo(point.Positive())
	expo.negative.moveTo(point.Negative())
}

func (hr *halfRange) moveTo(buckets pdata.Buckets) {
	if hr.counts == nil {
		// No values ever seen
		return
	}
	hrc := hr.counts
	lowIndex := 0
	hr.counts = nil

	for ; lowIndex < len(hrc); lowIndex++ {
		if hrc[lowIndex] != 0 {
			break
		}
	}

	if lowIndex == len(hrc) {
		// All buckets are empty
		return
	}

	highIndex := len(hrc) - 1
	for ; highIndex > 0; highIndex-- {
		if hrc[highIndex] != 0 {
			break
		}
	}

	buckets.SetOffset(hr.offset + int64(lowIndex))
	buckets.SetBucketCounts(hrc[lowIndex : highIndex+1])
}
