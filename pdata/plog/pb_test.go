// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package plog

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gootlplogs "go.opentelemetry.io/proto/slim/otlp/logs/v1"
	goproto "google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestLogsProtoWireCompatibility(t *testing.T) {
	// This test verifies that OTLP ProtoBufs generated using goproto lib in
	// opentelemetry-proto repository OTLP ProtoBufs generated using gogoproto lib in
	// this repository are wire compatible.

	// Generate Logs as pdata struct.
	td := generateTestLogs()

	// Marshal its underlying ProtoBuf to wire.
	marshaler := &ProtoMarshaler{}
	wire1, err := marshaler.MarshalLogs(td)
	require.NoError(t, err)
	assert.NotNil(t, wire1)

	// Unmarshal from the wire to OTLP Protobuf in goproto's representation.
	var goprotoMessage gootlplogs.LogsData
	err = goproto.Unmarshal(wire1, &goprotoMessage)
	require.NoError(t, err)

	// Marshal to the wire again.
	wire2, err := goproto.Marshal(&goprotoMessage)
	require.NoError(t, err)
	assert.NotNil(t, wire2)

	// Unmarshal from the wire into gogoproto's representation.
	var td2 Logs
	unmarshaler := &ProtoUnmarshaler{}
	td2, err = unmarshaler.UnmarshalLogs(wire2)
	require.NoError(t, err)

	// Now compare that the original and final ProtoBuf messages are the same.
	// This proves that goproto and gogoproto marshaling/unmarshaling are wire compatible.
	assert.Equal(t, td, td2)
}

func TestProtoLogsUnmarshalerError(t *testing.T) {
	p := &ProtoUnmarshaler{}
	_, err := p.UnmarshalLogs([]byte("+$%"))
	assert.Error(t, err)
}

func TestProtoSizer(t *testing.T) {
	marshaler := &ProtoMarshaler{}
	ld := NewLogs()
	ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().SetSeverityText("error")

	size := marshaler.LogsSize(ld)

	bytes, err := marshaler.MarshalLogs(ld)
	require.NoError(t, err)
	assert.Equal(t, len(bytes), size)
}

func TestProtoSizerEmptyLogs(t *testing.T) {
	sizer := &ProtoMarshaler{}
	assert.Equal(t, 0, sizer.LogsSize(NewLogs()))
}

func BenchmarkLogsToProto2k(b *testing.B) {
	marshaler := &ProtoMarshaler{}
	logs := generateBenchmarkLogs(2_000)

	for b.Loop() {
		buf, err := marshaler.MarshalLogs(logs)
		require.NoError(b, err)
		assert.NotEmpty(b, buf)
	}
}

func BenchmarkLogsFromProto2k(b *testing.B) {
	marshaler := &ProtoMarshaler{}
	unmarshaler := &ProtoUnmarshaler{}
	baseLogs := generateBenchmarkLogs(2_000)
	buf, err := marshaler.MarshalLogs(baseLogs)
	require.NoError(b, err)
	assert.NotEmpty(b, buf)

	b.ReportAllocs()
	for b.Loop() {
		logs, err := unmarshaler.UnmarshalLogs(buf)
		require.NoError(b, err)
		assert.Equal(b, baseLogs.ResourceLogs().Len(), logs.ResourceLogs().Len())
	}
}

func generateBenchmarkLogs(logsCount int) Logs {
	endTime := pcommon.NewTimestampFromTime(time.Now())

	md := NewLogs()
	ilm := md.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty()
	ilm.LogRecords().EnsureCapacity(logsCount)
	for range logsCount {
		im := ilm.LogRecords().AppendEmpty()
		im.SetTimestamp(endTime)
	}
	return md
}
