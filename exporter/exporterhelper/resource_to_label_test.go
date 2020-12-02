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
package exporterhelper

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/internal/testdata"
)

func TestConvertResourceToLabels(t *testing.T) {
	md := testdata.GenerateMetricsOneMetric()
	assert.NotNil(t, md)

	// Before converting resource to labels
	assert.Equal(t, 1, md.ResourceMetrics().At(0).Resource().Attributes().Len())
	assert.Equal(t, 1, md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).IntSum().DataPoints().At(0).LabelsMap().Len())

	cloneMd := convertResourceToLabels(md)

	// After converting resource to labels
	assert.Equal(t, 1, cloneMd.ResourceMetrics().At(0).Resource().Attributes().Len())
	assert.Equal(t, 2, cloneMd.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).IntSum().DataPoints().At(0).LabelsMap().Len())

	assert.Equal(t, 1, md.ResourceMetrics().At(0).Resource().Attributes().Len())
	assert.Equal(t, 1, md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).IntSum().DataPoints().At(0).LabelsMap().Len())

}

func TestConvertResourceToLabelsAllDataTypesEmptyDataPoint(t *testing.T) {
	md := testdata.GenerateMetricsAllTypesEmptyDataPoint()
	assert.NotNil(t, md)

	// Before converting resource to labels
	assert.Equal(t, 1, md.ResourceMetrics().At(0).Resource().Attributes().Len())
	assert.Equal(t, 0, md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).DoubleGauge().DataPoints().At(0).LabelsMap().Len())
	assert.Equal(t, 0, md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(1).IntGauge().DataPoints().At(0).LabelsMap().Len())
	assert.Equal(t, 0, md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(2).DoubleSum().DataPoints().At(0).LabelsMap().Len())
	assert.Equal(t, 0, md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(3).IntSum().DataPoints().At(0).LabelsMap().Len())
	assert.Equal(t, 0, md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(4).DoubleHistogram().DataPoints().At(0).LabelsMap().Len())
	assert.Equal(t, 0, md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(5).IntHistogram().DataPoints().At(0).LabelsMap().Len())

	cloneMd := convertResourceToLabels(md)

	// After converting resource to labels
	assert.Equal(t, 1, cloneMd.ResourceMetrics().At(0).Resource().Attributes().Len())
	assert.Equal(t, 1, cloneMd.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).DoubleGauge().DataPoints().At(0).LabelsMap().Len())
	assert.Equal(t, 1, cloneMd.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(1).IntGauge().DataPoints().At(0).LabelsMap().Len())
	assert.Equal(t, 1, cloneMd.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(2).DoubleSum().DataPoints().At(0).LabelsMap().Len())
	assert.Equal(t, 1, cloneMd.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(3).IntSum().DataPoints().At(0).LabelsMap().Len())
	assert.Equal(t, 1, cloneMd.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(4).DoubleHistogram().DataPoints().At(0).LabelsMap().Len())
	assert.Equal(t, 1, cloneMd.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(5).IntHistogram().DataPoints().At(0).LabelsMap().Len())

	assert.Equal(t, 1, md.ResourceMetrics().At(0).Resource().Attributes().Len())
	assert.Equal(t, 0, md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).DoubleGauge().DataPoints().At(0).LabelsMap().Len())
	assert.Equal(t, 0, md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(1).IntGauge().DataPoints().At(0).LabelsMap().Len())
	assert.Equal(t, 0, md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(2).DoubleSum().DataPoints().At(0).LabelsMap().Len())
	assert.Equal(t, 0, md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(3).IntSum().DataPoints().At(0).LabelsMap().Len())
	assert.Equal(t, 0, md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(4).DoubleHistogram().DataPoints().At(0).LabelsMap().Len())
	assert.Equal(t, 0, md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(5).IntHistogram().DataPoints().At(0).LabelsMap().Len())

}
