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

package otlptext

import (
	"fmt"
	"strconv"
	"strings"

	"go.opentelemetry.io/collector/consumer/pdata"
	tracetranslator "go.opentelemetry.io/collector/translator/trace"
)

type dataBuffer struct {
	str strings.Builder
}

func (b *dataBuffer) logEntry(format string, a ...interface{}) {
	b.str.WriteString(fmt.Sprintf(format, a...))
	b.str.WriteString("\n")
}

func (b *dataBuffer) logAttr(label string, value string) {
	b.logEntry("    %-15s: %s", label, value)
}

func (b *dataBuffer) logAttributeMap(label string, am pdata.AttributeMap) {
	if am.Len() == 0 {
		return
	}

	b.logEntry("%s:", label)
	am.Range(func(k string, v pdata.AttributeValue) bool {
		b.logEntry("     -> %s: %s(%s)", k, v.Type().String(), attributeValueToString(v))
		return true
	})
}

func (b *dataBuffer) logStringMap(description string, sm pdata.StringMap) {
	if sm.Len() == 0 {
		return
	}

	b.logEntry("%s:", description)
	sm.Range(func(k string, v string) bool {
		b.logEntry("     -> %s: %s", k, v)
		return true
	})
}

func (b *dataBuffer) logInstrumentationLibrary(il pdata.InstrumentationLibrary) {
	b.logEntry(
		"InstrumentationLibrary %s %s",
		il.Name(),
		il.Version())
}

func (b *dataBuffer) logMetricDescriptor(md pdata.Metric) {
	b.logEntry("Descriptor:")
	b.logEntry("     -> Name: %s", md.Name())
	b.logEntry("     -> Description: %s", md.Description())
	b.logEntry("     -> Unit: %s", md.Unit())
	b.logEntry("     -> DataType: %s", md.DataType().String())
}

func (b *dataBuffer) logMetricDataPoints(m pdata.Metric) {
	switch m.DataType() {
	case pdata.MetricDataTypeNone:
		return
	case pdata.MetricDataTypeIntGauge:
		b.logIntDataPoints(m.IntGauge().DataPoints())
	case pdata.MetricDataTypeDoubleGauge:
		b.logDoubleDataPoints(m.DoubleGauge().DataPoints())
	case pdata.MetricDataTypeIntSum:
		data := m.IntSum()
		b.logEntry("     -> IsMonotonic: %t", data.IsMonotonic())
		b.logEntry("     -> AggregationTemporality: %s", data.AggregationTemporality().String())
		b.logIntDataPoints(data.DataPoints())
	case pdata.MetricDataTypeDoubleSum:
		data := m.DoubleSum()
		b.logEntry("     -> IsMonotonic: %t", data.IsMonotonic())
		b.logEntry("     -> AggregationTemporality: %s", data.AggregationTemporality().String())
		b.logDoubleDataPoints(data.DataPoints())
	case pdata.MetricDataTypeIntHistogram:
		data := m.IntHistogram()
		b.logEntry("     -> AggregationTemporality: %s", data.AggregationTemporality().String())
		b.logIntHistogramDataPoints(data.DataPoints())
	case pdata.MetricDataTypeHistogram:
		data := m.Histogram()
		b.logEntry("     -> AggregationTemporality: %s", data.AggregationTemporality().String())
		b.logDoubleHistogramDataPoints(data.DataPoints())
	case pdata.MetricDataTypeSummary:
		data := m.Summary()
		b.logDoubleSummaryDataPoints(data.DataPoints())
	}
}

func (b *dataBuffer) logIntDataPoints(ps pdata.IntDataPointSlice) {
	for i := 0; i < ps.Len(); i++ {
		p := ps.At(i)
		b.logEntry("IntDataPoints #%d", i)
		b.logDataPointLabels(p.LabelsMap())

		b.logEntry("StartTimestamp: %s", p.StartTimestamp())
		b.logEntry("Timestamp: %s", p.Timestamp())
		b.logEntry("Value: %d", p.Value())
	}
}

func (b *dataBuffer) logDoubleDataPoints(ps pdata.DoubleDataPointSlice) {
	for i := 0; i < ps.Len(); i++ {
		p := ps.At(i)
		b.logEntry("DoubleDataPoints #%d", i)
		b.logDataPointLabels(p.LabelsMap())

		b.logEntry("StartTimestamp: %s", p.StartTimestamp())
		b.logEntry("Timestamp: %s", p.Timestamp())
		b.logEntry("Value: %f", p.Value())
	}
}

func (b *dataBuffer) logDoubleHistogramDataPoints(ps pdata.HistogramDataPointSlice) {
	for i := 0; i < ps.Len(); i++ {
		p := ps.At(i)
		b.logEntry("HistogramDataPoints #%d", i)
		b.logDataPointLabels(p.LabelsMap())

		b.logEntry("StartTimestamp: %s", p.StartTimestamp())
		b.logEntry("Timestamp: %s", p.Timestamp())
		b.logEntry("Count: %d", p.Count())
		b.logEntry("Sum: %f", p.Sum())

		bounds := p.ExplicitBounds()
		if len(bounds) != 0 {
			for i, bound := range bounds {
				b.logEntry("ExplicitBounds #%d: %f", i, bound)
			}
		}

		buckets := p.BucketCounts()
		if len(buckets) != 0 {
			for j, bucket := range buckets {
				b.logEntry("Buckets #%d, Count: %d", j, bucket)
			}
		}
	}
}

func (b *dataBuffer) logIntHistogramDataPoints(ps pdata.IntHistogramDataPointSlice) {
	for i := 0; i < ps.Len(); i++ {
		p := ps.At(i)
		b.logEntry("HistogramDataPoints #%d", i)
		b.logDataPointLabels(p.LabelsMap())

		b.logEntry("StartTimestamp: %s", p.StartTimestamp())
		b.logEntry("Timestamp: %s", p.Timestamp())
		b.logEntry("Count: %d", p.Count())
		b.logEntry("Sum: %d", p.Sum())

		bounds := p.ExplicitBounds()
		if len(bounds) != 0 {
			for i, bound := range bounds {
				b.logEntry("ExplicitBounds #%d: %f", i, bound)
			}
		}

		buckets := p.BucketCounts()
		if len(buckets) != 0 {
			for j, bucket := range buckets {
				b.logEntry("Buckets #%d, Count: %d", j, bucket)
			}
		}
	}
}

func (b *dataBuffer) logDoubleSummaryDataPoints(ps pdata.SummaryDataPointSlice) {
	for i := 0; i < ps.Len(); i++ {
		p := ps.At(i)
		b.logEntry("SummaryDataPoints #%d", i)
		b.logDataPointLabels(p.LabelsMap())

		b.logEntry("StartTimestamp: %s", p.StartTimestamp())
		b.logEntry("Timestamp: %s", p.Timestamp())
		b.logEntry("Count: %d", p.Count())
		b.logEntry("Sum: %f", p.Sum())

		quantiles := p.QuantileValues()
		for i := 0; i < quantiles.Len(); i++ {
			quantile := quantiles.At(i)
			b.logEntry("QuantileValue #%d: Quantile %f, Value %f", i, quantile.Quantile(), quantile.Value())
		}
	}
}

func (b *dataBuffer) logDataPointLabels(labels pdata.StringMap) {
	b.logStringMap("Data point labels", labels)
}

func (b *dataBuffer) logLogRecord(lr pdata.LogRecord) {
	b.logEntry("Timestamp: %s", lr.Timestamp())
	b.logEntry("Severity: %s", lr.SeverityText())
	b.logEntry("ShortName: %s", lr.Name())
	b.logEntry("Body: %s", attributeValueToString(lr.Body()))
	b.logAttributeMap("Attributes", lr.Attributes())
}

func (b *dataBuffer) logEvents(description string, se pdata.SpanEventSlice) {
	if se.Len() == 0 {
		return
	}

	b.logEntry("%s:", description)
	for i := 0; i < se.Len(); i++ {
		e := se.At(i)
		b.logEntry("SpanEvent #%d", i)
		b.logEntry("     -> Name: %s", e.Name())
		b.logEntry("     -> Timestamp: %s", e.Timestamp())
		b.logEntry("     -> DroppedAttributesCount: %d", e.DroppedAttributesCount())

		if e.Attributes().Len() == 0 {
			continue
		}
		b.logEntry("     -> Attributes:")
		e.Attributes().Range(func(k string, v pdata.AttributeValue) bool {
			b.logEntry("         -> %s: %s(%s)", k, v.Type().String(), attributeValueToString(v))
			return true
		})
	}
}

func (b *dataBuffer) logLinks(description string, sl pdata.SpanLinkSlice) {
	if sl.Len() == 0 {
		return
	}

	b.logEntry("%s:", description)

	for i := 0; i < sl.Len(); i++ {
		l := sl.At(i)
		b.logEntry("SpanLink #%d", i)
		b.logEntry("     -> Trace ID: %s", l.TraceID().HexString())
		b.logEntry("     -> ID: %s", l.SpanID().HexString())
		b.logEntry("     -> TraceState: %s", l.TraceState())
		b.logEntry("     -> DroppedAttributesCount: %d", l.DroppedAttributesCount())
		if l.Attributes().Len() == 0 {
			continue
		}
		b.logEntry("     -> Attributes:")
		l.Attributes().Range(func(k string, v pdata.AttributeValue) bool {
			b.logEntry("         -> %s: %s(%s)", k, v.Type().String(), attributeValueToString(v))
			return true
		})
	}
}

func attributeValueToString(av pdata.AttributeValue) string {
	switch av.Type() {
	case pdata.AttributeValueTypeString:
		return av.StringVal()
	case pdata.AttributeValueTypeBool:
		return strconv.FormatBool(av.BoolVal())
	case pdata.AttributeValueTypeDouble:
		return strconv.FormatFloat(av.DoubleVal(), 'f', -1, 64)
	case pdata.AttributeValueTypeInt:
		return strconv.FormatInt(av.IntVal(), 10)
	case pdata.AttributeValueTypeArray:
		return attributeValueArrayToString(av.ArrayVal())
	case pdata.AttributeValueTypeMap:
		return attributeMapToString(av.MapVal())
	default:
		return fmt.Sprintf("<Unknown OpenTelemetry attribute value type %q>", av.Type())
	}
}

func attributeValueArrayToString(av pdata.AnyValueArray) string {
	var b strings.Builder
	b.WriteByte('[')
	for i := 0; i < av.Len(); i++ {
		if i < av.Len()-1 {
			fmt.Fprintf(&b, "%s, ", attributeValueToString(av.At(i)))
		} else {
			b.WriteString(attributeValueToString(av.At(i)))
		}
	}

	b.WriteByte(']')
	return b.String()
}

func attributeMapToString(av pdata.AttributeMap) string {
	var b strings.Builder
	b.WriteString("{\n")

	av.Sort().Range(func(k string, v pdata.AttributeValue) bool {
		fmt.Fprintf(&b, "     -> %s: %s(%s)\n", k, v.Type(), tracetranslator.AttributeValueToString(v))
		return true
	})
	b.WriteByte('}')
	return b.String()
}
