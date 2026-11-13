// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package plogotlp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gootlpcollectorlogs "go.opentelemetry.io/proto/slim/otlp/collector/logs/v1"
	goproto "google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/internal/otlp"
	"go.opentelemetry.io/collector/pdata/plog"
)

var (
	_ json.Unmarshaler = ExportRequest{}
	_ json.Marshaler   = ExportRequest{}
)

var logsRequestJSON = []byte(`
	{
		"resourceLogs": [
		{
			"resource": {},
			"scopeLogs": [
				{
					"scope": {},
					"logRecords": [
						{
							"body": {
								"stringValue": "test_log_record"
							}
						}
					]
				}
			]
		}
		]
	}`)

func TestRequestToPData(t *testing.T) {
	tr := NewExportRequest()
	assert.Equal(t, 0, tr.Logs().LogRecordCount())
	tr.Logs().ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	assert.Equal(t, 1, tr.Logs().LogRecordCount())
}

func TestRequestJSON(t *testing.T) {
	lr := NewExportRequest()
	require.NoError(t, lr.UnmarshalJSON(logsRequestJSON))
	assert.Equal(t, "test_log_record", lr.Logs().ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Body().AsString())

	got, err := lr.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, strings.Join(strings.Fields(string(logsRequestJSON)), ""), string(got))
}

func TestLogsProtoWireCompatibility(t *testing.T) {
	// This test verifies that OTLP ProtoBufs generated using goproto lib in
	// opentelemetry-proto repository OTLP ProtoBufs generated using gogoproto lib in
	// this repository are wire compatible.

	// Generate Logs as pdata struct.
	ld := NewExportRequestFromLogs(plog.Logs(internal.GenTestLogsWrapper()))

	// Marshal its underlying ProtoBuf to wire.
	wire1, err := ld.MarshalProto()
	require.NoError(t, err)
	assert.NotNil(t, wire1)

	// Unmarshal from the wire to OTLP Protobuf in goproto's representation.
	var goprotoMessage gootlpcollectorlogs.ExportLogsServiceRequest
	err = goproto.Unmarshal(wire1, &goprotoMessage)
	require.NoError(t, err)

	// Marshal to the wire again.
	wire2, err := goproto.Marshal(&goprotoMessage)
	require.NoError(t, err)
	assert.NotNil(t, wire2)

	// Unmarshal from the wire into gogoproto's representation.
	ld2 := NewExportRequest()
	err = ld2.UnmarshalProto(wire2)
	require.NoError(t, err)

	// Now compare that the original and final ProtoBuf messages are the same.
	// This proves that goproto and gogoproto marshaling/unmarshaling are wire compatible.
	// Migration logic will run, so run it on the original message as well.
	otlp.MigrateLogs(ld.orig.ResourceLogs)
	assert.Equal(t, ld, ld2)
}

func TestSanitizeUTF8(t *testing.T) {
	ld := plog.NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()
	rl.Resource().Attributes().PutStr("bad\xff", "bad\xff")
	sl := rl.ScopeLogs().AppendEmpty()
	sl.Scope().SetName("scope\xff")
	lr := sl.LogRecords().AppendEmpty()
	lr.Body().SetStr("body\xff")
	lr.SetSeverityText("INFO\xff")
	lr.Attributes().PutStr("ok", "ok")
	lr.Attributes().PutEmptyMap("nested").PutStr("k\xff", "v\xff")

	NewExportRequestFromLogs(ld).SanitizeUTF8()

	expected := plog.NewLogs()
	erl := expected.ResourceLogs().AppendEmpty()
	erl.Resource().Attributes().PutStr("bad�", "bad�")
	esl := erl.ScopeLogs().AppendEmpty()
	esl.Scope().SetName("scope�")
	elr := esl.LogRecords().AppendEmpty()
	elr.Body().SetStr("body�")
	elr.SetSeverityText("INFO�")
	elr.Attributes().PutStr("ok", "ok")
	elr.Attributes().PutEmptyMap("nested").PutStr("k�", "v�")
	assert.Equal(t, expected, ld)
}

func TestSanitizeUTF8ReadOnly(t *testing.T) {
	ld := plog.NewLogs()
	ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr("bad\xff")
	ld.MarkReadOnly()
	assert.Panics(t, func() { NewExportRequestFromLogs(ld).SanitizeUTF8() })
}
