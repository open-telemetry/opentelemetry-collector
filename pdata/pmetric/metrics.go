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

package pmetric // import "go.opentelemetry.io/collector/pdata/pmetric"

import (
	"go.opentelemetry.io/collector/pdata/internal"
	otlpcollectormetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/metrics/v1"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// Metrics is the top-level struct that is propagated through the metrics pipeline.
// Use NewMetrics to create new instance, zero-initialized instance is not valid for use.
type Metrics internal.Metrics

func newMetrics(orig *otlpcollectormetrics.ExportMetricsServiceRequest, s pcommon.State) Metrics {
	return Metrics(internal.NewMetrics(orig, s))
}

func (ms Metrics) getOrig() *otlpcollectormetrics.ExportMetricsServiceRequest {
	return internal.GetOrigMetrics(internal.Metrics(ms))
}

// NewMetrics creates a new Metrics struct.
func NewMetrics() Metrics {
	orig := &otlpcollectormetrics.ExportMetricsServiceRequest{}
	return Metrics(internal.NewMetrics(orig, pcommon.StateExclusive))
}

// CopyTo copies the Metrics instance overriding the destination.
func (ms Metrics) CopyTo(dest Metrics) {
	ms.ResourceMetrics().CopyTo(dest.MutableResourceMetrics())
}

// MoveTo moves the Metrics instance overriding the destination and
// resetting the current instance to its zero value.
// Deprecated: [1.0.0-rc5] The method can be replaced with a plain assignment.
func (ms Metrics) MoveTo(dest Metrics) {
	*dest.getOrig() = *ms.getOrig()
	*ms.getOrig() = otlpcollectormetrics.ExportMetricsServiceRequest{}
}

// ResourceMetrics returns the ResourceMetricsSlice associated with this Metrics.
func (ms Metrics) ResourceMetrics() ResourceMetricsSlice {
	return newImmutableResourceMetricsSlice(&ms.getOrig().ResourceMetrics)
}

// MutableResourceMetrics returns the MutableResourceMetricsSlice associated with this Metrics object.
// This method should be called at once per ConsumeMetrics call if the slice has to be changed,
// otherwise use ResourceMetrics method.
func (ms Metrics) MutableResourceMetrics() MutableResourceMetricsSlice {
	if internal.GetMetricsState(internal.Metrics(ms)) == pcommon.StateShared {
		rms := NewResourceMetricsSlice()
		ms.ResourceMetrics().CopyTo(rms)
		ms.getOrig().ResourceMetrics = *rms.getOrig()
		internal.SetMetricsState(internal.Metrics(ms), pcommon.StateExclusive)
		return rms
	}
	return newMutableResourceMetricsSlice(&ms.getOrig().ResourceMetrics)
}

// MetricCount calculates the total number of metrics.
func (ms Metrics) MetricCount() int {
	metricCount := 0
	rms := ms.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		ilms := rm.ScopeMetrics()
		for j := 0; j < ilms.Len(); j++ {
			ilm := ilms.At(j)
			metricCount += ilm.Metrics().Len()
		}
	}
	return metricCount
}

// DataPointCount calculates the total number of data points.
func (ms Metrics) DataPointCount() (dataPointCount int) {
	rms := ms.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		ilms := rm.ScopeMetrics()
		for j := 0; j < ilms.Len(); j++ {
			ilm := ilms.At(j)
			ms := ilm.Metrics()
			for k := 0; k < ms.Len(); k++ {
				m := ms.At(k)
				switch m.Type() {
				case MetricTypeGauge:
					dataPointCount += m.Gauge().DataPoints().Len()
				case MetricTypeSum:
					dataPointCount += m.Sum().DataPoints().Len()
				case MetricTypeHistogram:
					dataPointCount += m.Histogram().DataPoints().Len()
				case MetricTypeExponentialHistogram:
					dataPointCount += m.ExponentialHistogram().DataPoints().Len()
				case MetricTypeSummary:
					dataPointCount += m.Summary().DataPoints().Len()
				}
			}
		}
	}
	return
}
