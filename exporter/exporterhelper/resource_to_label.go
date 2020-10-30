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
	"context"
	"strconv"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
)

var (
	resourceToLabels = &ResourceToLabels{}
)

// ResourceToLabels defines the consumer for converting resource attributes to labels
type ResourceToLabels struct {
	nextMetricsConsumer consumer.MetricsConsumer
}

// NewResourceToLabels creates a MetricsConsumer that converts the resource attributes to metric labels
func NewResourceToLabels() consumer.MetricsConsumer {
	return resourceToLabels
}

// ConsumeMetrics implements the consumer.ConsumeMetrics
func (rtl *ResourceToLabels) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	rms := md.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		resource := rms.At(i).Resource()
		if resource.IsNil() {
			continue
		}
		labelMap := extractLabelsFromResource(&resource)

		ilms := rms.At(i).InstrumentationLibraryMetrics()
		for j := 0; j < ilms.Len(); j++ {
			ilm := ilms.At(j)
			if ilm.IsNil() {
				continue
			}
			metricSlice := ilm.Metrics()
			for k := 0; k < metricSlice.Len(); k++ {
				metric := metricSlice.At(k)
				if metric.IsNil() {
					continue
				}
				addLabelsToMetric(&metric, labelMap)
			}
		}
	}
	return rtl.nextMetricsConsumer.ConsumeMetrics(ctx, md)
}

// ConvertResourceToLabels converts all resource attributes to metric labels
func ConvertResourceToLabels(md pdata.Metrics) {
	rms := md.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		resource := rms.At(i).Resource()
		if resource.IsNil() {
			continue
		}
		labelMap := extractLabelsFromResource(&resource)

		ilms := rms.At(i).InstrumentationLibraryMetrics()
		for j := 0; j < ilms.Len(); j++ {
			ilm := ilms.At(j)
			if ilm.IsNil() {
				continue
			}
			metricSlice := ilm.Metrics()
			for k := 0; k < metricSlice.Len(); k++ {
				metric := metricSlice.At(k)
				if metric.IsNil() {
					continue
				}
				addLabelsToMetric(&metric, labelMap)
			}
		}
	}
}

// extractAttributesFromResource extracts the attributes from a given resource and
// returns them as a StringMap.
// Attribute values can be of multiple types. Only string, int, and boolean values are converted
// to string data type and others are skipped.
func extractLabelsFromResource(resource *pdata.Resource) pdata.StringMap {
	labelMap := pdata.NewStringMap()

	attrMap := resource.Attributes()
	attrMap.ForEach(func(k string, av pdata.AttributeValue) {
		switch av.Type() {
		case pdata.AttributeValueSTRING:
			labelMap.Insert(k, av.StringVal())
		case pdata.AttributeValueBOOL:
			labelMap.Insert(k, strconv.FormatBool(av.BoolVal()))
		case pdata.AttributeValueINT:
			labelMap.Insert(k, strconv.FormatInt(av.IntVal(), 10))
		}
	})
	return labelMap
}

// addLabelsToMetric adds additional labels to the given metric
func addLabelsToMetric(metric *pdata.Metric, labelMap pdata.StringMap) {
	switch metric.DataType() {
	case pdata.MetricDataTypeIntGauge:
		addLabelsToIntDataPoints(metric.IntGauge().DataPoints(), labelMap)
	case pdata.MetricDataTypeDoubleGauge:
		addLabelsToDoubleDataPoints(metric.DoubleGauge().DataPoints(), labelMap)
	case pdata.MetricDataTypeIntSum:
		addLabelsToIntDataPoints(metric.IntSum().DataPoints(), labelMap)
	case pdata.MetricDataTypeDoubleSum:
		addLabelsToDoubleDataPoints(metric.DoubleSum().DataPoints(), labelMap)
	case pdata.MetricDataTypeIntHistogram:
		addLabelsToIntHistogramDataPoints(metric.IntHistogram().DataPoints(), labelMap)
	case pdata.MetricDataTypeDoubleHistogram:
		addLabelsToDoubleHistogramDataPoints(metric.DoubleHistogram().DataPoints(), labelMap)
	}
}

func addLabelsToIntDataPoints(ps pdata.IntDataPointSlice, newLabelMap pdata.StringMap) {
	for i := 0; i < ps.Len(); i++ {
		dataPoint := ps.At(i)
		if dataPoint.IsNil() {
			continue
		}
		newLabelMap.ForEach(func(k, v string) {
			dataPoint.LabelsMap().Upsert(k, v)
		})
	}
}

func addLabelsToDoubleDataPoints(ps pdata.DoubleDataPointSlice, newLabelMap pdata.StringMap) {
	for i := 0; i < ps.Len(); i++ {
		dataPoint := ps.At(i)
		if dataPoint.IsNil() {
			continue
		}
		newLabelMap.ForEach(func(k, v string) {
			dataPoint.LabelsMap().Upsert(k, v)
		})
	}
}

func addLabelsToIntHistogramDataPoints(ps pdata.IntHistogramDataPointSlice, newLabelMap pdata.StringMap) {
	for i := 0; i < ps.Len(); i++ {
		dataPoint := ps.At(i)
		if dataPoint.IsNil() {
			continue
		}
		newLabelMap.ForEach(func(k, v string) {
			dataPoint.LabelsMap().Upsert(k, v)
		})
	}
}

func addLabelsToDoubleHistogramDataPoints(ps pdata.DoubleHistogramDataPointSlice, newLabelMap pdata.StringMap) {
	for i := 0; i < ps.Len(); i++ {
		dataPoint := ps.At(i)
		if dataPoint.IsNil() {
			continue
		}
		newLabelMap.ForEach(func(k, v string) {
			dataPoint.LabelsMap().Upsert(k, v)
		})
	}
}
