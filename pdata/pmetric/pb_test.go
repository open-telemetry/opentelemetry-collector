// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pmetric

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gootlpmetrics "go.opentelemetry.io/proto/slim/otlp/metrics/v1"
	goproto "google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestMetricsProtoWireCompatibility(t *testing.T) {
	// This test verifies that OTLP ProtoBufs generated using goproto lib in
	// opentelemetry-proto repository OTLP ProtoBufs generated using gogoproto lib in
	// this repository are wire compatible.

	// Generate Metrics as pdata struct.
	td := generateTestMetrics()

	// Marshal its underlying ProtoBuf to wire.
	marshaler := &ProtoMarshaler{}
	wire1, err := marshaler.MarshalMetrics(td)
	require.NoError(t, err)
	assert.NotNil(t, wire1)

	// Unmarshal from the wire to OTLP Protobuf in goproto's representation.
	var goprotoMessage gootlpmetrics.MetricsData
	err = goproto.Unmarshal(wire1, &goprotoMessage)
	require.NoError(t, err)

	// Marshal to the wire again.
	wire2, err := goproto.Marshal(&goprotoMessage)
	require.NoError(t, err)
	assert.NotNil(t, wire2)

	// Unmarshal from the wire into gogoproto's representation.
	var td2 Metrics
	unmarshaler := &ProtoUnmarshaler{}
	td2, err = unmarshaler.UnmarshalMetrics(wire2)
	require.NoError(t, err)

	// Now compare that the original and final ProtoBuf messages are the same.
	// This proves that goproto and gogoproto marshaling/unmarshaling are wire compatible.
	assert.Equal(t, td, td2)
}

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

func BenchmarkMetricsToProto2k(b *testing.B) {
	marshaler := &ProtoMarshaler{}
	metrics := generateBenchmarkMetrics(2_000)

	for b.Loop() {
		buf, err := marshaler.MarshalMetrics(metrics)
		require.NoError(b, err)
		assert.NotEmpty(b, buf)
	}
}

func BenchmarkMetricsFromProto10k(b *testing.B) {
	marshaler := &ProtoMarshaler{}
	unmarshaler := &ProtoUnmarshaler{}
	baseMetrics := generateBenchmarkMetrics(2_000)
	buf, err := marshaler.MarshalMetrics(baseMetrics)
	require.NoError(b, err)
	assert.NotEmpty(b, buf)

	b.ReportAllocs()
	for b.Loop() {
		metrics, err := unmarshaler.UnmarshalMetrics(buf)
		require.NoError(b, err)
		assert.Equal(b, baseMetrics.ResourceMetrics().Len(), metrics.ResourceMetrics().Len())
	}
}

func generateBenchmarkMetrics(metricsCount int) Metrics {
	now := time.Now()
	startTime := pcommon.NewTimestampFromTime(now.Add(-10 * time.Second))
	endTime := pcommon.NewTimestampFromTime(now)

	md := NewMetrics()
	ilm := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty()
	ilm.Metrics().EnsureCapacity(metricsCount)
	for range metricsCount {
		im := ilm.Metrics().AppendEmpty()
		im.SetName("test_name")
		idp := im.SetEmptySum().DataPoints().AppendEmpty()
		idp.SetStartTimestamp(startTime)
		idp.SetTimestamp(endTime)
		idp.SetIntValue(123)
	}
	return md
}
