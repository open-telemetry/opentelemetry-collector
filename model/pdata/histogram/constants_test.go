package histogram

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMappingFunctionConstants(t *testing.T) {
	mapping := getExponentialMapping(10)
	require.Equal(t, int64(0), mapping.MapToIndex(1))
	require.Equal(t, 1.0, mapping.lowerBoundary(0))
	require.Equal(t, 1<<10, len(exponentialConstants))

	require.Equal(t, math.Float64frombits(1023<<52+exponentialConstants[1]), mapping.upperBoundary(0))
	require.Equal(t, math.Float64frombits(1023<<52+exponentialConstants[1023]), mapping.lowerBoundary(1023))

	require.Equal(t, math.Float64frombits(1024<<52+exponentialConstants[1]), mapping.upperBoundary(1024))
	require.Equal(t, math.Float64frombits(1024<<52+exponentialConstants[1023]), mapping.lowerBoundary(2047))
}
