package conversion

import (
	"math"
	"sync"
)

const (
	// mantissaWidth is the size of an IEEE 754 double-precision
	// floating-point mantissa.
	mantissaWidth = 52
	// exponentWidth is the size of an IEEE 754 double-precision
	// floating-point exponent.
	exponentWidth = 11

	// mantissaOnes is mantissaWidth 1 bits
	mantissaOnes = 1<<mantissaWidth - 1

	// exponentBias is the exponent bias specified for encoding
	// the IEEE 754 double-precision floating point exponent.
	exponentBias = 1<<(exponentWidth-1) - 1

	// exponentMask are set to 1 for the bits of an IEEE 754
	// floating point exponent (as distinct from the mantissa and
	// sign.
	exponentMask = ((1 << exponentWidth) - 1) << mantissaWidth
)

var (
	mappingLock sync.Mutex
	mappings    map[int]*exponentialMapping
)

type Mapping interface {
	MapToIndex(float64) int64
}

type Histogram struct {
	*exponential
}

var _ Mapping = &exponentialMapping{}

type exponentialMapping struct {
	// there are 2^scale buckets for the mantissa to map to.
	scale int
	// An array of indices (lookup table). Mantissas of floats are mapped to this array
	// of equidistant buckets, which points to a bucket in the boundaries array
	// that is no further away than two buckets from the correct target bucket.
	indices []int64
	// the array of boundaries, represented as the mantissas of the double values in bits.
	boundaries []uint64
}

func GetExponentialMapping(scale int) Mapping {
	mappingLock.Lock()
	defer mappingLock.Unlock()

	if mappings[scale] != nil {
		return mappings[scale]
	}

	if mappings == nil {
		mappings = map[int]*exponentialMapping{}
	}

	bounds := calculateBoundaries(scale)
	indices := calculateIndices(bounds, scale)

	mapping := &exponentialMapping{
		scale:      scale,
		indices:    indices,
		boundaries: bounds,
	}
	mappings[scale] = mapping
	return mapping
}

// calculate boundaries for the mantissa part only.
// This calculates the bucket boundaries independent of the exponent.
// This mapping is the same for all mantissas, independent of the exponents.
// It depends only on the desired number of buckets subdividing the [1, 2] range covered by the mantissa.
// The number of buckets is 2^scale
func calculateBoundaries(scale int) []uint64 {
	size := 1 << scale

	if len(exponentialConstants) < size {
		// See the code in ./printer to precompute larger constant arrays.
		panic("precomputed boundaries are not available")
	}

	// Note: boundaries is two longer than size to ensure the
	// `mantissa >= el.boundaries[i+1]` test below is correct.
	boundaries := make([]uint64, size+2)
	factor := len(exponentialConstants) / size

	for i := 0; i < size; i++ {
		boundaries[i] = exponentialConstants[i*factor]
	}

	boundaries[size] = 1 << mantissaWidth
	boundaries[size+1] = 1 << mantissaWidth

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
		mantissaLowerBound := uint64(i) << (mantissaWidth - scale)
		// find the lowest boundary that is smaller than or equal to the equidistant bucket bound
		for boundaries[c+1] <= mantissaLowerBound {
			c++
		}
		indices[i] = c
	}
	return indices
}

// This is the code that actually maps the double value to the correct bin.
func (el *exponentialMapping) MapToIndex(value float64) int64 {
	valueBits := math.Float64bits(value)

	// The last 52 bits (bits 0 through 51) of a double are the mantissa.
	// Get these from the valueBits, which is a bit representation of the double value
	mantissa := mantissaOnes & valueBits

	// The bits 52 through 63 are the exponent.
	// extract the exponent from the bit representation of the double value.
	// Then shift it over by 52 bits (the length of the mantissa) and convert it to an integer
	// This also removes the sign bit.
	exponent := int64((exponentMask & valueBits) >> mantissaWidth)
	exponent -= exponentBias

	// find the "rough" bucket index from the indices array
	// The indices array has evenly spaced buckets and contains indices into the boundaries array
	// The line below transforms the normalized mantissa into an index into the indices array
	// Then, the index into the boundaries array is retrieved.
	rough := el.indices[int(mantissa>>(mantissaWidth-el.scale))]

	// The index in the boundaries array might not be correct right away, there might be an offset of a maximum of two buckets.
	// Therefore, look at three buckets: The one specified by the index, and the next two.
	offset := rough
	if mantissa >= el.boundaries[rough+1] {
		offset++
	}
	if mantissa >= el.boundaries[rough+2] {
		offset++
	}

	// the indexOffset is only used to skip subnormal buckets
	// the exponent is used to find the correct "top-level" bucket,
	// and k is used to find the correct index therein.
	return (exponent << el.scale) + offset
}

// upperBoundary is exclusive in this implementation, thus we define
// it as the lowerBoundary of the next bucket.
func (el *exponentialMapping) upperBoundary(index int64) float64 {
	return el.lowerBoundary(index + 1)
}

// lowerBoundary computes the inclusive lower bound corresponding to a
// particular histogram bucket.  This returns the least result value that
// such that `MapToIndex(lowerBoundary(index)) == index`.
func (el *exponentialMapping) lowerBoundary(index int64) float64 {
	length := int64(1 << el.scale)
	exponent := index / length
	position := index % length
	if position < 0 {
		position += length
		exponent -= 1
	}
	mantissa := el.boundaries[position] & mantissaOnes
	expo := uint64((int64(exponent+exponentBias) << mantissaWidth))
	return math.Float64frombits(expo | mantissa)
}
