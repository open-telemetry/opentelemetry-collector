package proto

import (
	"testing"
)

func BenchmarkExponentialSov(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if Sov(1) != 1 {
			b.Fatal()
		}
	}
}
