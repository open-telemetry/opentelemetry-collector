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

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
)

func AssertContainsAttribute(t *testing.T, attr pdata.AttributeMap, key string) {
	_, ok := attr.Get(key)
	assert.True(t, ok)
}

func AssertDescriptorEqual(t *testing.T, expected pdata.Metric, actual pdata.Metric) {
	assert.Equal(t, expected.Name(), actual.Name())
	assert.Equal(t, expected.Description(), actual.Description())
	assert.Equal(t, expected.Unit(), actual.Unit())
	assert.Equal(t, expected.DataType(), actual.DataType())
}

func AssertIntSumMetricLabelHasValue(t *testing.T, metric pdata.Metric, index int, labelName string, expectedVal string) {
	val, ok := metric.IntSum().DataPoints().At(index).LabelsMap().Get(labelName)
	assert.Truef(t, ok, "Missing label %q in metric %q", labelName, metric.Name())
	assert.Equal(t, expectedVal, val)
}

func AssertIntGaugeMetricLabelHasValue(t *testing.T, metric pdata.Metric, index int, labelName string, expectedVal string) {
	val, ok := metric.IntGauge().DataPoints().At(index).LabelsMap().Get(labelName)
	assert.Truef(t, ok, "Missing label %q in metric %q", labelName, metric.Name())
	assert.Equal(t, expectedVal, val)
}

func AssertDoubleSumMetricLabelHasValue(t *testing.T, metric pdata.Metric, index int, labelName string, expectedVal string) {
	val, ok := metric.DoubleSum().DataPoints().At(index).LabelsMap().Get(labelName)
	assert.Truef(t, ok, "Missing label %q in metric %q", labelName, metric.Name())
	assert.Equal(t, expectedVal, val)
}

func AssertIntSumMetricLabelExists(t *testing.T, metric pdata.Metric, index int, labelName string) {
	_, ok := metric.IntSum().DataPoints().At(index).LabelsMap().Get(labelName)
	assert.Truef(t, ok, "Missing label %q in metric %q", labelName, metric.Name())
}

func AssertDoubleSumMetricLabelExists(t *testing.T, metric pdata.Metric, index int, labelName string) {
	_, ok := metric.DoubleSum().DataPoints().At(index).LabelsMap().Get(labelName)
	assert.Truef(t, ok, "Missing label %q in metric %q", labelName, metric.Name())
}

func AssertIntSumMetricStartTimeEquals(t *testing.T, metric pdata.Metric, startTime pdata.TimestampUnixNano) {
	idps := metric.IntSum().DataPoints()
	for i := 0; i < idps.Len(); i++ {
		require.Equal(t, startTime, idps.At(i).StartTime())
	}
}

func AssertDoubleSumMetricStartTimeEquals(t *testing.T, metric pdata.Metric, startTime pdata.TimestampUnixNano) {
	ddps := metric.DoubleSum().DataPoints()
	for i := 0; i < ddps.Len(); i++ {
		require.Equal(t, startTime, ddps.At(i).StartTime())
	}
}

func AssertSameTimeStampForAllMetrics(t *testing.T, metrics pdata.MetricSlice) {
	AssertSameTimeStampForMetrics(t, metrics, 0, metrics.Len())
}

func AssertSameTimeStampForMetrics(t *testing.T, metrics pdata.MetricSlice, startIdx, endIdx int) {
	var ts pdata.TimestampUnixNano
	for i := startIdx; i < endIdx; i++ {
		metric := metrics.At(i)

		dt := metric.DataType()
		if dt == pdata.MetricDataTypeIntSum {
			idps := metric.IntSum().DataPoints()
			for j := 0; j < idps.Len(); j++ {
				if ts == 0 {
					ts = idps.At(j).Timestamp()
				}
				require.Equalf(t, ts, idps.At(j).Timestamp(), "metrics contained different end timestamp values")
			}
		}

		if dt == pdata.MetricDataTypeDoubleSum {
			ddps := metric.DoubleSum().DataPoints()
			for j := 0; j < ddps.Len(); j++ {
				if ts == 0 {
					ts = ddps.At(j).Timestamp()
				}
				require.Equalf(t, ts, ddps.At(j).Timestamp(), "metrics contained different end timestamp values")
			}
		}
	}
}
