// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package plog

import (
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func ExampleNewLogs() {
	logs := NewLogs()

	resourceLogs := logs.ResourceLogs().AppendEmpty()

	resourceLogs.Resource().Attributes().PutStr("service.name", "my-service")
	resourceLogs.Resource().Attributes().PutStr("service.version", "1.0.0")
	resourceLogs.Resource().Attributes().PutStr("host.name", "server-01")

	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	scopeLogs.Scope().SetName("my-logger")
	scopeLogs.Scope().SetVersion("1.0.0")

	logRecord := scopeLogs.LogRecords().AppendEmpty()
	logRecord.SetTimestamp(pcommon.Timestamp(1640995200000000000))
	logRecord.SetSeverityNumber(SeverityNumberInfo)
	logRecord.SetSeverityText("INFO")
	logRecord.Body().SetStr("User login successful")

	logRecord.Attributes().PutStr("user.id", "user123")
	logRecord.Attributes().PutStr("session.id", "session456")
	logRecord.Attributes().PutStr("action", "login")

	fmt.Printf("Resource logs count: %d\n", logs.ResourceLogs().Len())
	fmt.Printf("Log records count: %d\n", scopeLogs.LogRecords().Len())
	fmt.Printf("Log message: %s\n", logRecord.Body().Str())
	fmt.Printf("Severity: %s\n", logRecord.SeverityText())
	// Output:
	// Resource logs count: 1
	// Log records count: 1
	// Log message: User login successful
	// Severity: INFO
}

func ExampleLogRecord_SetSeverityNumber() {
	logs := NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()

	severities := []struct {
		level SeverityNumber
		text  string
		msg   string
	}{
		{SeverityNumberDebug, "DEBUG", "Debug information"},
		{SeverityNumberInfo, "INFO", "Application started"},
		{SeverityNumberWarn, "WARN", "Configuration file not found, using defaults"},
		{SeverityNumberError, "ERROR", "Failed to connect to database"},
		{SeverityNumberFatal, "FATAL", "Critical system failure"},
	}

	for _, s := range severities {
		logRecord := scopeLogs.LogRecords().AppendEmpty()
		logRecord.SetSeverityNumber(s.level)
		logRecord.SetSeverityText(s.text)
		logRecord.Body().SetStr(s.msg)
		logRecord.SetTimestamp(pcommon.Timestamp(1640995200000000000))
	}

	fmt.Printf("Total log records: %d\n", scopeLogs.LogRecords().Len())

	first := scopeLogs.LogRecords().At(0)
	last := scopeLogs.LogRecords().At(scopeLogs.LogRecords().Len() - 1)

	fmt.Printf("First log: %s - %s\n", first.SeverityText(), first.Body().Str())
	fmt.Printf("Last log: %s - %s\n", last.SeverityText(), last.Body().Str())
	// Output:
	// Total log records: 5
	// First log: DEBUG - Debug information
	// Last log: FATAL - Critical system failure
}

func ExampleLogRecord_Body() {
	logs := NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()

	logRecord1 := scopeLogs.LogRecords().AppendEmpty()
	logRecord1.Body().SetStr("Simple string message")
	logRecord1.SetSeverityNumber(SeverityNumberInfo)

	logRecord2 := scopeLogs.LogRecords().AppendEmpty()
	body := logRecord2.Body().SetEmptyMap()
	body.PutStr("event", "user_action")
	body.PutStr("user_id", "user123")
	body.PutInt("timestamp", 1640995200)
	body.PutBool("success", true)
	logRecord2.SetSeverityNumber(SeverityNumberInfo)

	logRecord3 := scopeLogs.LogRecords().AppendEmpty()
	bodySlice := logRecord3.Body().SetEmptySlice()
	bodySlice.AppendEmpty().SetStr("Step 1: Initialize connection")
	bodySlice.AppendEmpty().SetStr("Step 2: Authenticate user")
	bodySlice.AppendEmpty().SetStr("Step 3: Load configuration")
	logRecord3.SetSeverityNumber(SeverityNumberDebug)

	fmt.Printf("Log 1 body type: %s\n", logRecord1.Body().Type())
	fmt.Printf("Log 2 body type: %s\n", logRecord2.Body().Type())
	fmt.Printf("Log 3 body type: %s\n", logRecord3.Body().Type())
	fmt.Printf("Log 3 steps count: %d\n", logRecord3.Body().Slice().Len())
	// Output:
	// Log 1 body type: Str
	// Log 2 body type: Map
	// Log 3 body type: Slice
	// Log 3 steps count: 3
}

func ExampleLogRecord_TraceID() {
	logs := NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()

	logRecord := scopeLogs.LogRecords().AppendEmpty()
	logRecord.Body().SetStr("Processing request")
	logRecord.SetSeverityNumber(SeverityNumberInfo)

	traceID := pcommon.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	spanID := pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8})

	logRecord.SetTraceID(traceID)
	logRecord.SetSpanID(spanID)
	logRecord.SetFlags(DefaultLogRecordFlags.WithIsSampled(true))

	fmt.Printf("Log message: %s\n", logRecord.Body().Str())
	fmt.Printf("TraceID: %s\n", logRecord.TraceID())
	fmt.Printf("SpanID: %s\n", logRecord.SpanID())
	fmt.Printf("Is sampled: %t\n", logRecord.Flags().IsSampled())
	// Output:
	// Log message: Processing request
	// TraceID: 0102030405060708090a0b0c0d0e0f10
	// SpanID: 0102030405060708
	// Is sampled: true
}
