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

	"go.opentelemetry.io/collector/model/pdata"
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

func TestEmptyGauge(t *testing.T) {
	testEmptyValue(t, pdata.MetricDataTypeGauge)
}

func TestEmptySum(t *testing.T) {
	testEmptyValue(t, pdata.MetricDataTypeSum)
}

func TestEmptyHistogram(t *testing.T) {
	testEmptyValue(t, pdata.MetricDataTypeHistogram)
}

func testEmptyValue(t *testing.T, dataType pdata.MetricDataType) {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.SetName("my.metric")
	m.SetDataType(dataType)
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	assert.False(t, matched)
}

func TestGaugeEmptyDataPoint(t *testing.T) {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.SetName("my.metric")
	m.SetDataType(pdata.MetricDataTypeGauge)
	m.Gauge().DataPoints().AppendEmpty()
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	assert.True(t, matched)
}

func TestSumEmptyDataPoint(t *testing.T) {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.SetName("my.metric")
	m.SetDataType(pdata.MetricDataTypeSum)
	m.Sum().DataPoints().AppendEmpty()
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	assert.True(t, matched)
}

func TestHistogramEmptyDataPoint(t *testing.T) {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.SetName("my.metric")
	m.SetDataType(pdata.MetricDataTypeHistogram)
	m.Histogram().DataPoints().AppendEmpty()
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	assert.True(t, matched)
}

func TestMatchIntGaugeDataPointByMetricAndSecondPointLabelValue(t *testing.T) {
	matcher, err := NewMatcher(
		`MetricName == 'my.metric' && Label("baz") == "glarch"`,
	)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.SetName("my.metric")
	m.SetDataType(pdata.MetricDataTypeGauge)
	dps := m.Gauge().DataPoints()

	dps.AppendEmpty().LabelsMap().Insert("foo", "bar")
	dps.AppendEmpty().LabelsMap().Insert("baz", "glarch")

	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	assert.True(t, matched)
}

func TestMatchGaugeByMetricName(t *testing.T) {
	expression := `MetricName == 'my.metric'`
	assert.True(t, testMatchGauge(t, "my.metric", expression, nil))
}

func TestNonMatchGaugeByMetricName(t *testing.T) {
	expression := `MetricName == 'my.metric'`
	assert.False(t, testMatchGauge(t, "foo.metric", expression, nil))
}

func TestNonMatchGaugeDataPointByMetricAndHasLabel(t *testing.T) {
	expression := `MetricName == 'my.metric' && HasLabel("foo")`
	assert.False(t, testMatchGauge(t, "foo.metric", expression, nil))
}

func TestMatchGaugeDataPointByMetricAndHasLabel(t *testing.T) {
	expression := `MetricName == 'my.metric' && HasLabel("foo")`
	assert.True(t, testMatchGauge(t, "my.metric", expression, map[string]string{"foo": ""}))
}

func TestMatchGaugeDataPointByMetricAndLabelValue(t *testing.T) {
	expression := `MetricName == 'my.metric' && Label("foo") == "bar"`
	assert.False(t, testMatchGauge(t, "my.metric", expression, map[string]string{"foo": ""}))
}

func TestNonMatchGaugeDataPointByMetricAndLabelValue(t *testing.T) {
	expression := `MetricName == 'my.metric' && Label("foo") == "bar"`
	assert.False(t, testMatchGauge(t, "my.metric", expression, map[string]string{"foo": ""}))
}

func testMatchGauge(t *testing.T, metricName, expression string, lbls map[string]string) bool {
	matcher, err := NewMatcher(expression)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.SetName(metricName)
	m.SetDataType(pdata.MetricDataTypeGauge)
	dps := m.Gauge().DataPoints()
	pt := dps.AppendEmpty()
	if lbls != nil {
		pt.LabelsMap().InitFromMap(lbls)
	}
	match, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	return match
}

func TestMatchSumByMetricName(t *testing.T) {
	assert.True(t, matchSum(t, "my.metric"))
}

func TestNonMatchSumByMetricName(t *testing.T) {
	assert.False(t, matchSum(t, "foo.metric"))
}

func matchSum(t *testing.T, metricName string) bool {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.SetName(metricName)
	m.SetDataType(pdata.MetricDataTypeSum)
	dps := m.Sum().DataPoints()
	dps.AppendEmpty()
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	return matched
}

func TestMatchHistogramByMetricName(t *testing.T) {
	assert.True(t, matchHistogram(t, "my.metric"))
}

func TestNonMatchHistogramByMetricName(t *testing.T) {
	assert.False(t, matchHistogram(t, "foo.metric"))
}

func matchHistogram(t *testing.T, metricName string) bool {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.SetName(metricName)
	m.SetDataType(pdata.MetricDataTypeHistogram)
	dps := m.Histogram().DataPoints()
	dps.AppendEmpty()
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	return matched
}
