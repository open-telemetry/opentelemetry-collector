// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package filterexpr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
)

func TestCompileExprError(t *testing.T) {
	_, err := NewMatcher("")
	require.Error(t, err)
}

func TestRunExprError(t *testing.T) {
	matcher, err := NewMatcher("foo")
	require.NoError(t, err)
	matched, _ := matcher.match(env{})
	require.False(t, matched)
}

func TestUnknownDataType(t *testing.T) {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.SetName("my.metric")
	m.SetDataType(-1)
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	assert.False(t, matched)
}

func TestNilIntGauge(t *testing.T) {
	dataType := pdata.MetricDataTypeIntGauge
	testNilValue(t, dataType)
}

func TestNilDoubleGauge(t *testing.T) {
	dataType := pdata.MetricDataTypeDoubleGauge
	testNilValue(t, dataType)
}

func TestNilDoubleSum(t *testing.T) {
	dataType := pdata.MetricDataTypeDoubleSum
	testNilValue(t, dataType)
}

func TestNilIntSum(t *testing.T) {
	dataType := pdata.MetricDataTypeIntSum
	testNilValue(t, dataType)
}

func TestNilIntHistogram(t *testing.T) {
	dataType := pdata.MetricDataTypeIntHistogram
	testNilValue(t, dataType)
}

func TestNilDoubleHistogram(t *testing.T) {
	dataType := pdata.MetricDataTypeDoubleHistogram
	testNilValue(t, dataType)
}

func testNilValue(t *testing.T, dataType pdata.MetricDataType) {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.SetName("my.metric")
	m.SetDataType(dataType)
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	assert.False(t, matched)
}

func TestIntGaugeEmptyDataPoint(t *testing.T) {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.SetName("my.metric")
	m.SetDataType(pdata.MetricDataTypeIntGauge)
	dps := m.IntGauge().DataPoints()
	dps.Resize(1)
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	assert.True(t, matched)
}

func TestDoubleGaugeEmptyDataPoint(t *testing.T) {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.SetName("my.metric")
	m.SetDataType(pdata.MetricDataTypeDoubleGauge)
	dps := m.DoubleGauge().DataPoints()
	dps.Resize(1)
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	assert.True(t, matched)
}

func TestDoubleSumEmptyDataPoint(t *testing.T) {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.SetName("my.metric")
	m.SetDataType(pdata.MetricDataTypeDoubleSum)
	dps := m.DoubleSum().DataPoints()
	dps.Resize(1)
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	assert.True(t, matched)
}

func TestIntSumEmptyDataPoint(t *testing.T) {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.SetName("my.metric")
	m.SetDataType(pdata.MetricDataTypeIntSum)
	dps := m.IntSum().DataPoints()
	dps.Resize(1)
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	assert.True(t, matched)
}

func TestIntHistogramEmptyDataPoint(t *testing.T) {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.SetName("my.metric")
	m.SetDataType(pdata.MetricDataTypeIntHistogram)
	dps := m.IntHistogram().DataPoints()
	dps.Resize(1)
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	assert.True(t, matched)
}

func TestDoubleHistogramEmptyDataPoint(t *testing.T) {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.SetName("my.metric")
	m.SetDataType(pdata.MetricDataTypeDoubleHistogram)
	dps := m.DoubleHistogram().DataPoints()
	dps.Resize(1)
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	assert.True(t, matched)
}

func TestMatchIntGaugeByMetricName(t *testing.T) {
	expression := `MetricName == 'my.metric'`
	assert.True(t, testMatchIntGauge(t, "my.metric", expression, nil))
}

func TestNonMatchIntGaugeByMetricName(t *testing.T) {
	expression := `MetricName == 'my.metric'`
	assert.False(t, testMatchIntGauge(t, "foo.metric", expression, nil))
}

func TestNonMatchIntGaugeDataPointByMetricAndHasLabel(t *testing.T) {
	expression := `MetricName == 'my.metric' && HasLabel("foo")`
	assert.False(t, testMatchIntGauge(t, "foo.metric", expression, nil))
}

func TestMatchIntGaugeDataPointByMetricAndHasLabel(t *testing.T) {
	expression := `MetricName == 'my.metric' && HasLabel("foo")`
	assert.True(t, testMatchIntGauge(t, "my.metric", expression, map[string]string{"foo": ""}))
}

func TestMatchIntGaugeDataPointByMetricAndLabelValue(t *testing.T) {
	expression := `MetricName == 'my.metric' && Label("foo") == "bar"`
	assert.False(t, testMatchIntGauge(t, "my.metric", expression, map[string]string{"foo": ""}))
}

func TestNonMatchIntGaugeDataPointByMetricAndLabelValue(t *testing.T) {
	expression := `MetricName == 'my.metric' && Label("foo") == "bar"`
	assert.False(t, testMatchIntGauge(t, "my.metric", expression, map[string]string{"foo": ""}))
}

func testMatchIntGauge(t *testing.T, metricName, expression string, lbls map[string]string) bool {
	matcher, err := NewMatcher(expression)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.SetName(metricName)
	m.SetDataType(pdata.MetricDataTypeIntGauge)
	dps := m.IntGauge().DataPoints()
	dps.Resize(1)
	pt := dps.At(0)
	if lbls != nil {
		pt.LabelsMap().InitFromMap(lbls)
	}
	match, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	return match
}

func TestMatchIntGaugeDataPointByMetricAndSecondPointLabelValue(t *testing.T) {
	matcher, err := NewMatcher(
		`MetricName == 'my.metric' && Label("baz") == "glarch"`,
	)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.SetName("my.metric")
	m.SetDataType(pdata.MetricDataTypeIntGauge)
	dps := m.IntGauge().DataPoints()
	dps.Resize(2)

	pt1 := dps.At(0)
	pt1.LabelsMap().Insert("foo", "bar")

	pt2 := dps.At(1)
	pt2.LabelsMap().Insert("baz", "glarch")

	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	assert.True(t, matched)
}

func TestMatchDoubleGaugeByMetricName(t *testing.T) {
	assert.True(t, testMatchDoubleGauge(t, "my.metric"))
}

func TestNonMatchDoubleGaugeByMetricName(t *testing.T) {
	assert.False(t, testMatchDoubleGauge(t, "foo.metric"))
}

func testMatchDoubleGauge(t *testing.T, metricName string) bool {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.SetName(metricName)
	m.SetDataType(pdata.MetricDataTypeDoubleGauge)
	dps := m.DoubleGauge().DataPoints()
	pt := pdata.NewDoubleDataPoint()
	dps.Append(pt)
	match, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	return match
}

func TestMatchDoubleSumByMetricName(t *testing.T) {
	assert.True(t, matchDoubleSum(t, "my.metric"))
}

func TestNonMatchDoubleSumByMetricName(t *testing.T) {
	assert.False(t, matchDoubleSum(t, "foo.metric"))
}

func matchDoubleSum(t *testing.T, metricName string) bool {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.SetName(metricName)
	m.SetDataType(pdata.MetricDataTypeDoubleSum)
	dps := m.DoubleSum().DataPoints()
	pt := pdata.NewDoubleDataPoint()
	dps.Append(pt)
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	return matched
}

func TestMatchIntSumByMetricName(t *testing.T) {
	assert.True(t, matchIntSum(t, "my.metric"))
}

func TestNonMatchIntSumByMetricName(t *testing.T) {
	assert.False(t, matchIntSum(t, "foo.metric"))
}

func matchIntSum(t *testing.T, metricName string) bool {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.SetName(metricName)
	m.SetDataType(pdata.MetricDataTypeIntSum)
	dps := m.IntSum().DataPoints()
	pt := pdata.NewIntDataPoint()
	dps.Append(pt)
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	return matched
}

func TestMatchIntHistogramByMetricName(t *testing.T) {
	assert.True(t, matchIntHistogram(t, "my.metric"))
}

func TestNonMatchIntHistogramByMetricName(t *testing.T) {
	assert.False(t, matchIntHistogram(t, "foo.metric"))
}

func matchIntHistogram(t *testing.T, metricName string) bool {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.SetName(metricName)
	m.SetDataType(pdata.MetricDataTypeIntHistogram)
	dps := m.IntHistogram().DataPoints()
	pt := pdata.NewIntHistogramDataPoint()
	dps.Append(pt)
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	return matched
}

func TestMatchDoubleHistogramByMetricName(t *testing.T) {
	assert.True(t, matchDoubleHistogram(t, "my.metric"))
}

func TestNonMatchDoubleHistogramByMetricName(t *testing.T) {
	assert.False(t, matchDoubleHistogram(t, "foo.metric"))
}

func matchDoubleHistogram(t *testing.T, metricName string) bool {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.SetName(metricName)
	m.SetDataType(pdata.MetricDataTypeDoubleHistogram)
	dps := m.DoubleHistogram().DataPoints()
	pt := pdata.NewDoubleHistogramDataPoint()
	dps.Append(pt)
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	return matched
}
