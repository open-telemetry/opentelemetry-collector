package conversion

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/model/pdata"
)

func TestHistogramConversion(t *testing.T) {
	var (
		testAttrs = pdata.NewAttributeMap().InitFromMap(map[string]pdata.AttributeValue{
			"A": pdata.NewAttributeValueString("B"),
		})
		testStartTimestamp = pdata.TimestampFromTime(time.Unix(1, 0))
		testTimestamp      = pdata.TimestampFromTime(time.Unix(2, 0))
		testScale          = int32(3)
		testSum            = float64(1e6)
		testCount          = uint64(10000)
	)

	mp := pdata.NewExponentialHistogramDataPoint()
	testAttrs.CopyTo(mp.Attributes())
	mp.SetStartTimestamp(testStartTimestamp)
	mp.SetTimestamp(testTimestamp)
	mp.SetCount(testCount)
	mp.SetSum(testSum)
	mp.SetScale(testScale)
	mp.Positive().SetOffset(-4)
	mp.Positive().SetBucketCounts([]uint64{
		// 8 buckets offset -4 means indices in the range [-4, 3].
		100, 100, 100, 100, 100, 100, 100, 100,

		// These have positions
		// 2^(-4/8)=0.707107
		// 2^(-3/8)=0.771105
		// 2^(-2/8)=0.840896
		// 2^(-1/8)=0.917004
		// 2^(0/8)=1.000000
		// 2^(1/8)=1.090508
		// 2^(2/8)=1.189207
		// 2^(3/8)=1.296840
		// 2^(4/8)=1.414214
	})

	xp := toExplicitPoint(mp, []float64{
		// explicit boundaries
		0.8, 0.9, 1.0, 1.1, 1.2,
		// map to indices:
		// -3, -2, 0, 1, 2
	})

	require.Equal(t, 800, int(sumUint64s(xp.BucketCounts())))
	require.Equal(t, []uint64{
		141, // (-Inf, 0.8]
		137, // (0.8, 0.9]
		122, // (0.9, 1.0]
		110, // (1.0, 1.1]
		100, // (1.1, 1.2]
		190, // (1.2, +Inf]
	}, xp.BucketCounts())
}

func TestMappingFunction(t *testing.T) {
	for scale := 0; scale < 8; scale++ {
		layout := getExponentialLayout(scale)
		size := int64(1) << 3

		require.Equal(t, 0, layout.mapToBinIndex(1))
		require.Equal(t, size, layout.mapToBinIndex(2))

		require.Equal(t, 1.0, layout.lowerBoundary(0))
		require.Equal(t, 1.0, layout.upperBoundary(-1))

		require.Equal(t, 2.0, layout.lowerBoundary(size))
		require.Equal(t, 2.0, layout.upperBoundary(size-1))
	}
}
