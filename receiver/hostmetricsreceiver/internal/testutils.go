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

	"go.opentelemetry.io/collector/consumer/pdata"
)

func AssertContainsAttribute(t *testing.T, attr pdata.AttributeMap, key string) {
	_, ok := attr.Get(key)
	assert.True(t, ok)
}

func AssertDescriptorEqual(t *testing.T, expected pdata.MetricDescriptor, actual pdata.MetricDescriptor) {
	assert.Equal(t, expected.Name(), actual.Name())
	assert.Equal(t, expected.Description(), actual.Description())
	assert.Equal(t, expected.Unit(), actual.Unit())
	assert.Equal(t, expected.Type(), actual.Type())
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
