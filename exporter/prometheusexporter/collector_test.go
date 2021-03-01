// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package prometheusexporter

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer/pdata"
)

type mockAccumulator struct {
	metrics []pdata.Metric
}

func (a *mockAccumulator) Accumulate(rm pdata.ResourceMetrics) (n int) {
	return 0
}
func (a *mockAccumulator) Collect() []pdata.Metric {

	return a.metrics
}

func TestConvertInvalidDataType(t *testing.T) {
	metric := pdata.NewMetric()
	metric.SetDataType(-100)
	c := collector{
		accumulator: &mockAccumulator{
			[]pdata.Metric{metric},
		},
		logger: zap.NewNop(),
	}

	_, err := c.convertMetric(metric)
	require.Equal(t, errUnknownMetricType, err)

	ch := make(chan prometheus.Metric, 1)
	go func() {
		c.Collect(ch)
		close(ch)
	}()

	j := 0
	for range ch {
		require.Fail(t, "Expected no reported metrics")
		j++
	}
}

func TestConvertInvalidMetric(t *testing.T) {
	for _, mType := range []pdata.MetricDataType{
		pdata.MetricDataTypeDoubleHistogram,
		pdata.MetricDataTypeIntHistogram,
		pdata.MetricDataTypeDoubleSum,
		pdata.MetricDataTypeIntSum,
		pdata.MetricDataTypeDoubleGauge,
		pdata.MetricDataTypeIntGauge,
	} {
		metric := pdata.NewMetric()
		metric.SetDataType(mType)
		switch metric.DataType() {
		case pdata.MetricDataTypeIntGauge:
			metric.IntGauge().DataPoints().Append(pdata.NewIntDataPoint())
		case pdata.MetricDataTypeIntSum:
			metric.IntSum().DataPoints().Append(pdata.NewIntDataPoint())
		case pdata.MetricDataTypeDoubleGauge:
			metric.DoubleGauge().DataPoints().Append(pdata.NewDoubleDataPoint())
		case pdata.MetricDataTypeDoubleSum:
			metric.DoubleSum().DataPoints().Append(pdata.NewDoubleDataPoint())
		case pdata.MetricDataTypeIntHistogram:
			metric.IntHistogram().DataPoints().Append(pdata.NewIntHistogramDataPoint())
		case pdata.MetricDataTypeDoubleHistogram:
			metric.DoubleHistogram().DataPoints().Append(pdata.NewDoubleHistogramDataPoint())
		}
		c := collector{}

		_, err := c.convertMetric(metric)
		require.Error(t, err)
	}
}

func TestCollectMetrics(t *testing.T) {
	tests := []struct {
		name       string
		metric     func(time.Time) pdata.Metric
		metricType prometheus.ValueType
		value      float64
	}{
		{
			name:       "IntGauge",
			metricType: prometheus.GaugeValue,
			value:      42.0,
			metric: func(ts time.Time) (metric pdata.Metric) {
				dp := pdata.NewIntDataPoint()
				dp.SetValue(42)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeIntGauge)
				metric.IntGauge().DataPoints().Append(dp)
				metric.SetDescription("test description")

				return
			},
		},
		{
			name:       "DoubleGauge",
			metricType: prometheus.GaugeValue,
			value:      42.42,
			metric: func(ts time.Time) (metric pdata.Metric) {
				dp := pdata.NewDoubleDataPoint()
				dp.SetValue(42.42)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeDoubleGauge)
				metric.DoubleGauge().DataPoints().Append(dp)
				metric.SetDescription("test description")

				return
			},
		},
		{
			name:       "IntSum",
			metricType: prometheus.GaugeValue,
			value:      42.0,
			metric: func(ts time.Time) (metric pdata.Metric) {
				dp := pdata.NewIntDataPoint()
				dp.SetValue(42)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeIntSum)
				metric.IntSum().DataPoints().Append(dp)
				metric.IntSum().SetIsMonotonic(false)
				metric.IntSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
				metric.SetDescription("test description")

				return
			},
		},
		{
			name:       "DoubleSum",
			metricType: prometheus.GaugeValue,
			value:      42.42,
			metric: func(ts time.Time) (metric pdata.Metric) {
				dp := pdata.NewDoubleDataPoint()
				dp.SetValue(42.42)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeDoubleSum)
				metric.DoubleSum().DataPoints().Append(dp)
				metric.DoubleSum().SetIsMonotonic(false)
				metric.DoubleSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
				metric.SetDescription("test description")

				return
			},
		},
		{
			name:       "MonotonicIntSum",
			metricType: prometheus.CounterValue,
			value:      42.0,
			metric: func(ts time.Time) (metric pdata.Metric) {
				dp := pdata.NewIntDataPoint()
				dp.SetValue(42)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeIntSum)
				metric.IntSum().DataPoints().Append(dp)
				metric.IntSum().SetIsMonotonic(true)
				metric.IntSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
				metric.SetDescription("test description")

				return
			},
		},
		{
			name:       "MonotonicDoubleSum",
			metricType: prometheus.CounterValue,
			value:      42.42,
			metric: func(ts time.Time) (metric pdata.Metric) {
				dp := pdata.NewDoubleDataPoint()
				dp.SetValue(42.42)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeDoubleSum)
				metric.DoubleSum().DataPoints().Append(dp)
				metric.DoubleSum().SetIsMonotonic(true)
				metric.DoubleSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
				metric.SetDescription("test description")

				return
			},
		},
	}

	for _, tt := range tests {
		for _, sendTimestamp := range []bool{true, false} {
			name := tt.name
			if sendTimestamp {
				name += "/WithTimestamp"
			}
			t.Run(name, func(t *testing.T) {
				ts := time.Now()
				metric := tt.metric(ts)
				c := collector{
					namespace: "test_space",
					accumulator: &mockAccumulator{
						[]pdata.Metric{metric},
					},
					sendTimestamps: sendTimestamp,
					logger:         zap.NewNop(),
				}

				ch := make(chan prometheus.Metric, 1)
				go func() {
					c.Collect(ch)
					close(ch)
				}()

				j := 0
				for m := range ch {
					j++
					require.Contains(t, m.Desc().String(), "fqName: \"test_space_test_metric\"")
					require.Contains(t, m.Desc().String(), "variableLabels: [label_1 label_2]")

					pbMetric := io_prometheus_client.Metric{}
					m.Write(&pbMetric)

					labelsKeys := map[string]string{"label_1": "1", "label_2": "2"}
					for _, l := range pbMetric.Label {
						require.Equal(t, labelsKeys[*l.Name], *l.Value)
					}

					if sendTimestamp {
						require.Equal(t, ts.UnixNano()/1e6, *(pbMetric.TimestampMs))
					} else {
						require.Nil(t, pbMetric.TimestampMs)
					}

					switch tt.metricType {
					case prometheus.CounterValue:
						require.Equal(t, tt.value, *pbMetric.Counter.Value)
						require.Nil(t, pbMetric.Gauge)
						require.Nil(t, pbMetric.Histogram)
					case prometheus.GaugeValue:
						require.Equal(t, tt.value, *pbMetric.Gauge.Value)
						require.Nil(t, pbMetric.Counter)
						require.Nil(t, pbMetric.Histogram)
					}
				}
				require.Equal(t, 1, j)
			})
		}
	}
}

func TestAccumulateHistograms(t *testing.T) {
	tests := []struct {
		name   string
		metric func(time.Time) pdata.Metric

		histogramPoints map[float64]uint64
		histogramSum    float64
		histogramCount  uint64
	}{
		{
			name: "IntHistogram",
			histogramPoints: map[float64]uint64{
				1.2:  5,
				10.0: 7,
			},
			histogramSum:   42.0,
			histogramCount: 7,
			metric: func(ts time.Time) (metric pdata.Metric) {
				dp := pdata.NewIntHistogramDataPoint()
				dp.SetBucketCounts([]uint64{5, 2})
				dp.SetCount(7)
				dp.SetExplicitBounds([]float64{1.2, 10.0})
				dp.SetSum(42)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeIntHistogram)
				metric.IntHistogram().DataPoints().Append(dp)
				metric.IntHistogram().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
				metric.SetDescription("test description")

				return
			},
		},
		{
			name: "DoubleHistogram",
			histogramPoints: map[float64]uint64{
				3.5:  5,
				10.0: 7,
			},
			histogramSum:   42.42,
			histogramCount: 7,
			metric: func(ts time.Time) (metric pdata.Metric) {
				dp := pdata.NewDoubleHistogramDataPoint()
				dp.SetBucketCounts([]uint64{5, 2})
				dp.SetCount(7)
				dp.SetExplicitBounds([]float64{3.5, 10.0})
				dp.SetSum(42.42)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeDoubleHistogram)
				metric.DoubleHistogram().DataPoints().Append(dp)
				metric.DoubleHistogram().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
				metric.SetDescription("test description")

				return
			},
		},
	}

	for _, tt := range tests {
		for _, sendTimestamp := range []bool{true, false} {
			name := tt.name
			if sendTimestamp {
				name += "/WithTimestamp"
			}
			t.Run(name, func(t *testing.T) {
				ts := time.Now()
				metric := tt.metric(ts)
				c := collector{
					accumulator: &mockAccumulator{
						[]pdata.Metric{metric},
					},
					sendTimestamps: sendTimestamp,
					logger:         zap.NewNop(),
				}

				ch := make(chan prometheus.Metric, 1)
				go func() {
					c.Collect(ch)
					close(ch)
				}()

				n := 0
				for m := range ch {
					n++
					require.Contains(t, m.Desc().String(), "fqName: \"test_metric\"")
					require.Contains(t, m.Desc().String(), "variableLabels: [label_1 label_2]")

					pbMetric := io_prometheus_client.Metric{}
					m.Write(&pbMetric)

					labelsKeys := map[string]string{"label_1": "1", "label_2": "2"}
					for _, l := range pbMetric.Label {
						require.Equal(t, labelsKeys[*l.Name], *l.Value)
					}

					if sendTimestamp {
						require.Equal(t, ts.UnixNano()/1e6, *(pbMetric.TimestampMs))
					} else {
						require.Nil(t, pbMetric.TimestampMs)
					}

					require.Nil(t, pbMetric.Gauge)
					require.Nil(t, pbMetric.Counter)

					h := *pbMetric.Histogram
					require.Equal(t, tt.histogramCount, h.GetSampleCount())
					require.Equal(t, tt.histogramSum, h.GetSampleSum())
					require.Equal(t, len(tt.histogramPoints), len(h.Bucket))

					for _, b := range h.Bucket {
						require.Equal(t, tt.histogramPoints[(*b).GetUpperBound()], b.GetCumulativeCount())
					}
				}
				require.Equal(t, 1, n)
			})
		}
	}
}
