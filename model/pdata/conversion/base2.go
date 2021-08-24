package conversion

import (
	"math"
	"sync"
)

var (
	layoutLock sync.Mutex
	layouts    map[int]*exponentialLayout
)

type Layout interface {
	MapToIndex(float64) int
}

type exponentialLayout struct {
	// there are 2^scale buckets for the mantissa to map to.
	scale int
	// An array of indices (lookup table). Mantissas of floats are mapped to this array
	// of equidistant buckets, which points to a bucket in the boundaries array
	// that is no further away than two buckets from the correct target bucket.
	indices []int64
	// the array of boundaries, represented as the mantissas of the double values in bits.
	boundaries []uint64
}

func getExponentialLayout(scale int) *exponentialLayout {
	layoutLock.Lock()
	defer layoutLock.Unlock()

	if layouts[scale] != nil {
		return layouts[scale]
	}

	if layouts == nil {
		layouts = map[int]*exponentialLayout{}
	}

	bounds := calculateBoundaries(scale)
	indices := calculateIndices(bounds, scale)

	layout := &exponentialLayout{
		scale:      scale,
		indices:    indices,
		boundaries: bounds,
	}
	layouts[scale] = layout
	return layout
}

// calculate boundaries for the mantissa part only.
// This calculates the bucket boundaries independent of the exponent.
// This layout is the same for all mantissas, independent of the exponents.
// It depends only on the desired number of buckets subdividing the [1, 2] range covered by the mantissa.
// The number of buckets is 2^scale
func calculateBoundaries(scale int) []uint64 {
	length := 1 << scale

	// Note: boundaries is one longer than length to ensure the
	// `mantissa >= el.boundaries[i+1]` test below is correct.
	boundaries := make([]uint64, length+1)

	for i := 0; i < length; i++ {
		boundaries[i] = 0x000fffffffffffff & math.Float64bits(math.Pow(2, float64(i+1)/float64(length)))
	}

	boundaries[length-1] = 0x0010000000000000
	boundaries[length] = 0x0010000000000000

	return boundaries
}

// Create the array which is roughly mapping into the boundaries array
func calculateIndices(boundaries []uint64, scale int) []int64 {
	length := 1 << scale
	indices := make([]int64, length)
	c := int64(0)
	for i := 0; i < length; i++ {
		// e.g. for scale = 2, this evaluates to:
		// i=0: 1.0; i=1: 1.25; i=2: 1.5; i=3: 1.75
		mantissaLowerBound := uint64(i) << (52 - scale)
		// find the lowest boundary that is smaller than or equal to the equidistant bucket bound
		for boundaries[c] <= mantissaLowerBound {
			c++
		}
		indices[i] = c
	}
	return indices
}

// This is the code that actually maps the double value to the correct bin.
func (el *exponentialLayout) mapToBinIndex(value float64) int64 {
	valueBits := math.Float64bits(value)

	// The last 52 bits (bits 0 through 51) of a double are the mantissa.
	// Get these from the valueBits, which is a bit representation of the double value
	mantissa := 0xfffffffffffff & valueBits

	// The bits 52 through 63 are the exponent.
	// extract the exponent from the bit representation of the double value.
	// Then shift it over by 52 bits (the length of the mantissa) and convert it to an integer
	// This also removes the sign bit.
	exponent := int64((0x7ff0000000000000 & valueBits) >> 52)
	// 1023 (2^10-1) is the exponent bias corresponding to 11 exponent bits.
	exponent -= 1023

	// find the "rough" bucket index from the indices array
	// The indices array has evenly spaced buckets and contains indices into the boundaries array
	// The line below transforms the normalized mantissa into an index into the indices array
	// Then, the index into the boundaries array is retrieved.
	i := el.indices[int(mantissa>>(52-el.scale))]

	// The index in the boundaries array might not be correct right away, there might be an offset of a maximum of two buckets.
	// Therefore, look at three buckets: The one specified by the index, and the next two.
	// in other languages, this can be done in one line with a ternary operator
	// e.g. return (exponent << el.scale) + i + (mantissa >= el.boundaries[i] ? 1 : 0) + (mantissa >= el.boundaries[i+1] ? 1 : 0) + el.indexOffset
	offset := i
	if mantissa >= el.boundaries[i] {
		offset++
	}
	if mantissa >= el.boundaries[i+1] {
		offset++
	}
	// the indexOffset is only used to skip subnormal buckets
	// the exponent is used to find the correct "top-level" bucket,
	// and k is used to find the correct index therein.
	return (exponent << el.scale) + offset
}

func (el *exponentialLayout) upperBoundary(index int64) float64 {
	length := int64(1 << el.scale)
	exponent := index / length
	position := index % length
	if position < 0 {
		position += length
		exponent -= 1
	}
	mantissa := el.boundaries[position]
	expo := uint64((int64(exponent+1023) << 52))
	return math.Float64frombits(expo | mantissa)
}

func (el *exponentialLayout) lowerBoundary(index int64) float64 {
	return el.upperBoundary(index - 1)
}
