// TODO(@petethepig): This file is here temporarily and should be deleted before we merge profiles spec.

package pprofile

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/jzelinskie/must"

	"go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1/alternatives/pprof"
)

var simpleExample = `foo;bar;baz 100
foo;bar 200
`

var simpleWithLabelsAndTimestamps = `foo;bar;baz 100 region=us,trace_id=0x01020304010203040102030401020304,span_id=0x9999999999999999 1687841528000000
foo;bar 200 region=us
`

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

// TODO
/*

* go traces
* benchmark for a lot of timestamps
* benchmark for a lot of links, attributes, etc

 */

func init() {
	runtime.MemProfileRate = 1

	jfr1 := readJfr("profiles/jfr_1.jfr")
	jfr2 := readJfr("profiles/jfr_2.jfr")
	jfr3 := readJfr("profiles/jfr_3.jfr")
	jfr4 := readJfr("profiles/jfr_4.jfr")
	profiles = map[string]*pprof.Profile{
		"simple":                        collapsedToPprof(simpleExample),
		"simpleWithLabelsAndTimestamps": collapsedToPprof(simpleWithLabelsAndTimestamps),
		"average":                       readPprof("./profiles/average.pprof"),
		"averageTimestamps":             splitIntoManyEventsWithTimestamps(readPprof("./profiles/average.pprof")),
		"ruby":                          readPprof("profiles/ruby_1.pprof"),
		"large":                         readPprof("profiles/large.pprof"),
		"profile_1":                     readPprof("profiles/profile_1.pprof"),
		"profile_2":                     readPprof("profiles/profile_2.pprof"),
		"profile_3":                     readPprof("profiles/profile_3.pprof"),
		"profile_4":                     readPprof("profiles/profile_4.pprof"),
		"profile_5":                     readPprof("profiles/profile_5.pprof"),
		"profile_6":                     readPprof("profiles/profile_6.pprof"),
		"profile_7":                     readPprof("profiles/profile_7.pprof"),
		"profile_8":                     readPprof("profiles/profile_8.pprof"),
		"profile_9":                     readPprof("profiles/profile_9.pprof"),
		"profile_10":                    readPprof("profiles/profile_10.pprof"),
		"profile_11":                    readPprof("profiles/profile_11.pprof"),
		"profile_12":                    readPprof("profiles/profile_12.pprof"),
		"profile_13":                    readPprof("profiles/profile_13.pprof"),
		"jfr_1_1":                       jfr1[0],
		"jfr_1_2":                       jfr1[1],
		"jfr_1_3":                       jfr1[2],
		"jfr_1_4":                       jfr1[3],
		"jfr_1_5":                       jfr1[4],
		"jfr_1_6":                       jfr1[5],
		"jfr_1_7":                       jfr1[6],
		"jfr_2_1":                       jfr2[0],
		"jfr_3_1":                       jfr3[0],
		"jfr_4_1":                       jfr4[0],
		"java_1":                        readPprof("profiles/java.pprof"),
	}
}

var printCache = map[string]bool{}

func benchmark(b *testing.B, profileName string, flavor string) {
	b.ReportAllocs()
	p := profiles[profileName]
	if p == nil {
		fmt.Printf("profile %s not found\n", profileName)
		return
	}
	for i := 0; i < b.N; i++ {
		impact, retainedObject := measureMemoryImpact(func() any {
			return pprofStructToOprof(p, []byte{}, flavor)
		})
		profiles := retainedObject.(Profiles)
		rop := profiles.getOrig()
		if (profileName == "simple" || profileName == "simpleWithLabelsAndTimestamps") && i == 0 && !printCache[profileName+flavor] {
			printCache[profileName+flavor] = true
			fmt.Printf("\n\nspot check [profile:%s; flavor: %s]\n", profileName, flavor)
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

//
// benchmarks
//

func BenchmarkSimplePprof(b *testing.B) {
	benchmark(b, "simple", "pprof")
}

func BenchmarkSimpleDenormalized(b *testing.B) {
	benchmark(b, "simple", "denormalized")
}
func BenchmarkSimpleNormalized(b *testing.B) {
	benchmark(b, "simple", "normalized")
}

func BenchmarkSimpleArrays(b *testing.B) {
	benchmark(b, "simple", "arrays")
}

func BenchmarkSimpleWithLabelsAndTimestampsPprof(b *testing.B) {
	benchmark(b, "simpleWithLabelsAndTimestamps", "pprof")
}

func BenchmarkSimpleWithLabelsAndTimestampsDenormalized(b *testing.B) {
	benchmark(b, "simpleWithLabelsAndTimestamps", "denormalized")
}
func BenchmarkSimpleWithLabelsAndTimestampsNormalized(b *testing.B) {
	benchmark(b, "simpleWithLabelsAndTimestamps", "normalized")
}
func BenchmarkSimpleWithLabelsAndTimestampsArrays(b *testing.B) {
	benchmark(b, "simpleWithLabelsAndTimestamps", "arrays")
}

func BenchmarkAveragePprof(b *testing.B) {
	benchmark(b, "average", "pprof")
}

func BenchmarkAverageDenormalized(b *testing.B) {
	benchmark(b, "average", "denormalized")
}
func BenchmarkAverageNormalized(b *testing.B) {
	benchmark(b, "average", "normalized")
}
func BenchmarkAverageArrays(b *testing.B) {
	benchmark(b, "average", "arrays")
}

func BenchmarkAverageTimestampsPprof(b *testing.B) {
	benchmark(b, "averageTimestamps", "pprof")
}

func BenchmarkAverageTimestampsDenormalized(b *testing.B) {
	benchmark(b, "averageTimestamps", "denormalized")
}
func BenchmarkAverageTimestampsNormalized(b *testing.B) {
	benchmark(b, "averageTimestamps", "normalized")
}
func BenchmarkAverageTimestampsArrays(b *testing.B) {
	benchmark(b, "averageTimestamps", "arrays")
}

func BenchmarkRubyPprof(b *testing.B) {
	benchmark(b, "ruby", "pprof")
}

func BenchmarkRubyDenormalized(b *testing.B) {
	benchmark(b, "ruby", "denormalized")
}
func BenchmarkRubyNormalized(b *testing.B) {
	benchmark(b, "ruby", "normalized")
}
func BenchmarkRubyArrays(b *testing.B) {
	benchmark(b, "ruby", "arrays")
}

func BenchmarkRuby_timestampsPprof(b *testing.B) {
	benchmark(b, "ruby_timestamps", "pprof")
}

func BenchmarkRuby_timestampsDenormalized(b *testing.B) {
	benchmark(b, "ruby_timestamps", "denormalized")
}
func BenchmarkRuby_timestampsNormalized(b *testing.B) {
	benchmark(b, "ruby_timestamps", "normalized")
}
func BenchmarkRuby_timestampsArrays(b *testing.B) {
	benchmark(b, "ruby_timestamps", "arrays")
}

func BenchmarkLargePprof(b *testing.B) {
	benchmark(b, "large", "pprof")
}

func BenchmarkLargeDenormalized(b *testing.B) {
	benchmark(b, "large", "denormalized")
}
func BenchmarkLargeNormalized(b *testing.B) {
	benchmark(b, "large", "normalized")
}
func BenchmarkLargeArrays(b *testing.B) {
	benchmark(b, "large", "arrays")
}

func BenchmarkLargeTimestampsPprof(b *testing.B) {
	benchmark(b, "largeTimestamps", "pprof")
}

func BenchmarkLargeTimestampsDenormalized(b *testing.B) {
	benchmark(b, "largeTimestamps", "denormalized")
}
func BenchmarkLargeTimestampsNormalized(b *testing.B) {
	benchmark(b, "largeTimestamps", "normalized")
}
func BenchmarkLargeTimestampsArrays(b *testing.B) {
	benchmark(b, "largeTimestamps", "arrays")
}

func BenchmarkProfile_1Pprof(b *testing.B) {
	benchmark(b, "profile_1", "pprof")
}

func BenchmarkProfile_1Denormalized(b *testing.B) {
	benchmark(b, "profile_1", "denormalized")
}
func BenchmarkProfile_1Normalized(b *testing.B) {
	benchmark(b, "profile_1", "normalized")
}
func BenchmarkProfile_1Arrays(b *testing.B) {
	benchmark(b, "profile_1", "arrays")
}

func BenchmarkProfile_2Pprof(b *testing.B) {
	benchmark(b, "profile_2", "pprof")
}

func BenchmarkProfile_2Denormalized(b *testing.B) {
	benchmark(b, "profile_2", "denormalized")
}
func BenchmarkProfile_2Normalized(b *testing.B) {
	benchmark(b, "profile_2", "normalized")
}
func BenchmarkProfile_2Arrays(b *testing.B) {
	benchmark(b, "profile_2", "arrays")
}

func BenchmarkProfile_3Pprof(b *testing.B) {
	benchmark(b, "profile_3", "pprof")
}

func BenchmarkProfile_3Denormalized(b *testing.B) {
	benchmark(b, "profile_3", "denormalized")
}
func BenchmarkProfile_3Normalized(b *testing.B) {
	benchmark(b, "profile_3", "normalized")
}
func BenchmarkProfile_3Arrays(b *testing.B) {
	benchmark(b, "profile_3", "arrays")
}

func BenchmarkProfile_4Pprof(b *testing.B) {
	benchmark(b, "profile_4", "pprof")
}

func BenchmarkProfile_4Denormalized(b *testing.B) {
	benchmark(b, "profile_4", "denormalized")
}
func BenchmarkProfile_4Normalized(b *testing.B) {
	benchmark(b, "profile_4", "normalized")
}
func BenchmarkProfile_4Arrays(b *testing.B) {
	benchmark(b, "profile_4", "arrays")
}

func BenchmarkProfile_5Pprof(b *testing.B) {
	benchmark(b, "profile_5", "pprof")
}

func BenchmarkProfile_5Denormalized(b *testing.B) {
	benchmark(b, "profile_5", "denormalized")
}
func BenchmarkProfile_5Normalized(b *testing.B) {
	benchmark(b, "profile_5", "normalized")
}
func BenchmarkProfile_5Arrays(b *testing.B) {
	benchmark(b, "profile_5", "arrays")
}

func BenchmarkProfile_6Pprof(b *testing.B) {
	benchmark(b, "profile_6", "pprof")
}

func BenchmarkProfile_6Denormalized(b *testing.B) {
	benchmark(b, "profile_6", "denormalized")
}
func BenchmarkProfile_6Normalized(b *testing.B) {
	benchmark(b, "profile_6", "normalized")
}
func BenchmarkProfile_6Arrays(b *testing.B) {
	benchmark(b, "profile_6", "arrays")
}

func BenchmarkProfile_7Pprof(b *testing.B) {
	benchmark(b, "profile_7", "pprof")
}

func BenchmarkProfile_7Denormalized(b *testing.B) {
	benchmark(b, "profile_7", "denormalized")
}
func BenchmarkProfile_7Normalized(b *testing.B) {
	benchmark(b, "profile_7", "normalized")
}
func BenchmarkProfile_7Arrays(b *testing.B) {
	benchmark(b, "profile_7", "arrays")
}

func BenchmarkProfile_8Pprof(b *testing.B) {
	benchmark(b, "profile_8", "pprof")
}

func BenchmarkProfile_8Denormalized(b *testing.B) {
	benchmark(b, "profile_8", "denormalized")
}
func BenchmarkProfile_8Normalized(b *testing.B) {
	benchmark(b, "profile_8", "normalized")
}
func BenchmarkProfile_8Arrays(b *testing.B) {
	benchmark(b, "profile_8", "arrays")
}

func BenchmarkProfile_9Pprof(b *testing.B) {
	benchmark(b, "profile_9", "pprof")
}

func BenchmarkProfile_9Denormalized(b *testing.B) {
	benchmark(b, "profile_9", "denormalized")
}
func BenchmarkProfile_9Normalized(b *testing.B) {
	benchmark(b, "profile_9", "normalized")
}
func BenchmarkProfile_9Arrays(b *testing.B) {
	benchmark(b, "profile_9", "arrays")
}

func BenchmarkProfile_10Pprof(b *testing.B) {
	benchmark(b, "profile_10", "pprof")
}

func BenchmarkProfile_10Denormalized(b *testing.B) {
	benchmark(b, "profile_10", "denormalized")
}
func BenchmarkProfile_10Normalized(b *testing.B) {
	benchmark(b, "profile_10", "normalized")
}
func BenchmarkProfile_10Arrays(b *testing.B) {
	benchmark(b, "profile_10", "arrays")
}

func BenchmarkProfile_11Pprof(b *testing.B) {
	benchmark(b, "profile_11", "pprof")
}

func BenchmarkProfile_11Denormalized(b *testing.B) {
	benchmark(b, "profile_11", "denormalized")
}
func BenchmarkProfile_11Normalized(b *testing.B) {
	benchmark(b, "profile_11", "normalized")
}
func BenchmarkProfile_11Arrays(b *testing.B) {
	benchmark(b, "profile_11", "arrays")
}

func BenchmarkProfile_12Pprof(b *testing.B) {
	benchmark(b, "profile_12", "pprof")
}

func BenchmarkProfile_12Denormalized(b *testing.B) {
	benchmark(b, "profile_12", "denormalized")
}
func BenchmarkProfile_12Normalized(b *testing.B) {
	benchmark(b, "profile_12", "normalized")
}
func BenchmarkProfile_12Arrays(b *testing.B) {
	benchmark(b, "profile_12", "arrays")
}

func BenchmarkProfile_13Pprof(b *testing.B) {
	benchmark(b, "profile_13", "pprof")
}

func BenchmarkProfile_13Denormalized(b *testing.B) {
	benchmark(b, "profile_13", "denormalized")
}
func BenchmarkProfile_13Normalized(b *testing.B) {
	benchmark(b, "profile_13", "normalized")
}
func BenchmarkProfile_13Arrays(b *testing.B) {
	benchmark(b, "profile_13", "arrays")
}

func BenchmarkJfr_1_1Pprof(b *testing.B) {
	benchmark(b, "jfr_1_1", "pprof")
}

func BenchmarkJfr_1_1Denormalized(b *testing.B) {
	benchmark(b, "jfr_1_1", "denormalized")
}
func BenchmarkJfr_1_1Normalized(b *testing.B) {
	benchmark(b, "jfr_1_1", "normalized")
}
func BenchmarkJfr_1_1Arrays(b *testing.B) {
	benchmark(b, "jfr_1_1", "arrays")
}

func BenchmarkJfr_1_2Pprof(b *testing.B) {
	benchmark(b, "jfr_1_2", "pprof")
}

func BenchmarkJfr_1_2Denormalized(b *testing.B) {
	benchmark(b, "jfr_1_2", "denormalized")
}
func BenchmarkJfr_1_2Normalized(b *testing.B) {
	benchmark(b, "jfr_1_2", "normalized")
}
func BenchmarkJfr_1_2Arrays(b *testing.B) {
	benchmark(b, "jfr_1_2", "arrays")
}

func BenchmarkJfr_1_3Pprof(b *testing.B) {
	benchmark(b, "jfr_1_3", "pprof")
}

func BenchmarkJfr_1_3Denormalized(b *testing.B) {
	benchmark(b, "jfr_1_3", "denormalized")
}
func BenchmarkJfr_1_3Normalized(b *testing.B) {
	benchmark(b, "jfr_1_3", "normalized")
}
func BenchmarkJfr_1_3Arrays(b *testing.B) {
	benchmark(b, "jfr_1_3", "arrays")
}

func BenchmarkJfr_1_4Pprof(b *testing.B) {
	benchmark(b, "jfr_1_4", "pprof")
}

func BenchmarkJfr_1_4Denormalized(b *testing.B) {
	benchmark(b, "jfr_1_4", "denormalized")
}
func BenchmarkJfr_1_4Normalized(b *testing.B) {
	benchmark(b, "jfr_1_4", "normalized")
}
func BenchmarkJfr_1_4Arrays(b *testing.B) {
	benchmark(b, "jfr_1_4", "arrays")
}

func BenchmarkJfr_1_5Pprof(b *testing.B) {
	benchmark(b, "jfr_1_5", "pprof")
}

func BenchmarkJfr_1_5Denormalized(b *testing.B) {
	benchmark(b, "jfr_1_5", "denormalized")
}
func BenchmarkJfr_1_5Normalized(b *testing.B) {
	benchmark(b, "jfr_1_5", "normalized")
}
func BenchmarkJfr_1_5Arrays(b *testing.B) {
	benchmark(b, "jfr_1_5", "arrays")
}

func BenchmarkJfr_1_6Pprof(b *testing.B) {
	benchmark(b, "jfr_1_6", "pprof")
}

func BenchmarkJfr_1_6Denormalized(b *testing.B) {
	benchmark(b, "jfr_1_6", "denormalized")
}
func BenchmarkJfr_1_6Normalized(b *testing.B) {
	benchmark(b, "jfr_1_6", "normalized")
}
func BenchmarkJfr_1_6Arrays(b *testing.B) {
	benchmark(b, "jfr_1_6", "arrays")
}

func BenchmarkJfr_1_7Pprof(b *testing.B) {
	benchmark(b, "jfr_1_7", "pprof")
}

func BenchmarkJfr_1_7Denormalized(b *testing.B) {
	benchmark(b, "jfr_1_7", "denormalized")
}
func BenchmarkJfr_1_7Normalized(b *testing.B) {
	benchmark(b, "jfr_1_7", "normalized")
}
func BenchmarkJfr_1_7Arrays(b *testing.B) {
	benchmark(b, "jfr_1_7", "arrays")
}

func BenchmarkJfr_2_1Pprof(b *testing.B) {
	benchmark(b, "jfr_2_1", "pprof")
}

func BenchmarkJfr_2_1Denormalized(b *testing.B) {
	benchmark(b, "jfr_2_1", "denormalized")
}
func BenchmarkJfr_2_1Normalized(b *testing.B) {
	benchmark(b, "jfr_2_1", "normalized")
}
func BenchmarkJfr_2_1Arrays(b *testing.B) {
	benchmark(b, "jfr_2_1", "arrays")
}

func BenchmarkJfr_3_1Pprof(b *testing.B) {
	benchmark(b, "jfr_3_1", "pprof")
}

func BenchmarkJfr_3_1Denormalized(b *testing.B) {
	benchmark(b, "jfr_3_1", "denormalized")
}
func BenchmarkJfr_3_1Normalized(b *testing.B) {
	benchmark(b, "jfr_3_1", "normalized")
}
func BenchmarkJfr_3_1Arrays(b *testing.B) {
	benchmark(b, "jfr_3_1", "arrays")
}

func BenchmarkJfr_4_1Pprof(b *testing.B) {
	benchmark(b, "jfr_4_1", "pprof")
}

func BenchmarkJfr_4_1Denormalized(b *testing.B) {
	benchmark(b, "jfr_4_1", "denormalized")
}
func BenchmarkJfr_4_1Normalized(b *testing.B) {
	benchmark(b, "jfr_4_1", "normalized")
}
func BenchmarkJfr_4_1Arrays(b *testing.B) {
	benchmark(b, "jfr_4_1", "arrays")
}

func BenchmarkJava_1Pprof(b *testing.B) {
	benchmark(b, "java_1", "pprof")
}

func BenchmarkJava_1Denormalized(b *testing.B) {
	benchmark(b, "java_1", "denormalized")
}
func BenchmarkJava_1Normalized(b *testing.B) {
	benchmark(b, "java_1", "normalized")
}
func BenchmarkJava_1Arrays(b *testing.B) {
	benchmark(b, "java_1", "arrays")
}
