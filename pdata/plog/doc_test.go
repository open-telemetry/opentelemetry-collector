// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package plog_test

import (
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

func ExampleNewLogs() {
	logs := plog.NewLogs()

	resourceLogs := logs.ResourceLogs().AppendEmpty()

	resourceLogs.Resource().Attributes().PutStr("service.name", "my-service")
	resourceLogs.Resource().Attributes().PutStr("service.version", "1.0.0")
	resourceLogs.Resource().Attributes().PutStr("host.name", "server-01")

	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	scopeLogs.Scope().SetName("my-logger")
	scopeLogs.Scope().SetVersion("1.0.0")

	logRecord := scopeLogs.LogRecords().AppendEmpty()
	logRecord.SetTimestamp(pcommon.Timestamp(1640995200000000000))
	logRecord.SetSeverityNumber(plog.SeverityNumberInfo)
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
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()

	severities := []struct {
		level plog.SeverityNumber
		text  string
		msg   string
	}{
		{plog.SeverityNumberDebug, "DEBUG", "Debug information"},
		{plog.SeverityNumberInfo, "INFO", "Application started"},
		{plog.SeverityNumberWarn, "WARN", "Configuration file not found, using defaults"},
		{plog.SeverityNumberError, "ERROR", "Failed to connect to database"},
		{plog.SeverityNumberFatal, "FATAL", "Critical system failure"},
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
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()

	logRecord1 := scopeLogs.LogRecords().AppendEmpty()
	logRecord1.Body().SetStr("Simple string message")
	logRecord1.SetSeverityNumber(plog.SeverityNumberInfo)

	logRecord2 := scopeLogs.LogRecords().AppendEmpty()
	body := logRecord2.Body().SetEmptyMap()
	body.PutStr("event", "user_action")
	body.PutStr("user_id", "user123")
	body.PutInt("timestamp", 1640995200)
	body.PutBool("success", true)
	logRecord2.SetSeverityNumber(plog.SeverityNumberInfo)

	logRecord3 := scopeLogs.LogRecords().AppendEmpty()
	bodySlice := logRecord3.Body().SetEmptySlice()
	bodySlice.AppendEmpty().SetStr("Step 1: Initialize connection")
	bodySlice.AppendEmpty().SetStr("Step 2: Authenticate user")
	bodySlice.AppendEmpty().SetStr("Step 3: Load configuration")
	logRecord3.SetSeverityNumber(plog.SeverityNumberDebug)

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
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()

	logRecord := scopeLogs.LogRecords().AppendEmpty()
	logRecord.Body().SetStr("Processing request")
	logRecord.SetSeverityNumber(plog.SeverityNumberInfo)

	traceID := pcommon.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	spanID := pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8})

	logRecord.SetTraceID(traceID)
	logRecord.SetSpanID(spanID)
	logRecord.SetFlags(plog.DefaultLogRecordFlags.WithIsSampled(true))

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

func ExampleLogRecord_ObservedTimestamp() {
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()

	logRecord := scopeLogs.LogRecords().AppendEmpty()
	logRecord.Body().SetStr("Log entry with observation time")
	logRecord.SetSeverityNumber(plog.SeverityNumberInfo)

	// Set both original timestamp and observed timestamp
	originalTime := pcommon.Timestamp(1640995200000000000) // 2022-01-01 00:00:00 UTC
	observedTime := pcommon.Timestamp(1640995200500000000) // 2022-01-01 00:00:00.5 UTC

	logRecord.SetTimestamp(originalTime)
	logRecord.SetObservedTimestamp(observedTime)

	fmt.Printf("Original timestamp: %d\n", logRecord.Timestamp())
	fmt.Printf("Observed timestamp: %d\n", logRecord.ObservedTimestamp())
	fmt.Printf("Delay (ns): %d\n", logRecord.ObservedTimestamp()-logRecord.Timestamp())
	// Output:
	// Original timestamp: 1640995200000000000
	// Observed timestamp: 1640995200500000000
	// Delay (ns): 500000000
}

func ExampleLogRecord_EventName() {
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()

	logRecord := scopeLogs.LogRecords().AppendEmpty()
	logRecord.SetEventName("user.login")
	logRecord.Body().SetStr("User authentication event")
	logRecord.SetSeverityNumber(plog.SeverityNumberInfo)

	logRecord.Attributes().PutStr("user.id", "user123")
	logRecord.Attributes().PutStr("session.id", "session456")
	logRecord.Attributes().PutBool("success", true)

	fmt.Printf("Event name: %s\n", logRecord.EventName())
	fmt.Printf("Log body: %s\n", logRecord.Body().Str())
	// Output:
	// Event name: user.login
	// Log body: User authentication event
}

func ExampleLogRecordFlags() {
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()

	logRecord := scopeLogs.LogRecords().AppendEmpty()
	logRecord.Body().SetStr("Log with flags")
	logRecord.SetSeverityNumber(plog.SeverityNumberInfo)

	// Test default flags
	defaultFlags := plog.DefaultLogRecordFlags
	logRecord.SetFlags(defaultFlags)
	fmt.Printf("Default flags IsSampled: %t\n", logRecord.Flags().IsSampled())

	// Test with sampled flag
	flagsWithSampled := defaultFlags.WithIsSampled(true)
	logRecord.SetFlags(flagsWithSampled)
	fmt.Printf("With sampled flag: %t\n", logRecord.Flags().IsSampled())

	// Test removing sampled flag
	flagsWithoutSampled := flagsWithSampled.WithIsSampled(false)
	logRecord.SetFlags(flagsWithoutSampled)
	fmt.Printf("Without sampled flag: %t\n", logRecord.Flags().IsSampled())

	// Output:
	// Default flags IsSampled: false
	// With sampled flag: true
	// Without sampled flag: false
}

func ExampleSeverityNumber() {
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()

	// Test all severity levels
	severityLevels := []struct {
		level plog.SeverityNumber
		name  string
	}{
		{plog.SeverityNumberUnspecified, "Unspecified"},
		{plog.SeverityNumberTrace, "Trace"},
		{plog.SeverityNumberTrace2, "Trace2"},
		{plog.SeverityNumberTrace3, "Trace3"},
		{plog.SeverityNumberTrace4, "Trace4"},
		{plog.SeverityNumberDebug, "Debug"},
		{plog.SeverityNumberDebug2, "Debug2"},
		{plog.SeverityNumberDebug3, "Debug3"},
		{plog.SeverityNumberDebug4, "Debug4"},
		{plog.SeverityNumberInfo, "Info"},
		{plog.SeverityNumberInfo2, "Info2"},
		{plog.SeverityNumberInfo3, "Info3"},
		{plog.SeverityNumberInfo4, "Info4"},
		{plog.SeverityNumberWarn, "Warn"},
		{plog.SeverityNumberWarn2, "Warn2"},
		{plog.SeverityNumberWarn3, "Warn3"},
		{plog.SeverityNumberWarn4, "Warn4"},
		{plog.SeverityNumberError, "Error"},
		{plog.SeverityNumberError2, "Error2"},
		{plog.SeverityNumberError3, "Error3"},
		{plog.SeverityNumberError4, "Error4"},
		{plog.SeverityNumberFatal, "Fatal"},
		{plog.SeverityNumberFatal2, "Fatal2"},
		{plog.SeverityNumberFatal3, "Fatal3"},
		{plog.SeverityNumberFatal4, "Fatal4"},
	}

	for i, s := range severityLevels {
		if i < 5 { // Only create first 5 to keep output manageable
			logRecord := scopeLogs.LogRecords().AppendEmpty()
			logRecord.SetSeverityNumber(s.level)
			logRecord.SetSeverityText(s.name)
			logRecord.Body().SetStr("Log at " + s.name + " level")
		}
	}

	fmt.Printf("Total severity levels tested: %d\n", len(severityLevels))
	fmt.Printf("Created log records: %d\n", scopeLogs.LogRecords().Len())
	fmt.Printf("First severity: %s\n", scopeLogs.LogRecords().At(0).SeverityText())
	fmt.Printf("Last severity: %s\n", scopeLogs.LogRecords().At(4).SeverityText())
	// Output:
	// Total severity levels tested: 25
	// Created log records: 5
	// First severity: Unspecified
	// Last severity: Trace4
}

func ExampleLogRecord_DroppedAttributesCount() {
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()

	logRecord := scopeLogs.LogRecords().AppendEmpty()
	logRecord.Body().SetStr("Log with some attributes dropped")
	logRecord.SetSeverityNumber(plog.SeverityNumberWarn)

	// Add some attributes
	logRecord.Attributes().PutStr("included.attr1", "value1")
	logRecord.Attributes().PutStr("included.attr2", "value2")
	logRecord.Attributes().PutInt("included.count", 42)

	// Set dropped attributes count
	logRecord.SetDroppedAttributesCount(7)

	fmt.Printf("Current attributes: %d\n", logRecord.Attributes().Len())
	fmt.Printf("Dropped attributes: %d\n", logRecord.DroppedAttributesCount())
	fmt.Printf("Total original attributes: %d\n", logRecord.Attributes().Len()+int(logRecord.DroppedAttributesCount()))
	// Output:
	// Current attributes: 3
	// Dropped attributes: 7
	// Total original attributes: 10
}
