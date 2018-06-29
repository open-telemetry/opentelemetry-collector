package statsd

import (
	"fmt"
	"strconv"
	"testing"
)

var statBytes []byte
var stat string

// Results:
// BenchmarkStatBuildGauge_Sprintf-8       	     500	  45699958 ns/op
// BenchmarkStatBuildGauge_Concat-8        	    1000	  23452863 ns/op
// BenchmarkStatBuildGauge_BytesAppend-8   	    1000	  21705121 ns/op
func BenchmarkStatBuildGauge_Sprintf(b *testing.B) {
	for n := 0; n < b.N; n++ {
		for x := 0; x < 100000; x++ {
			stat = fmt.Sprintf("%f|g", 3.14159)
		}
	}
}

func BenchmarkStatBuildGauge_Concat(b *testing.B) {
	for n := 0; n < b.N; n++ {
		for x := 0; x < 100000; x++ {
			stat = strconv.FormatFloat(3.14159, 'f', -1, 64) + "|g"
		}
	}
}

func BenchmarkStatBuildGauge_BytesAppend(b *testing.B) {
	suffix := []byte("|g")

	for n := 0; n < b.N; n++ {
		for x := 0; x < 100000; x++ {
			statBytes = []byte{}
			statBytes = append(strconv.AppendFloat(statBytes, 3.14159, 'f', -1, 64), suffix...)
		}
	}
}

func BenchmarkStatBuildCount_Sprintf(b *testing.B) {
	for n := 0; n < b.N; n++ {
		for x := 0; x < 100000; x++ {
			stat = fmt.Sprintf("%d|c", 314)
		}
	}
}

func BenchmarkStatBuildCount_Concat(b *testing.B) {
	for n := 0; n < b.N; n++ {
		for x := 0; x < 100000; x++ {
			stat = strconv.FormatInt(314, 10) + "|c"
		}
	}
}

func BenchmarkStatBuildCount_BytesAppend(b *testing.B) {
	suffix := []byte("|c")

	for n := 0; n < b.N; n++ {
		for x := 0; x < 100000; x++ {
			statBytes = []byte{}
			statBytes = append(strconv.AppendInt(statBytes, 314, 10), suffix...)
		}
	}
}
