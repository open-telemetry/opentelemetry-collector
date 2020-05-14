// Copyright  OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/pdata"
)

func AssertDescriptorEqual(t *testing.T, expected pdata.MetricDescriptor, actual pdata.MetricDescriptor) {
	assert.Equal(t, expected.Name(), actual.Name())
	assert.Equal(t, expected.Description(), actual.Description())
	assert.Equal(t, expected.Unit(), actual.Unit())
	assert.Equal(t, expected.Type(), actual.Type())
	assert.EqualValues(t, expected.LabelsMap().Sort(), actual.LabelsMap().Sort())
}

func AssertInt64MetricLabelHasValue(t *testing.T, metric pdata.Metric, index int, labelName string, expectedVal string) {
	val, ok := metric.Int64DataPoints().At(index).LabelsMap().Get(labelName)
	assert.Truef(t, ok, "Missing label %q in metric %q", labelName, metric.MetricDescriptor().Name())
	assert.Equal(t, expectedVal, val.Value())
}

func AssertDoubleMetricLabelHasValue(t *testing.T, metric pdata.Metric, index int, labelName string, expectedVal string) {
	val, ok := metric.DoubleDataPoints().At(index).LabelsMap().Get(labelName)
	assert.Truef(t, ok, "Missing label %q in metric %q", labelName, metric.MetricDescriptor().Name())
	assert.Equal(t, expectedVal, val.Value())
}

func AssertInt64MetricLabelExists(t *testing.T, metric pdata.Metric, index int, labelName string) {
	_, ok := metric.Int64DataPoints().At(index).LabelsMap().Get(labelName)
	assert.Truef(t, ok, "Missing label %q in metric %q", labelName, metric.MetricDescriptor().Name())
}

func AssertDoubleMetricLabelExists(t *testing.T, metric pdata.Metric, index int, labelName string) {
	_, ok := metric.DoubleDataPoints().At(index).LabelsMap().Get(labelName)
	assert.Truef(t, ok, "Missing label %q in metric %q", labelName, metric.MetricDescriptor().Name())
}

func AssertInt64MetricLabelDoesNotExist(t *testing.T, metric pdata.Metric, index int, labelName string) {
	_, ok := metric.Int64DataPoints().At(index).LabelsMap().Get(labelName)
	assert.Falsef(t, ok, "Unexpected label %q in metric %q", labelName, metric.MetricDescriptor().Name())
}

func AssertDoubleMetricLabelDoesNotExist(t *testing.T, metric pdata.Metric, index int, labelName string) {
	_, ok := metric.DoubleDataPoints().At(index).LabelsMap().Get(labelName)
	assert.Falsef(t, ok, "Unexpected label %q in metric %q", labelName, metric.MetricDescriptor().Name())
}

func AssertInt64MetricLabelNonEmptyAtLeastOneDataPoint(t *testing.T, metric pdata.Metric, labelName string) {
	idps := metric.Int64DataPoints()
	for i := 0; i < idps.Len(); i++ {
		value, ok := idps.At(i).LabelsMap().Get(labelName)
		if ok && value.Value() != "" {
			return
		}
	}

	assert.Failf(t, "Missing label %q for all datapoints in metric %q", labelName, metric.MetricDescriptor().Name())
}

func AssertInt64MetricLabelHasValuesAllDataPoints(t *testing.T, metric pdata.Metric, labelName string, expectedValues []string) {
	values := getAllValuesForLabel(t, metric.Int64DataPoints(), labelName)

	sort.Strings(values)
	sort.Strings(expectedValues)

	assert.Equalf(t, expectedValues, values, "The set of label values for %q did not match the expected values in metric %q", labelName, metric.MetricDescriptor().Name())
}

func getAllValuesForLabel(t *testing.T, idps pdata.Int64DataPointSlice, labelName string) []string {
	valuesMap := make(map[string]bool, idps.Len())
	for i := 0; i < idps.Len(); i++ {
		value, ok := idps.At(i).LabelsMap().Get(labelName)
		if ok {
			valuesMap[value.Value()] = true
		}
	}

	values := make([]string, 0, len(valuesMap))
	for value := range valuesMap {
		values = append(values, value)
	}
	return values
}
