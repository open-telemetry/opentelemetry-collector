// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptext // import "go.opentelemetry.io/collector/exporter/debugexporter/internal/otlptext"

import (
	"bytes"
	"fmt"
	"math"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type dataBuffer struct {
	buf bytes.Buffer
}

func (b *dataBuffer) logEntry(format string, a ...any) {
	b.buf.WriteString(fmt.Sprintf(format, a...))
	b.buf.WriteString("\n")
}

func (b *dataBuffer) logAttr(attr string, value any) {
	b.logEntry("    %-15s: %s", attr, value)
}

func (b *dataBuffer) logAttributes(header string, m pcommon.Map) {
	if m.Len() == 0 {
		return
	}

	b.logEntry("%s:", header)
	attrPrefix := "     ->"

	// Add offset to attributes if needed.
	headerParts := strings.Split(header, "->")
	if len(headerParts) > 1 {
		attrPrefix = headerParts[0] + attrPrefix
	}

	m.Range(func(k string, v pcommon.Value) bool {
		b.logEntry("%s %s: %s", attrPrefix, k, valueToString(v))
		return true
	})
}

func (b *dataBuffer) logAttributesWithIndentation(header string, m pcommon.Map, indentVal int) {
	if m.Len() == 0 {
		return
	}

	indent := strings.Repeat(" ", indentVal)

	b.logEntry("%s%s:", indent, header)
	attrPrefix := indent + "     ->"

	// Add offset to attributes if needed.
	headerParts := strings.Split(header, "->")
	if len(headerParts) > 1 {
		attrPrefix = headerParts[0] + attrPrefix
	}

	m.Range(func(k string, v pcommon.Value) bool {
		b.logEntry("%s %s: %s", attrPrefix, k, valueToString(v))
		return true
	})
}

func (b *dataBuffer) logInstrumentationScope(il pcommon.InstrumentationScope) {
	b.logEntry(
		"InstrumentationScope %s %s",
		il.Name(),
		il.Version())
	b.logAttributes("InstrumentationScope attributes", il.Attributes())
}

func (b *dataBuffer) logMetricDescriptor(md pmetric.Metric) {
	b.logEntry("Descriptor:")
	b.logEntry("     -> Name: %s", md.Name())
	b.logEntry("     -> Description: %s", md.Description())
	b.logEntry("     -> Unit: %s", md.Unit())
	b.logEntry("     -> DataType: %s", md.Type().String())
}

func (b *dataBuffer) logMetricDataPoints(m pmetric.Metric) {
	switch m.Type() {
	case pmetric.MetricTypeEmpty:
		return
	case pmetric.MetricTypeGauge:
		b.logNumberDataPoints(m.Gauge().DataPoints())
	case pmetric.MetricTypeSum:
		data := m.Sum()
		b.logEntry("     -> IsMonotonic: %t", data.IsMonotonic())
		b.logEntry("     -> AggregationTemporality: %s", data.AggregationTemporality().String())
		b.logNumberDataPoints(data.DataPoints())
	case pmetric.MetricTypeHistogram:
		data := m.Histogram()
		b.logEntry("     -> AggregationTemporality: %s", data.AggregationTemporality().String())
		b.logHistogramDataPoints(data.DataPoints())
	case pmetric.MetricTypeExponentialHistogram:
		data := m.ExponentialHistogram()
		b.logEntry("     -> AggregationTemporality: %s", data.AggregationTemporality().String())
		b.logExponentialHistogramDataPoints(data.DataPoints())
	case pmetric.MetricTypeSummary:
		data := m.Summary()
		b.logDoubleSummaryDataPoints(data.DataPoints())
	}
}

func (b *dataBuffer) logNumberDataPoints(ps pmetric.NumberDataPointSlice) {
	for i := 0; i < ps.Len(); i++ {
		p := ps.At(i)
		b.logEntry("NumberDataPoints #%d", i)
		b.logDataPointAttributes(p.Attributes())

		b.logEntry("StartTimestamp: %s", p.StartTimestamp())
		b.logEntry("Timestamp: %s", p.Timestamp())
		switch p.ValueType() {
		case pmetric.NumberDataPointValueTypeInt:
			b.logEntry("Value: %d", p.IntValue())
		case pmetric.NumberDataPointValueTypeDouble:
			b.logEntry("Value: %f", p.DoubleValue())
		}

		b.logExemplars("Exemplars", p.Exemplars())
	}
}

func (b *dataBuffer) logHistogramDataPoints(ps pmetric.HistogramDataPointSlice) {
	for i := 0; i < ps.Len(); i++ {
		p := ps.At(i)
		b.logEntry("HistogramDataPoints #%d", i)
		b.logDataPointAttributes(p.Attributes())

		b.logEntry("StartTimestamp: %s", p.StartTimestamp())
		b.logEntry("Timestamp: %s", p.Timestamp())
		b.logEntry("Count: %d", p.Count())

		if p.HasSum() {
			b.logEntry("Sum: %f", p.Sum())
		}

		if p.HasMin() {
			b.logEntry("Min: %f", p.Min())
		}

		if p.HasMax() {
			b.logEntry("Max: %f", p.Max())
		}

		for i := 0; i < p.ExplicitBounds().Len(); i++ {
			b.logEntry("ExplicitBounds #%d: %f", i, p.ExplicitBounds().At(i))
		}

		for j := 0; j < p.BucketCounts().Len(); j++ {
			b.logEntry("Buckets #%d, Count: %d", j, p.BucketCounts().At(j))
		}

		b.logExemplars("Exemplars", p.Exemplars())
	}
}

func (b *dataBuffer) logExponentialHistogramDataPoints(ps pmetric.ExponentialHistogramDataPointSlice) {
	for i := 0; i < ps.Len(); i++ {
		p := ps.At(i)
		b.logEntry("ExponentialHistogramDataPoints #%d", i)
		b.logDataPointAttributes(p.Attributes())

		b.logEntry("StartTimestamp: %s", p.StartTimestamp())
		b.logEntry("Timestamp: %s", p.Timestamp())
		b.logEntry("Count: %d", p.Count())

		if p.HasSum() {
			b.logEntry("Sum: %f", p.Sum())
		}

		if p.HasMin() {
			b.logEntry("Min: %f", p.Min())
		}

		if p.HasMax() {
			b.logEntry("Max: %f", p.Max())
		}

		scale := int(p.Scale())
		factor := math.Ldexp(math.Ln2, -scale)
		// Note: the equation used here, which is
		//   math.Exp(index * factor)
		// reports +Inf as the _lower_ boundary of the bucket nearest
		// infinity, which is incorrect and can be addressed in various
		// ways.  The OTel-Go implementation of this histogram pending
		// in https://github.com/open-telemetry/opentelemetry-go/pull/2393
		// uses a lookup table for the last finite boundary, which can be
		// easily computed using `math/big` (for scales up to 20).

		negB := p.Negative().BucketCounts()
		posB := p.Positive().BucketCounts()

		for i := 0; i < negB.Len(); i++ {
			pos := negB.Len() - i - 1
			index := float64(p.Negative().Offset()) + float64(pos)
			lower := math.Exp(index * factor)
			upper := math.Exp((index + 1) * factor)
			b.logEntry("Bucket [%f, %f), Count: %d", -upper, -lower, negB.At(pos))
		}

		if p.ZeroCount() != 0 {
			b.logEntry("Bucket [0, 0], Count: %d", p.ZeroCount())
		}

		for pos := 0; pos < posB.Len(); pos++ {
			index := float64(p.Positive().Offset()) + float64(pos)
			lower := math.Exp(index * factor)
			upper := math.Exp((index + 1) * factor)
			b.logEntry("Bucket (%f, %f], Count: %d", lower, upper, posB.At(pos))
		}

		b.logExemplars("Exemplars", p.Exemplars())
	}
}

func (b *dataBuffer) logDoubleSummaryDataPoints(ps pmetric.SummaryDataPointSlice) {
	for i := 0; i < ps.Len(); i++ {
		p := ps.At(i)
		b.logEntry("SummaryDataPoints #%d", i)
		b.logDataPointAttributes(p.Attributes())

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

func (b *dataBuffer) logDataPointAttributes(attributes pcommon.Map) {
	b.logAttributes("Data point attributes", attributes)
}

func (b *dataBuffer) logEvents(description string, se ptrace.SpanEventSlice) {
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
		b.logAttributes("     -> Attributes:", e.Attributes())
	}
}

func (b *dataBuffer) logLinks(description string, sl ptrace.SpanLinkSlice) {
	if sl.Len() == 0 {
		return
	}

	b.logEntry("%s:", description)

	for i := 0; i < sl.Len(); i++ {
		l := sl.At(i)
		b.logEntry("SpanLink #%d", i)
		b.logEntry("     -> Trace ID: %s", l.TraceID())
		b.logEntry("     -> ID: %s", l.SpanID())
		b.logEntry("     -> TraceState: %s", l.TraceState().AsRaw())
		b.logEntry("     -> DroppedAttributesCount: %d", l.DroppedAttributesCount())
		b.logAttributes("     -> Attributes:", l.Attributes())
	}
}

func (b *dataBuffer) logExemplars(description string, se pmetric.ExemplarSlice) {
	if se.Len() == 0 {
		return
	}

	b.logEntry("%s:", description)

	for i := 0; i < se.Len(); i++ {
		e := se.At(i)
		b.logEntry("Exemplar #%d", i)
		b.logEntry("     -> Trace ID: %s", e.TraceID())
		b.logEntry("     -> Span ID: %s", e.SpanID())
		b.logEntry("     -> Timestamp: %s", e.Timestamp())
		switch e.ValueType() {
		case pmetric.ExemplarValueTypeInt:
			b.logEntry("     -> Value: %d", e.IntValue())
		case pmetric.ExemplarValueTypeDouble:
			b.logEntry("     -> Value: %f", e.DoubleValue())
		}
		b.logAttributes("     -> FilteredAttributes", e.FilteredAttributes())
	}
}

func (b *dataBuffer) logProfileSamples(ss pprofile.SampleSlice) {
	if ss.Len() == 0 {
		return
	}

	for i := 0; i < ss.Len(); i++ {
		b.logEntry("    Sample #%d", i)
		sample := ss.At(i)

		b.logEntry("        Location index: %d", sample.LocationIndex().AsRaw())
		b.logEntry("        Location length: %d", sample.LocationsLength())
		b.logEntry("        Stacktrace ID index: %d", sample.StacktraceIdIndex())
		if lb := sample.Label().Len(); lb > 0 {
			for j := 0; j < lb; j++ {
				b.logEntry("        Label #%d", j)
				b.logEntry("             -> Key: %d", sample.Label().At(j).Key())
				b.logEntry("             -> Str: %d", sample.Label().At(j).Str())
				b.logEntry("             -> Num: %d", sample.Label().At(j).Num())
				b.logEntry("             -> Num unit: %d", sample.Label().At(j).NumUnit())
			}
		}
		b.logEntry("        Value: %d", sample.Value().AsRaw())
		b.logEntry("        Attributes: %d", sample.Attributes().AsRaw())
		b.logEntry("        Link: %d", sample.Link())
	}
}

func (b *dataBuffer) logProfileMappings(ms pprofile.MappingSlice) {
	if ms.Len() == 0 {
		return
	}

	for i := 0; i < ms.Len(); i++ {
		b.logEntry("    Mapping #%d", i)
		mapping := ms.At(i)

		b.logEntry("        ID: %d", mapping.ID())
		b.logEntry("        Memory start: %d", mapping.MemoryStart())
		b.logEntry("        Memory limit: %d", mapping.MemoryLimit())
		b.logEntry("        File offset: %d", mapping.FileOffset())
		b.logEntry("        File name: %d", mapping.Filename())
		b.logEntry("        Build ID: %d", mapping.BuildID())
		b.logEntry("        Attributes: %d", mapping.Attributes().AsRaw())
		b.logEntry("        Has functions: %t", mapping.HasFunctions())
		b.logEntry("        Has filenames: %t", mapping.HasFilenames())
		b.logEntry("        Has line numbers: %t", mapping.HasLineNumbers())
		b.logEntry("        Has inline frames: %t", mapping.HasInlineFrames())

	}
}

func (b *dataBuffer) logProfileLocations(ls pprofile.LocationSlice) {
	if ls.Len() == 0 {
		return
	}

	for i := 0; i < ls.Len(); i++ {
		b.logEntry("    Location #%d", i)
		location := ls.At(i)

		b.logEntry("        ID: %d", location.ID())
		b.logEntry("        Mapping index: %d", location.MappingIndex())
		b.logEntry("        Address: %d", location.Address())
		if ll := location.Line().Len(); ll > 0 {
			for j := 0; j < ll; j++ {
				b.logEntry("        Line #%d", j)
				line := location.Line().At(j)
				b.logEntry("            Function index: %d", line.FunctionIndex())
				b.logEntry("            Line: %d", line.Line())
				b.logEntry("            Column: %d", line.Column())
			}
		}
		b.logEntry("        Is folded: %t", location.IsFolded())
		b.logEntry("        Type index: %d", location.TypeIndex())
		b.logEntry("        Attributes: %d", location.Attributes().AsRaw())
	}
}

func (b *dataBuffer) logProfileFunctions(fs pprofile.FunctionSlice) {
	if fs.Len() == 0 {
		return
	}

	for i := 0; i < fs.Len(); i++ {
		b.logEntry("    Function #%d", i)
		function := fs.At(i)

		b.logEntry("        ID: %d", function.ID())
		b.logEntry("        Name: %d", function.Name())
		b.logEntry("        System name: %d", function.SystemName())
		b.logEntry("        Filename: %d", function.Filename())
		b.logEntry("        Start line: %d", function.StartLine())
	}
}

func (b *dataBuffer) logStringTable(ss pcommon.StringSlice) {
	if ss.Len() == 0 {
		return
	}

	b.logEntry("    String table:")
	for i := 0; i < ss.Len(); i++ {
		b.logEntry("        %s", ss.At(i))
	}
}

func (b *dataBuffer) logComment(c pcommon.Int64Slice) {
	if c.Len() == 0 {
		return
	}

	b.logEntry("    Comment:")
	for i := 0; i < c.Len(); i++ {
		b.logEntry("        %d", c.At(i))
	}
}

func attributeUnitsToMap(aus pprofile.AttributeUnitSlice) pcommon.Map {
	m := pcommon.NewMap()
	for i := 0; i < aus.Len(); i++ {
		au := aus.At(i)
		m.PutInt("attributeKey", au.AttributeKey())
		m.PutInt("unit", au.Unit())
	}
	return m
}

func linkTableToMap(ls pprofile.LinkSlice) pcommon.Map {
	m := pcommon.NewMap()
	for i := 0; i < ls.Len(); i++ {
		l := ls.At(i)
		m.PutStr("Trace ID", l.TraceID().String())
		m.PutStr("Span ID", l.SpanID().String())
	}
	return m
}

func valueToString(v pcommon.Value) string {
	return fmt.Sprintf("%s(%s)", v.Type().String(), v.AsString())
}
