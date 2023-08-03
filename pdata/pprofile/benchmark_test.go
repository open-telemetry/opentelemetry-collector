// TODO(@petethepig): This file is here temporarily and should be deleted before we merge profiles spec.

package pprofile

import (
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
		"simple": collapsedToPprof(simpleExample),
		// add your own profiles here
		"average": readPprof("./profiles/average.pprof"),
		"large":   readPprof("./profiles/large.pprof"),

		"profile_1":  readPprof("profiles/profile_1.pprof"),
		"profile_2":  readPprof("profiles/profile_2.pprof"),
		"profile_3":  readPprof("profiles/profile_3.pprof"),
		"profile_4":  readPprof("profiles/profile_4.pprof"),
		"profile_5":  readPprof("profiles/profile_5.pprof"),
		"profile_6":  readPprof("profiles/profile_6.pprof"),
		"profile_7":  readPprof("profiles/profile_7.pprof"),
		"profile_8":  readPprof("profiles/profile_8.pprof"),
		"profile_9":  readPprof("profiles/profile_9.pprof"),
		"profile_10": readPprof("profiles/profile_10.pprof"),
		"profile_11": readPprof("profiles/profile_11.pprof"),
		"profile_12": readPprof("profiles/profile_12.pprof"),
		"profile_13": readPprof("profiles/profile_13.pprof"),
		"jfr_1":      readJfr("profiles/jfr_1.jfr")[0],
		"jfr_2":      readJfr("profiles/jfr_1.jfr")[1],
		"jfr_3":      readJfr("profiles/jfr_1.jfr")[2],
		"jfr_4":      readJfr("profiles/jfr_1.jfr")[3],
		"jfr_5":      readJfr("profiles/jfr_1.jfr")[4],
		"jfr_6":      readJfr("profiles/jfr_1.jfr")[5],
		"jfr_7":      readJfr("profiles/jfr_1.jfr")[6],
		"ruby_1":     readPprof("profiles/ruby_1.pprof"),
		"java_1":     readPprof("profiles/java.pprof"),
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

// func BenchmarkSimplePprof(b *testing.B) {
// 	benchmark(b, "simple", "pprof", true)
// }
// func BenchmarkSimpleDenormalized(b *testing.B) {
// 	benchmark(b, "simple", "denormalized", true)
// }
// func BenchmarkSimpleNormalized(b *testing.B) {
// 	benchmark(b, "simple", "normalized", true)
// }
// func BenchmarkSimpleArrays(b *testing.B) {
// 	benchmark(b, "simple", "arrays", true)
// }

// func BenchmarkAveragePprof(b *testing.B) {
// 	benchmark(b, "average", "pprof", false)
// }
// func BenchmarkAverageDenormalized(b *testing.B) {
// 	benchmark(b, "average", "denormalized", false)
// }
// func BenchmarkAverageNormalized(b *testing.B) {
// 	benchmark(b, "average", "normalized", false)
// }
// func BenchmarkAverageArrays(b *testing.B) {
// 	benchmark(b, "average", "arrays", false)
// }

// func BenchmarkLargePprof(b *testing.B) {
// 	benchmark(b, "large", "pprof", false)
// }
// func BenchmarkLargeDenormalized(b *testing.B) {
// 	benchmark(b, "large", "denormalized", false)
// }
// func BenchmarkLargeNormalized(b *testing.B) {
// 	benchmark(b, "large", "normalized", false)
// }
// func BenchmarkLargeArrays(b *testing.B) {
// 	benchmark(b, "large", "arrays", false)
// }

// func BenchmarkProfile1Pprof(b *testing.B) {
// 	benchmark(b, "profile_1", "pprof", false)
// }
// func BenchmarkProfile1Denormalized(b *testing.B) {
// 	benchmark(b, "profile_1", "denormalized", false)
// }
// func BenchmarkProfile1Normalized(b *testing.B) {
// 	benchmark(b, "profile_1", "normalized", false)
// }
// func BenchmarkProfile1Arrays(b *testing.B) {
// 	benchmark(b, "profile_1", "arrays", false)
// }

// func BenchmarkProfile2Pprof(b *testing.B) {
// 	benchmark(b, "profile_2", "pprof", false)
// }
// func BenchmarkProfile2Denormalized(b *testing.B) {
// 	benchmark(b, "profile_2", "denormalized", false)
// }
// func BenchmarkProfile2Normalized(b *testing.B) {
// 	benchmark(b, "profile_2", "normalized", false)
// }
// func BenchmarkProfile2Arrays(b *testing.B) {
// 	benchmark(b, "profile_2", "arrays", false)
// }

// func BenchmarkProfile3Pprof(b *testing.B) {
// 	benchmark(b, "profile_3", "pprof", false)
// }
// func BenchmarkProfile3Denormalized(b *testing.B) {
// 	benchmark(b, "profile_3", "denormalized", false)
// }
// func BenchmarkProfile3Normalized(b *testing.B) {
// 	benchmark(b, "profile_3", "normalized", false)
// }
// func BenchmarkProfile3Arrays(b *testing.B) {
// 	benchmark(b, "profile_3", "arrays", false)
// }

// func BenchmarkProfile4Pprof(b *testing.B) {
// 	benchmark(b, "profile_4", "pprof", false)
// }
// func BenchmarkProfile4Denormalized(b *testing.B) {
// 	benchmark(b, "profile_4", "denormalized", false)
// }
// func BenchmarkProfile4Normalized(b *testing.B) {
// 	benchmark(b, "profile_4", "normalized", false)
// }
// func BenchmarkProfile4Arrays(b *testing.B) {
// 	benchmark(b, "profile_4", "arrays", false)
// }

// func BenchmarkProfile5Pprof(b *testing.B) {
// 	benchmark(b, "profile_5", "pprof", false)
// }
// func BenchmarkProfile5Denormalized(b *testing.B) {
// 	benchmark(b, "profile_5", "denormalized", false)
// }
// func BenchmarkProfile5Normalized(b *testing.B) {
// 	benchmark(b, "profile_5", "normalized", false)
// }
// func BenchmarkProfile5Arrays(b *testing.B) {
// 	benchmark(b, "profile_5", "arrays", false)
// }

// func BenchmarkProfile6Pprof(b *testing.B) {
// 	benchmark(b, "profile_6", "pprof", false)
// }
// func BenchmarkProfile6Denormalized(b *testing.B) {
// 	benchmark(b, "profile_6", "denormalized", false)
// }
// func BenchmarkProfile6Normalized(b *testing.B) {
// 	benchmark(b, "profile_6", "normalized", false)
// }
// func BenchmarkProfile6Arrays(b *testing.B) {
// 	benchmark(b, "profile_6", "arrays", false)
// }

// func BenchmarkProfile7Pprof(b *testing.B) {
// 	benchmark(b, "profile_7", "pprof", false)
// }
// func BenchmarkProfile7Denormalized(b *testing.B) {
// 	benchmark(b, "profile_7", "denormalized", false)
// }
// func BenchmarkProfile7Normalized(b *testing.B) {
// 	benchmark(b, "profile_7", "normalized", false)
// }
// func BenchmarkProfile7Arrays(b *testing.B) {
// 	benchmark(b, "profile_7", "arrays", false)
// }

// func BenchmarkProfile8Pprof(b *testing.B) {
// 	benchmark(b, "profile_8", "pprof", false)
// }
// func BenchmarkProfile8Denormalized(b *testing.B) {
// 	benchmark(b, "profile_8", "denormalized", false)
// }
// func BenchmarkProfile8Normalized(b *testing.B) {
// 	benchmark(b, "profile_8", "normalized", false)
// }
// func BenchmarkProfile8Arrays(b *testing.B) {
// 	benchmark(b, "profile_8", "arrays", false)
// }

// func BenchmarkProfile9Pprof(b *testing.B) {
// 	benchmark(b, "profile_9", "pprof", false)
// }
// func BenchmarkProfile9Denormalized(b *testing.B) {
// 	benchmark(b, "profile_9", "denormalized", false)
// }
// func BenchmarkProfile9Normalized(b *testing.B) {
// 	benchmark(b, "profile_9", "normalized", false)
// }
// func BenchmarkProfile9Arrays(b *testing.B) {
// 	benchmark(b, "profile_9", "arrays", false)
// }

// func BenchmarkProfile10Pprof(b *testing.B) {
// 	benchmark(b, "profile_10", "pprof", false)
// }
// func BenchmarkProfile10Denormalized(b *testing.B) {
// 	benchmark(b, "profile_10", "denormalized", false)
// }
// func BenchmarkProfile10Normalized(b *testing.B) {
// 	benchmark(b, "profile_10", "normalized", false)
// }
// func BenchmarkProfile10Arrays(b *testing.B) {
// 	benchmark(b, "profile_10", "arrays", false)
// }

// func BenchmarkProfile11Pprof(b *testing.B) {
// 	benchmark(b, "profile_11", "pprof", false)
// }
// func BenchmarkProfile11Denormalized(b *testing.B) {
// 	benchmark(b, "profile_11", "denormalized", false)
// }
// func BenchmarkProfile11Normalized(b *testing.B) {
// 	benchmark(b, "profile_11", "normalized", false)
// }
// func BenchmarkProfile11Arrays(b *testing.B) {
// 	benchmark(b, "profile_11", "arrays", false)
// }

// func BenchmarkProfile12Pprof(b *testing.B) {
// 	benchmark(b, "profile_12", "pprof", false)
// }
// func BenchmarkProfile12Denormalized(b *testing.B) {
// 	benchmark(b, "profile_12", "denormalized", false)
// }
// func BenchmarkProfile12Normalized(b *testing.B) {
// 	benchmark(b, "profile_12", "normalized", false)
// }
// func BenchmarkProfile12Arrays(b *testing.B) {
// 	benchmark(b, "profile_12", "arrays", false)
// }

// func BenchmarkProfile13Pprof(b *testing.B) {
// 	benchmark(b, "profile_13", "pprof", false)
// }
// func BenchmarkProfile13Denormalized(b *testing.B) {
// 	benchmark(b, "profile_13", "denormalized", false)
// }
// func BenchmarkProfile13Normalized(b *testing.B) {
// 	benchmark(b, "profile_13", "normalized", false)
// }
// func BenchmarkProfile13Arrays(b *testing.B) {
// 	benchmark(b, "profile_13", "arrays", false)
// }

// func BenchmarkJfr1Pprof(b *testing.B) {
// 	benchmark(b, "jfr_1", "pprof", false)
// }
// func BenchmarkJfr1Denormalized(b *testing.B) {
// 	benchmark(b, "jfr_1", "denormalized", false)
// }
// func BenchmarkJfr1Normalized(b *testing.B) {
// 	benchmark(b, "jfr_1", "normalized", false)
// }
// func BenchmarkJfr1Arrays(b *testing.B) {
// 	benchmark(b, "jfr_1", "arrays", false)
// }

// func BenchmarkJfr2Pprof(b *testing.B) {
// 	benchmark(b, "jfr_2", "pprof", false)
// }
// func BenchmarkJfr2Denormalized(b *testing.B) {
// 	benchmark(b, "jfr_2", "denormalized", false)
// }
// func BenchmarkJfr2Normalized(b *testing.B) {
// 	benchmark(b, "jfr_2", "normalized", false)
// }
// func BenchmarkJfr2Arrays(b *testing.B) {
// 	benchmark(b, "jfr_2", "arrays", false)
// }

// func BenchmarkJfr3Pprof(b *testing.B) {
// 	benchmark(b, "jfr_3", "pprof", false)
// }
// func BenchmarkJfr3Denormalized(b *testing.B) {
// 	benchmark(b, "jfr_3", "denormalized", false)
// }
// func BenchmarkJfr3Normalized(b *testing.B) {
// 	benchmark(b, "jfr_3", "normalized", false)
// }
// func BenchmarkJfr3Arrays(b *testing.B) {
// 	benchmark(b, "jfr_3", "arrays", false)
// }

// func BenchmarkJfr4Pprof(b *testing.B) {
// 	benchmark(b, "jfr_4", "pprof", false)
// }
// func BenchmarkJfr4Denormalized(b *testing.B) {
// 	benchmark(b, "jfr_4", "denormalized", false)
// }
// func BenchmarkJfr4Normalized(b *testing.B) {
// 	benchmark(b, "jfr_4", "normalized", false)
// }
// func BenchmarkJfr4Arrays(b *testing.B) {
// 	benchmark(b, "jfr_4", "arrays", false)
// }

// func BenchmarkJfr5Pprof(b *testing.B) {
// 	benchmark(b, "jfr_5", "pprof", false)
// }
// func BenchmarkJfr5Denormalized(b *testing.B) {
// 	benchmark(b, "jfr_5", "denormalized", false)
// }
// func BenchmarkJfr5Normalized(b *testing.B) {
// 	benchmark(b, "jfr_5", "normalized", false)
// }
// func BenchmarkJfr5Arrays(b *testing.B) {
// 	benchmark(b, "jfr_5", "arrays", false)
// }

// func BenchmarkJfr6Pprof(b *testing.B) {
// 	benchmark(b, "jfr_6", "pprof", false)
// }
// func BenchmarkJfr6Denormalized(b *testing.B) {
// 	benchmark(b, "jfr_6", "denormalized", false)
// }
// func BenchmarkJfr6Normalized(b *testing.B) {
// 	benchmark(b, "jfr_6", "normalized", false)
// }
// func BenchmarkJfr6Arrays(b *testing.B) {
// 	benchmark(b, "jfr_6", "arrays", false)
// }

// func BenchmarkJfr7Pprof(b *testing.B) {
// 	benchmark(b, "jfr_7", "pprof", false)
// }
// func BenchmarkJfr7Denormalized(b *testing.B) {
// 	benchmark(b, "jfr_7", "denormalized", false)
// }
// func BenchmarkJfr7Normalized(b *testing.B) {
// 	benchmark(b, "jfr_7", "normalized", false)
// }
// func BenchmarkJfr7Arrays(b *testing.B) {
// 	benchmark(b, "jfr_7", "arrays", false)
// }

func BenchmarkRuby1Pprof(b *testing.B) {
	benchmark(b, "ruby_1", "pprof", false)
}
func BenchmarkRuby1Denormalized(b *testing.B) {
	benchmark(b, "ruby_1", "denormalized", false)
}
func BenchmarkRuby1Normalized(b *testing.B) {
	benchmark(b, "ruby_1", "normalized", false)
}
func BenchmarkRuby1Arrays(b *testing.B) {
	benchmark(b, "ruby_1", "arrays", false)
}

func BenchmarkJava1Pprof(b *testing.B) {
	benchmark(b, "java_1", "pprof", false)
}
func BenchmarkJava1Denormalized(b *testing.B) {
	benchmark(b, "java_1", "denormalized", false)
}
func BenchmarkJava1Normalized(b *testing.B) {
	benchmark(b, "java_1", "normalized", false)
}
func BenchmarkJava1Arrays(b *testing.B) {
	benchmark(b, "java_1", "arrays", false)
}
