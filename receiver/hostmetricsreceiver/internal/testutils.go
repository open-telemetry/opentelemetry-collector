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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdatautil"
)

func AssertSingleMetricDataAndGetMetricsSlice(t *testing.T, metrics []pdata.Metrics) pdata.MetricSlice {
	// expect 1 MetricData object
	assert.Equal(t, 1, len(metrics))
	return AssertMetricDataAndGetMetricsSlice(t, metrics[0])
}

func AssertMetricDataAndGetMetricsSlice(t *testing.T, metrics pdata.Metrics) pdata.MetricSlice {
	md := pdatautil.MetricsToInternalMetrics(metrics)

	// expect 1 ResourceMetrics object
	rms := md.ResourceMetrics()
	assert.Equal(t, 1, rms.Len())
	rm := rms.At(0)

	// expect 1 InstrumentationLibraryMetrics object
	ilms := rm.InstrumentationLibraryMetrics()
	assert.Equal(t, 1, ilms.Len())
	return ilms.At(0).Metrics()
}

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

func AssertInt64MetricLabelExists(t *testing.T, metric pdata.Metric, index int, labelName string) {
	_, ok := metric.Int64DataPoints().At(index).LabelsMap().Get(labelName)
	assert.Truef(t, ok, "Missing label %q in metric %q", labelName, metric.MetricDescriptor().Name())
}

func AssertInt64MetricLabelDoesNotExist(t *testing.T, metric pdata.Metric, index int, labelName string) {
	_, ok := metric.Int64DataPoints().At(index).LabelsMap().Get(labelName)
	assert.Falsef(t, ok, "Unexpected label %q in metric %q", labelName, metric.MetricDescriptor().Name())
}
