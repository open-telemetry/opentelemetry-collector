package conversion

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMappingFunctionConstants(t *testing.T) {
	layout := getExponentialLayout(10)
	require.Equal(t, int64(0), layout.mapToIndex(1))
	require.Equal(t, 1.0, layout.lowerBoundary(0))
	require.Equal(t, 1<<10, len(exponentialConstants))

	require.Equal(t, math.Float64frombits(1023<<52+exponentialConstants[1]), layout.upperBoundary(0))
	require.Equal(t, math.Float64frombits(1023<<52+exponentialConstants[1023]), layout.lowerBoundary(1023))

	require.Equal(t, math.Float64frombits(1024<<52+exponentialConstants[1]), layout.upperBoundary(1024))
	require.Equal(t, math.Float64frombits(1024<<52+exponentialConstants[1023]), layout.lowerBoundary(2047))
}
