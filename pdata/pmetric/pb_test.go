// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pmetric

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gootlpcollectormetrics "go.opentelemetry.io/proto/slim/otlp/collector/metrics/v1"
	"google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpcollectormetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/metrics/v1"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestProtoMetricsUnmarshalerError(t *testing.T) {
	p := &ProtoUnmarshaler{}
	_, err := p.UnmarshalMetrics([]byte("+$%"))
	assert.Error(t, err)
}

func TestProtoSizer(t *testing.T) {
	marshaler := &ProtoMarshaler{}
	md := NewMetrics()
	md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetName("foo")

	size := marshaler.MetricsSize(md)

	bytes, err := marshaler.MarshalMetrics(md)
	require.NoError(t, err)
	assert.Equal(t, len(bytes), size)
}

func TestProtoSizerEmptyMetrics(t *testing.T) {
	sizer := &ProtoMarshaler{}
	assert.Equal(t, 0, sizer.MetricsSize(NewMetrics()))
}

func BenchmarkMetricsProtoMarshal(b *testing.B) {
	metrics := generateBenchmarkMetrics(128)
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		buf, err := metrics.getOrig().Marshal()
		if err != nil {
			b.Fatal(err)
		}
		if len(buf) == 0 {
			b.Fatal("empty buf")
		}
	}
}

func BenchmarkMetricsProtoMarshalNew(b *testing.B) {
	metrics := generateBenchmarkMetrics(128)
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		orig := metrics.getOrig()
		size := internal.SizeProtoOrigExportMetricsServiceRequest(orig)
		buf := make([]byte, size)
		if internal.MarshalProtoOrigExportMetricsServiceRequest(orig, buf) != size {
			b.Fatal("unexpected size")
		}
	}
}

func BenchmarkMetricsProtoMarshalGoProtobuf(b *testing.B) {
	orig := &gootlpcollectormetrics.ExportMetricsServiceRequest{}
	{
		metrics := generateBenchmarkMetrics(128)
		buf, err := metrics.getOrig().Marshal()
		require.NoError(b, err)
		require.NoError(b, proto.Unmarshal(buf, orig))
	}
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		buf, err := proto.Marshal(orig)
		if err != nil {
			b.Fatal(err)
		}
		if len(buf) == 0 {
			b.Fatal("empty buf")
		}
	}
}

func BenchmarkMetricsProtoUnmarshal(b *testing.B) {
	marshaler := &ProtoMarshaler{}
	baseMetrics := generateBenchmarkMetrics(128)
	buf, err := marshaler.MarshalMetrics(baseMetrics)
	require.NoError(b, err)
	assert.NotEmpty(b, buf)
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		orig := &otlpcollectormetrics.ExportMetricsServiceRequest{}
		if err = orig.Unmarshal(buf); err != nil {
			b.Fatal(err)
		}
		if len(orig.ResourceMetrics) != baseMetrics.ResourceMetrics().Len() {
			b.Fatal("unexpected number of resource metrics")
		}
	}
}

func BenchmarkMetricsProtoUnmarshalNew(b *testing.B) {
	marshaler := &ProtoMarshaler{}
	baseMetrics := generateBenchmarkMetrics(128)
	buf, err := marshaler.MarshalMetrics(baseMetrics)
	require.NoError(b, err)
	assert.NotEmpty(b, buf)
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		orig := &otlpcollectormetrics.ExportMetricsServiceRequest{}
		if err = internal.UnmarshalProtoOrigExportMetricsServiceRequest(orig, buf); err != nil {
			b.Fatal(err)
		}
		if len(orig.ResourceMetrics) != baseMetrics.ResourceMetrics().Len() {
			b.Fatal("unexpected number of resource metrics")
		}
	}
}

func BenchmarkMetricsProtoUnmarshalGoProtobuf(b *testing.B) {
	marshaler := &ProtoMarshaler{}
	baseMetrics := generateBenchmarkMetrics(128)
	buf, err := marshaler.MarshalMetrics(baseMetrics)
	require.NoError(b, err)
	assert.NotEmpty(b, buf)
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		orig := &gootlpcollectormetrics.ExportMetricsServiceRequest{}
		if err = proto.Unmarshal(buf, orig); err != nil {
			b.Fatal(err)
		}
		if len(orig.ResourceMetrics) != baseMetrics.ResourceMetrics().Len() {
			b.Fatal("unexpected number of resource metrics")
		}
	}
}

func generateBenchmarkMetrics(metricsCount int) Metrics {
	now := time.Now()
	startTime := pcommon.NewTimestampFromTime(now.Add(-10 * time.Second))
	endTime := pcommon.NewTimestampFromTime(now)

	md := NewMetrics()
	ilm := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty()
	ilm.Metrics().EnsureCapacity(metricsCount)
	for i := 0; i < metricsCount; i++ {
		im := ilm.Metrics().AppendEmpty()
		im.SetName("test_name")
		idp := im.SetEmptySum().DataPoints().AppendEmpty()
		idp.SetStartTimestamp(startTime)
		idp.SetTimestamp(endTime)
		idp.SetIntValue(123)
	}
	return md
}
