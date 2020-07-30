package cortexexporter
 import (
	 otlp "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
	 "strconv"

	 // "go.opentelemetry.io/collector/internal/data/testdata"
	 "testing"
	 "github.com/stretchr/testify/assert"
	// "github.com/stretchr/testify/require"
 )


func Test_validateMetrics(t *testing.T) {
	type combTest struct {
		name string
		desc *otlp.MetricDescriptor
		want bool
	}
	tests := []combTest{}
	// append true cases
	for i, _ := range validCombinations {
		name := "validateMetric_"+ strconv.Itoa(i)
		desc := generateDescriptor(name, i, validCombinations)
		tests = append(tests,  combTest{
			name,
			desc,
			true,
		})
	}
	for i, _ := range invalidCombinations {
		name := "invalidateMetric_"+ strconv.Itoa(i)
		desc := generateDescriptor(name, i,invalidCombinations)
		tests = append(tests,  combTest{
			name,
			desc,
			false,
		})
	}
	tests = append(tests, combTest{"invalidMertics_nil", nil, false})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateMetrics(tt.desc)
			assert.Equal(t, tt.want, got)
		})
	}
}



// ------ Utilities ---------

func generateDescriptor(name string, i int, comb[]combination) *otlp.MetricDescriptor {

	return &otlp.MetricDescriptor{
		Name:        name,
		Description: "",
		Unit:        "1",
		Type:        comb[i].ty,
		Temporality: comb[i].temp,
	}
}
type combination struct {
	ty   otlp.MetricDescriptor_Type
	temp otlp.MetricDescriptor_Temporality
}
var (
	validCombinations = []combination {
		{otlp.MetricDescriptor_MONOTONIC_INT64,otlp.MetricDescriptor_CUMULATIVE},
		{otlp.MetricDescriptor_MONOTONIC_DOUBLE, otlp.MetricDescriptor_CUMULATIVE},
		{otlp.MetricDescriptor_HISTOGRAM, otlp.MetricDescriptor_CUMULATIVE},
		{otlp.MetricDescriptor_SUMMARY, otlp.MetricDescriptor_CUMULATIVE},
		{otlp.MetricDescriptor_INT64, otlp.MetricDescriptor_DELTA},
		{otlp.MetricDescriptor_DOUBLE, otlp.MetricDescriptor_DELTA},
		{otlp.MetricDescriptor_INT64, otlp.MetricDescriptor_INSTANTANEOUS},
		{otlp.MetricDescriptor_DOUBLE, otlp.MetricDescriptor_INSTANTANEOUS},
		{otlp.MetricDescriptor_INT64, otlp.MetricDescriptor_CUMULATIVE},
		{otlp.MetricDescriptor_DOUBLE, otlp.MetricDescriptor_CUMULATIVE},
	}
	invalidCombinations = [] combination {
		{otlp.MetricDescriptor_MONOTONIC_INT64,otlp.MetricDescriptor_DELTA},
		{otlp.MetricDescriptor_MONOTONIC_DOUBLE, otlp.MetricDescriptor_DELTA},
		{otlp.MetricDescriptor_HISTOGRAM, otlp.MetricDescriptor_DELTA},
		{otlp.MetricDescriptor_SUMMARY, otlp.MetricDescriptor_DELTA},
		{otlp.MetricDescriptor_MONOTONIC_INT64,otlp.MetricDescriptor_DELTA},
		{otlp.MetricDescriptor_MONOTONIC_DOUBLE, otlp.MetricDescriptor_DELTA},
		{otlp.MetricDescriptor_HISTOGRAM, otlp.MetricDescriptor_DELTA},
		{otlp.MetricDescriptor_SUMMARY, otlp.MetricDescriptor_DELTA},
		{ty: otlp.MetricDescriptor_INVALID_TYPE},
		{temp: otlp.MetricDescriptor_INVALID_TEMPORALITY},
		{},
	}
)

