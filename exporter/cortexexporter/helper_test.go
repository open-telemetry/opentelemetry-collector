package cortexexporter

import (
	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	common "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
	otlp "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
	"strconv"
	"testing"
)

//return false if descriptor type is nil
func Test_validateMetrics(t *testing.T) {
	// define a single test
	type combTest struct {
		name string
		desc *otlp.MetricDescriptor
		want bool
	}

	tests := []combTest{}

	// append true cases
	for i := range validCombinations {
		name := "valid_" + strconv.Itoa(i)
		desc := getDescriptor(name, i, validCombinations)
		tests = append(tests, combTest{
			name,
			desc,
			true,
		})
	}
	// append false cases
	for i := range invalidCombinations {
		name := "invalid_" + strconv.Itoa(i)
		desc := getDescriptor(name, i, invalidCombinations)
		tests = append(tests, combTest{
			name,
			desc,
			false,
		})
	}
	// append nil case
	tests = append(tests, combTest{"invalid_nil", nil, false})

	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateMetrics(tt.desc)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_addSample(t *testing.T) {
	type testCase struct {
		desc   otlp.MetricDescriptor_Type
		sample prompb.Sample
		labels []prompb.Label
	}

	tests := []struct {
		name     string
		orig     map[string]*prompb.TimeSeries
		testCase []testCase
		want     map[string]*prompb.TimeSeries
	}{
		{
			"two_points_same_ts_same_metric",
			map[string]*prompb.TimeSeries{},
			[]testCase{
				{otlp.MetricDescriptor_INT64,
					getSample(float64(intVal1), time1),
					promLbs1,
				},
				{
					otlp.MetricDescriptor_INT64,
					getSample(float64(intVal2), time2),
					promLbs1,
				},
			},
			twoPointsSameTs,
		},
		{
			"two_points_different_ts_same_metric",
			map[string]*prompb.TimeSeries{},
			[]testCase{
				{otlp.MetricDescriptor_INT64,
					getSample(float64(intVal1), time1),
					promLbs1,
				},
				{otlp.MetricDescriptor_INT64,
					getSample(float64(intVal1), time2),
					promLbs2,
				},
			},
			twoPointsDifferentTs,
		},
	}
	t.Run("nil_case", func(t *testing.T) {
		tsMap := map[string]*prompb.TimeSeries{}
		addSample(tsMap, nil, nil, 0)
		assert.Exactly(t, tsMap, map[string]*prompb.TimeSeries{})
	})
	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addSample(tt.orig, &tt.testCase[0].sample, tt.testCase[0].labels, tt.testCase[0].desc)
			addSample(tt.orig, &tt.testCase[1].sample, tt.testCase[1].labels, tt.testCase[1].desc)
			assert.Exactly(t, tt.want, tt.orig)
		})
	}
}

func Test_timeSeriesSignature(t *testing.T) {
	tests := []struct {
		name string
		lbs  []prompb.Label
		desc otlp.MetricDescriptor_Type
		want string
	}{
		{
			"int64_signature",
			promLbs1,
			otlp.MetricDescriptor_INT64,
			typeInt64 + lb1Sig,
		},
		{
			"histogram_signature",
			promLbs2,
			otlp.MetricDescriptor_HISTOGRAM,
			typeHistogram + lb2Sig,
		},
		{
			"unordered_signature",
			getPromLabels(label22, value22, label21, value21),
			otlp.MetricDescriptor_HISTOGRAM,
			typeHistogram + lb2Sig,
		},
		// descriptor type cannot be nil, as checked by validateMetrics
		{
			"nil_case",
			nil,
			otlp.MetricDescriptor_HISTOGRAM,
			typeHistogram,
		},
	}

	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.EqualValues(t, tt.want, timeSeriesSignature(tt.desc, &tt.lbs))
		})
	}
}

// Labels should be sanitized; label in extra overrides label in labels if collision happens
// Labels are not sorted
func Test_createLabelSet(t *testing.T) {
	tests := []struct {
		name   string
		orig   []*common.StringKeyValue
		extras []string
		want   []prompb.Label
	}{
		{
			"labels_clean",
			lbs1,
			[]string{label31, value31, label32, value32},
			getPromLabels(label11, value11, label12, value12, label31, value31, label32, value32),
		},
		{
			"labels_duplicate_in_extras",
			lbs1,
			[]string{label11, value31},
			getPromLabels(label11, value31, label12, value12),
		},
		{
			"labels_dirty",
			lbs1Dirty,
			[]string{label31 + dirty1, value31, label32, value32},
			getPromLabels(label11+"_", value11, "key_"+label12, value12, label31+"_", value31, label32, value32),
		},
		{
			"no_extras_case",
			nil,
			[]string{label31, value31, label32, value32},
			getPromLabels(label31, value31, label32, value32),
		},
	}
	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ElementsMatch(t, tt.want, createLabelSet(tt.orig, tt.extras...))
		})
	}
}

func Test_handleScalarMetric(t *testing.T) {
	sameTs := map[string]*prompb.TimeSeries{
		typeMonotonicInt64 + "-name-same_ts_int_points_total" + lb1Sig: getTimeSeries(getPromLabels(label11, value11, label12, value12, "name", "same_ts_int_points_total"),
			getSample(float64(intVal1), time1),
			getSample(float64(intVal2), time1)),
	}
	differentTs := map[string]*prompb.TimeSeries{
		typeMonotonicInt64 + "-name-different_ts_int_points_total" + lb1Sig: getTimeSeries(getPromLabels(label11, value11, label12, value12, "name", "different_ts_int_points_total"),
			getSample(float64(intVal1), time1)),
		typeMonotonicInt64 + "-name-different_ts_int_points_total" + lb2Sig: getTimeSeries(getPromLabels(label21, value21, label22, value22, "name", "different_ts_int_points_total"),
			getSample(float64(intVal1), time2)),
	}
	tests := []struct {
		name        string
		m           *otlp.Metric
		returnError bool
		want        map[string]*prompb.TimeSeries
	}{
		{
			"invalid_nil_array",
			&otlp.Metric{
				MetricDescriptor:    getDescriptor("invalid_nil_array", monotonicInt64, validCombinations),
				Int64DataPoints:     nil,
				DoubleDataPoints:    nil,
				HistogramDataPoints: nil,
				SummaryDataPoints:   nil,
			},
			true,
			map[string]*prompb.TimeSeries{},
		},
		{
			"same_ts_int_points",
			&otlp.Metric{
				MetricDescriptor: getDescriptor("same_ts_int_points", monotonicInt64, validCombinations),
				Int64DataPoints: []*otlp.Int64DataPoint{
					getIntDataPoint(lbs1, intVal1, time1),
					getIntDataPoint(lbs1, intVal2, time1),
				},
				DoubleDataPoints:    nil,
				HistogramDataPoints: nil,
				SummaryDataPoints:   nil,
			},
			false,
			sameTs,
		},
		{
			"different_ts_int_points",
			&otlp.Metric{
				MetricDescriptor: getDescriptor("different_ts_int_points", monotonicInt64, validCombinations),
				Int64DataPoints: []*otlp.Int64DataPoint{
					getIntDataPoint(lbs1, intVal1, time1),
					getIntDataPoint(lbs2, intVal1, time2),
				},
				DoubleDataPoints:    nil,
				HistogramDataPoints: nil,
				SummaryDataPoints:   nil,
			},
			false,
			differentTs,
		},
	}
	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tsMap := map[string]*prompb.TimeSeries{}
			ce := &cortexExporter{}
			ok := ce.handleScalarMetric(tsMap, tt.m)
			if tt.returnError {
				assert.Error(t, ok)
				return
			}
			assert.Exactly(t, len(tt.want), len(tsMap))
			for k, v := range tsMap {
				require.NotNil(t, tt.want[k])
				assert.ElementsMatch(t, tt.want[k].Labels, v.Labels)
				assert.ElementsMatch(t, tt.want[k].Samples, v.Samples)
			}
		})
	}
}