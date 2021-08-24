package conversion

import (
	"fmt"
	"math"
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
		testZeros          = uint64(1000)
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
	})
	mp.SetZeroCount(testZeros)

	xp := toExplicitPoint(mp, []float64{
		0.8, 0.9, 1.0, 1.1, 1.2,
	})

	require.Equal(t, []uint64{
		0,
	}, xp.BucketCounts())
}

func TestB(t *testing.T) {
	scale := 3
	length := 1 << scale
	fmt.Println("------------------------")
	fmt.Println("scale:", scale)
	fmt.Println("number of buckets:", length)
	fmt.Println("------------------------")
	for i := -1; i < length; i++ {
		fmt.Printf("2^(%d/%d)=%f\n", i+1, length, math.Pow(2, float64(i+1)/float64(length)))
	}

	fmt.Println("index mapping:")
	layout := getExponentialLayout(scale)
	fmt.Printf("% 4.5f\t%d\n", 0.0001, layout.mapToBinIndex(0.0001))
	fmt.Printf("% 4.5f\t%d\n", 0.01, layout.mapToBinIndex(0.01))
	fmt.Printf("% 4.5f\t%d\n", 0.1, layout.mapToBinIndex(0.1))
	fmt.Printf("% 4.5f\t%d\n", 1.0, layout.mapToBinIndex(1))
	fmt.Printf("% 4.5f\t%d\n", 2.0, layout.mapToBinIndex(2))
	fmt.Printf("% 4.5f\t%d\n", 3.0, layout.mapToBinIndex(3))
	fmt.Printf("% 4.1f\t\t%d\n", 10.0, layout.mapToBinIndex(10))
	fmt.Printf("% 4.1f\t\t%d\n", 100.0, layout.mapToBinIndex(100))
	fmt.Printf("% 4.1f\t\t%d\n", 1000.0, layout.mapToBinIndex(1000))
	fmt.Printf("%s\t%d\n", "math.SmallestNonzeroFloat64", layout.mapToBinIndex(math.SmallestNonzeroFloat64))
	fmt.Printf("%s\t\t\t%d\n", "math.MaxFloat64", layout.mapToBinIndex(math.MaxFloat64))
}
