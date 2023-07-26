// TODO(@petethepig): This file is here temporarily and should be deleted before we merge profiles spec.

package pprofile

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/jzelinskie/must"

	"go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1/alternatives/pprof"
)

var simpleExample = `foo;bar;baz 100 region=us,span_id=123 1687841528000000
foo;bar 200 region=us
`

func prettyPrint(v any) {
	str := string(must.NotError(json.MarshalIndent(v, "", "  ")))
	if len(str) > 5000 {
		str = str[:5000]
	}
	fmt.Println(str)
}

func measureMemoryImpact(cb func() any) (uint64, any) {
	runtime.GC()
	runtime.GC()
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	before := m.HeapObjects
	retainedObjects := []any{}
	retainedObjects = append(retainedObjects, cb())
	runtime.GC()
	runtime.GC()
	runtime.GC()
	runtime.ReadMemStats(&m)
	after := m.HeapObjects

	return after - before, retainedObjects[0]
}

var profiles map[string]*pprof.Profile

func init() {
	runtime.MemProfileRate = 1
	profiles = map[string]*pprof.Profile{
		"simple":  collapsedToPprof(simpleExample),
		// add your own profiles here
		"average": readPprof("./profiles/average.pprof"),
		"large":   readPprof("./profiles/large.pprof"),
	}
}

func benchmark(b *testing.B, profileName string, flavor string, print bool) {
	b.ReportAllocs()
	p := profiles[profileName]
	for i := 0; i < b.N; i++ {
		impact, retainedObject := measureMemoryImpact(func() any {
			return pprofStructToOprof(p, []byte{}, flavor)
		})
		profiles := retainedObject.(Profiles)
		rop := profiles.getOrig()
		if print && i == 0 && os.Getenv("DEBUG") != "" {
			fmt.Println("\nflavor: " + flavor)
			prettyPrint(rop)
		}
		protoBytes := must.NotError(rop.Marshal())
		gProtoBytes := gzipBuffer(protoBytes)
		b.ReportMetric(float64(calculateLabelsCount(p)), "unique_label_sets")
		b.ReportMetric(float64(len(protoBytes)), "bytes")
		b.ReportMetric(float64(impact), "retained_objects")
		b.ReportMetric(float64(len(gProtoBytes)), "gzipped_bytes")
	}
}

func BenchmarkSimplePprof(b *testing.B) {
	benchmark(b, "simple", "pprof", true)
}
func BenchmarkSimpleDenormalized(b *testing.B) {
	benchmark(b, "simple", "denormalized", true)
}
func BenchmarkSimpleNormalized(b *testing.B) {
	benchmark(b, "simple", "normalized", true)
}
func BenchmarkSimpleArrays(b *testing.B) {
	benchmark(b, "simple", "arrays", true)
}

func BenchmarkAveragePprof(b *testing.B) {
	benchmark(b, "average", "pprof", false)
}
func BenchmarkAverageDenormalized(b *testing.B) {
	benchmark(b, "average", "denormalized", false)
}
func BenchmarkAverageNormalized(b *testing.B) {
	benchmark(b, "average", "normalized", false)
}
func BenchmarkAverageArrays(b *testing.B) {
	benchmark(b, "average", "arrays", false)
}

func BenchmarkLargePprof(b *testing.B) {
	benchmark(b, "large", "pprof", false)
}
func BenchmarkLargeDenormalized(b *testing.B) {
	benchmark(b, "large", "denormalized", false)
}
func BenchmarkLargeNormalized(b *testing.B) {
	benchmark(b, "large", "normalized", false)
}
func BenchmarkLargeArrays(b *testing.B) {
	benchmark(b, "large", "arrays", false)
}
