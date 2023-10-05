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

func selectProfile(arr []*pprof.Profile, i int) *pprof.Profile {
	if arr == nil {
		return nil
	}
	return arr[i]
}

func init() {
	runtime.MemProfileRate = 1
	jfr1 := readJfr("profiles/jfr_1.jfr")
	// jfr2 := readJfr("profiles/jfr_2.jfr")
	// jfr3 := readJfr("profiles/jfr_3.jfr")
	// jfr4 := readJfr("profiles/jfr_4.jfr")
	profiles = map[string]*pprof.Profile{
		"simple":                        collapsedToPprof(simpleExample),
		"simpleWithLabelsAndTimestamps": collapsedToPprof(simpleWithLabelsAndTimestamps),
		"average":                       readPprof("./profiles/average.pprof"),
		"averageTimestamps":             splitIntoManyEventsWithTimestamps(readPprof("./profiles/average.pprof")),
		"ruby":                          readPprof("profiles/ruby_1.pprof"),
		// "large10percent":                limitSamples(readPprof("profiles/large.pprof"), 0.1),
		// "large1percent": limitSamples(readPprof("profiles/large.pprof"), 0.001),
		"large": readPprof("profiles/large.pprof"),
		// "profile_1":                     readPprof("profiles/profile_1.pprof"),
		// "profile_2":                     readPprof("profiles/profile_2.pprof"),
		// "profile_3":                     readPprof("profiles/profile_3.pprof"),
		// "profile_4":                     readPprof("profiles/profile_4.pprof"),
		// "profile_5":                     readPprof("profiles/profile_5.pprof"),
		// "profile_6":                     readPprof("profiles/profile_6.pprof"),
		// "profile_7":                     readPprof("profiles/profile_7.pprof"),
		// "profile_8":                     readPprof("profiles/profile_8.pprof"),
		// "profile_9":                     readPprof("profiles/profile_9.pprof"),
		// "profile_10":                    readPprof("profiles/profile_10.pprof"),
		// "profile_11":                    readPprof("profiles/profile_11.pprof"),
		// "profile_12":                    readPprof("profiles/profile_12.pprof"),
		// "profile_13":                    readPprof("profiles/profile_13.pprof"),
		"jfr_1_1": selectProfile(jfr1, 0),
		// "jfr_1_2":                       selectProfile(jfr1, 1),
		// "jfr_1_3":                       selectProfile(jfr1, 2),
		// "jfr_1_4":                       selectProfile(jfr1, 3),
		// "jfr_1_5":                       selectProfile(jfr1, 4),
		// "jfr_1_6":                       selectProfile(jfr1, 5),
		// "jfr_1_7":                       selectProfile(jfr1, 6),
		// "jfr_2_1":                       selectProfile(jfr2, 0),
		// "jfr_3_1":                       selectProfile(jfr3, 0),
		// "jfr_4_1":                       selectProfile(jfr4, 0),
		"java_1": readPprof("profiles/java.pprof"),
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
		if (profileName == "simple" || profileName == "simpleWithLabelsAndTimestamps" || profileName == "ruby2") && i == 0 && !printCache[profileName+flavor] {
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
func BenchmarkSimplePprofExtended(b *testing.B) {
	benchmark(b, "simple", "pprofextended")
}
func BenchmarkSimpleArrays(b *testing.B) {
	benchmark(b, "simple", "arrays")
}

func BenchmarkSimpleWithLabelsAndTimestampsPprof(b *testing.B) {
	benchmark(b, "simpleWithLabelsAndTimestamps", "pprof")
}
func BenchmarkSimpleWithLabelsAndTimestampsPprofExtended(b *testing.B) {
	benchmark(b, "simpleWithLabelsAndTimestamps", "pprofextended")
}
func BenchmarkSimpleWithLabelsAndTimestampsArrays(b *testing.B) {
	benchmark(b, "simpleWithLabelsAndTimestamps", "arrays")
}

func BenchmarkAveragePprof(b *testing.B) {
	benchmark(b, "average", "pprof")
}
func BenchmarkAveragePprofExtended(b *testing.B) {
	benchmark(b, "average", "pprofextended")
}
func BenchmarkAverageArrays(b *testing.B) {
	benchmark(b, "average", "arrays")
}

func BenchmarkAverageTimestampsPprof(b *testing.B) {
	benchmark(b, "averageTimestamps", "pprof")
}
func BenchmarkAverageTimestampsPprofExtended(b *testing.B) {
	benchmark(b, "averageTimestamps", "pprofextended")
}
func BenchmarkAverageTimestampsArrays(b *testing.B) {
	benchmark(b, "averageTimestamps", "arrays")
}

func BenchmarkRubyPprof(b *testing.B) {
	benchmark(b, "ruby", "pprof")
}
func BenchmarkRubyPprofExtended(b *testing.B) {
	benchmark(b, "ruby", "pprofextended")
}
func BenchmarkRubyArrays(b *testing.B) {
	benchmark(b, "ruby", "arrays")
}

func BenchmarkLarge10percentPprof(b *testing.B) {
	benchmark(b, "large10percent", "pprof")
}
func BenchmarkLarge10percentPprofExtended(b *testing.B) {
	benchmark(b, "large10percent", "pprofextended")
}
func BenchmarkLarge10percentArrays(b *testing.B) {
	benchmark(b, "large10percent", "arrays")
}

func BenchmarkLarge1percentPprof(b *testing.B) {
	benchmark(b, "large1percent", "pprof")
}
func BenchmarkLarge1percentPprofExtended(b *testing.B) {
	benchmark(b, "large1percent", "pprofextended")
}
func BenchmarkLarge1percentArrays(b *testing.B) {
	benchmark(b, "large1percent", "arrays")
}

func BenchmarkLargePprof(b *testing.B) {
	benchmark(b, "large", "pprof")
}
func BenchmarkLargePprofExtended(b *testing.B) {
	benchmark(b, "large", "pprofextended")
}
func BenchmarkLargeArrays(b *testing.B) {
	benchmark(b, "large", "arrays")
}

func BenchmarkProfile_1Pprof(b *testing.B) {
	benchmark(b, "profile_1", "pprof")
}
func BenchmarkProfile_1PprofExtended(b *testing.B) {
	benchmark(b, "profile_1", "pprofextended")
}
func BenchmarkProfile_1Arrays(b *testing.B) {
	benchmark(b, "profile_1", "arrays")
}

func BenchmarkProfile_2Pprof(b *testing.B) {
	benchmark(b, "profile_2", "pprof")
}
func BenchmarkProfile_2PprofExtended(b *testing.B) {
	benchmark(b, "profile_2", "pprofextended")
}
func BenchmarkProfile_2Arrays(b *testing.B) {
	benchmark(b, "profile_2", "arrays")
}

func BenchmarkProfile_3Pprof(b *testing.B) {
	benchmark(b, "profile_3", "pprof")
}
func BenchmarkProfile_3PprofExtended(b *testing.B) {
	benchmark(b, "profile_3", "pprofextended")
}
func BenchmarkProfile_3Arrays(b *testing.B) {
	benchmark(b, "profile_3", "arrays")
}

func BenchmarkProfile_4Pprof(b *testing.B) {
	benchmark(b, "profile_4", "pprof")
}
func BenchmarkProfile_4PprofExtended(b *testing.B) {
	benchmark(b, "profile_4", "pprofextended")
}
func BenchmarkProfile_4Arrays(b *testing.B) {
	benchmark(b, "profile_4", "arrays")
}

func BenchmarkProfile_5Pprof(b *testing.B) {
	benchmark(b, "profile_5", "pprof")
}
func BenchmarkProfile_5PprofExtended(b *testing.B) {
	benchmark(b, "profile_5", "pprofextended")
}
func BenchmarkProfile_5Arrays(b *testing.B) {
	benchmark(b, "profile_5", "arrays")
}

func BenchmarkProfile_6Pprof(b *testing.B) {
	benchmark(b, "profile_6", "pprof")
}
func BenchmarkProfile_6PprofExtended(b *testing.B) {
	benchmark(b, "profile_6", "pprofextended")
}
func BenchmarkProfile_6Arrays(b *testing.B) {
	benchmark(b, "profile_6", "arrays")
}

func BenchmarkProfile_7Pprof(b *testing.B) {
	benchmark(b, "profile_7", "pprof")
}
func BenchmarkProfile_7PprofExtended(b *testing.B) {
	benchmark(b, "profile_7", "pprofextended")
}
func BenchmarkProfile_7Arrays(b *testing.B) {
	benchmark(b, "profile_7", "arrays")
}

func BenchmarkProfile_8Pprof(b *testing.B) {
	benchmark(b, "profile_8", "pprof")
}
func BenchmarkProfile_8PprofExtended(b *testing.B) {
	benchmark(b, "profile_8", "pprofextended")
}
func BenchmarkProfile_8Arrays(b *testing.B) {
	benchmark(b, "profile_8", "arrays")
}

func BenchmarkProfile_9Pprof(b *testing.B) {
	benchmark(b, "profile_9", "pprof")
}
func BenchmarkProfile_9PprofExtended(b *testing.B) {
	benchmark(b, "profile_9", "pprofextended")
}
func BenchmarkProfile_9Arrays(b *testing.B) {
	benchmark(b, "profile_9", "arrays")
}

func BenchmarkProfile_10Pprof(b *testing.B) {
	benchmark(b, "profile_10", "pprof")
}
func BenchmarkProfile_10PprofExtended(b *testing.B) {
	benchmark(b, "profile_10", "pprofextended")
}
func BenchmarkProfile_10Arrays(b *testing.B) {
	benchmark(b, "profile_10", "arrays")
}

func BenchmarkProfile_11Pprof(b *testing.B) {
	benchmark(b, "profile_11", "pprof")
}
func BenchmarkProfile_11PprofExtended(b *testing.B) {
	benchmark(b, "profile_11", "pprofextended")
}
func BenchmarkProfile_11Arrays(b *testing.B) {
	benchmark(b, "profile_11", "arrays")
}

func BenchmarkProfile_12Pprof(b *testing.B) {
	benchmark(b, "profile_12", "pprof")
}
func BenchmarkProfile_12PprofExtended(b *testing.B) {
	benchmark(b, "profile_12", "pprofextended")
}
func BenchmarkProfile_12Arrays(b *testing.B) {
	benchmark(b, "profile_12", "arrays")
}

func BenchmarkProfile_13Pprof(b *testing.B) {
	benchmark(b, "profile_13", "pprof")
}
func BenchmarkProfile_13PprofExtended(b *testing.B) {
	benchmark(b, "profile_13", "pprofextended")
}
func BenchmarkProfile_13Arrays(b *testing.B) {
	benchmark(b, "profile_13", "arrays")
}

func BenchmarkJfr_1_1Pprof(b *testing.B) {
	benchmark(b, "jfr_1_1", "pprof")
}
func BenchmarkJfr_1_1PprofExtended(b *testing.B) {
	benchmark(b, "jfr_1_1", "pprofextended")
}
func BenchmarkJfr_1_1Arrays(b *testing.B) {
	benchmark(b, "jfr_1_1", "arrays")
}

func BenchmarkJfr_1_2Pprof(b *testing.B) {
	benchmark(b, "jfr_1_2", "pprof")
}
func BenchmarkJfr_1_2PprofExtended(b *testing.B) {
	benchmark(b, "jfr_1_2", "pprofextended")
}
func BenchmarkJfr_1_2Arrays(b *testing.B) {
	benchmark(b, "jfr_1_2", "arrays")
}

func BenchmarkJfr_1_3Pprof(b *testing.B) {
	benchmark(b, "jfr_1_3", "pprof")
}
func BenchmarkJfr_1_3PprofExtended(b *testing.B) {
	benchmark(b, "jfr_1_3", "pprofextended")
}
func BenchmarkJfr_1_3Arrays(b *testing.B) {
	benchmark(b, "jfr_1_3", "arrays")
}

func BenchmarkJfr_1_4Pprof(b *testing.B) {
	benchmark(b, "jfr_1_4", "pprof")
}
func BenchmarkJfr_1_4PprofExtended(b *testing.B) {
	benchmark(b, "jfr_1_4", "pprofextended")
}
func BenchmarkJfr_1_4Arrays(b *testing.B) {
	benchmark(b, "jfr_1_4", "arrays")
}

func BenchmarkJfr_1_5Pprof(b *testing.B) {
	benchmark(b, "jfr_1_5", "pprof")
}
func BenchmarkJfr_1_5PprofExtended(b *testing.B) {
	benchmark(b, "jfr_1_5", "pprofextended")
}
func BenchmarkJfr_1_5Arrays(b *testing.B) {
	benchmark(b, "jfr_1_5", "arrays")
}

func BenchmarkJfr_1_6Pprof(b *testing.B) {
	benchmark(b, "jfr_1_6", "pprof")
}
func BenchmarkJfr_1_6PprofExtended(b *testing.B) {
	benchmark(b, "jfr_1_6", "pprofextended")
}
func BenchmarkJfr_1_6Arrays(b *testing.B) {
	benchmark(b, "jfr_1_6", "arrays")
}

func BenchmarkJfr_1_7Pprof(b *testing.B) {
	benchmark(b, "jfr_1_7", "pprof")
}
func BenchmarkJfr_1_7PprofExtended(b *testing.B) {
	benchmark(b, "jfr_1_7", "pprofextended")
}
func BenchmarkJfr_1_7Arrays(b *testing.B) {
	benchmark(b, "jfr_1_7", "arrays")
}

func BenchmarkJfr_2_1Pprof(b *testing.B) {
	benchmark(b, "jfr_2_1", "pprof")
}
func BenchmarkJfr_2_1PprofExtended(b *testing.B) {
	benchmark(b, "jfr_2_1", "pprofextended")
}
func BenchmarkJfr_2_1Arrays(b *testing.B) {
	benchmark(b, "jfr_2_1", "arrays")
}

func BenchmarkJfr_3_1Pprof(b *testing.B) {
	benchmark(b, "jfr_3_1", "pprof")
}
func BenchmarkJfr_3_1PprofExtended(b *testing.B) {
	benchmark(b, "jfr_3_1", "pprofextended")
}
func BenchmarkJfr_3_1Arrays(b *testing.B) {
	benchmark(b, "jfr_3_1", "arrays")
}

func BenchmarkJfr_4_1Pprof(b *testing.B) {
	benchmark(b, "jfr_4_1", "pprof")
}
func BenchmarkJfr_4_1PprofExtended(b *testing.B) {
	benchmark(b, "jfr_4_1", "pprofextended")
}
func BenchmarkJfr_4_1Arrays(b *testing.B) {
	benchmark(b, "jfr_4_1", "arrays")
}

func BenchmarkJava_1Pprof(b *testing.B) {
	benchmark(b, "java_1", "pprof")
}
func BenchmarkJava_1PprofExtended(b *testing.B) {
	benchmark(b, "java_1", "pprofextended")
}
func BenchmarkJava_1Arrays(b *testing.B) {
	benchmark(b, "java_1", "arrays")
}
