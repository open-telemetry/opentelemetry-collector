// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ptrace

import (
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func ExampleNewTraces() {
	traces := NewTraces()

	resourceSpans := traces.ResourceSpans().AppendEmpty()

	resourceSpans.Resource().Attributes().PutStr("service.name", "my-service")
	resourceSpans.Resource().Attributes().PutStr("service.version", "1.0.0")

	scopeSpans := resourceSpans.ScopeSpans().AppendEmpty()
	scopeSpans.Scope().SetName("my-instrumentation-library")
	scopeSpans.Scope().SetVersion("1.0.0")

	span := scopeSpans.Spans().AppendEmpty()
	span.SetName("my-operation")
	span.SetKind(SpanKindServer)
	span.SetStartTimestamp(pcommon.Timestamp(1640995200000000000)) // 2022-01-01 00:00:00 UTC
	span.SetEndTimestamp(pcommon.Timestamp(1640995200100000000))   // 2022-01-01 00:00:00.1 UTC

	span.Attributes().PutStr("http.method", "GET")
	span.Attributes().PutStr("http.url", "/api/v1/users")
	span.Attributes().PutInt("http.status_code", 200)

	fmt.Printf("Resource spans count: %d\n", traces.ResourceSpans().Len())
	fmt.Printf("Spans count: %d\n", scopeSpans.Spans().Len())
	fmt.Printf("Span name: %s\n", span.Name())
	// Output:
	// Resource spans count: 1
	// Spans count: 1
	// Span name: my-operation
}

func ExampleSpan_SetTraceID() {
	traces := NewTraces()
	resourceSpans := traces.ResourceSpans().AppendEmpty()
	scopeSpans := resourceSpans.ScopeSpans().AppendEmpty()
	span := scopeSpans.Spans().AppendEmpty()

	traceID := pcommon.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	spanID := pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8})
	parentSpanID := pcommon.SpanID([8]byte{9, 10, 11, 12, 13, 14, 15, 16})

	span.SetTraceID(traceID)
	span.SetSpanID(spanID)
	span.SetParentSpanID(parentSpanID)
	span.SetName("child-operation")

	fmt.Printf("TraceID: %s\n", span.TraceID())
	fmt.Printf("SpanID: %s\n", span.SpanID())
	fmt.Printf("ParentSpanID: %s\n", span.ParentSpanID())
	// Output:
	// TraceID: 0102030405060708090a0b0c0d0e0f10
	// SpanID: 0102030405060708
	// ParentSpanID: 090a0b0c0d0e0f10
}

func ExampleSpan_Events() {
	traces := NewTraces()
	resourceSpans := traces.ResourceSpans().AppendEmpty()
	scopeSpans := resourceSpans.ScopeSpans().AppendEmpty()
	span := scopeSpans.Spans().AppendEmpty()

	span.SetName("database-query")

	event1 := span.Events().AppendEmpty()
	event1.SetName("query.start")
	event1.SetTimestamp(pcommon.Timestamp(1640995200000000000))
	event1.Attributes().PutStr("query", "SELECT * FROM users")

	event2 := span.Events().AppendEmpty()
	event2.SetName("query.end")
	event2.SetTimestamp(pcommon.Timestamp(1640995200050000000))
	event2.Attributes().PutInt("rows_returned", 42)

	fmt.Printf("Events count: %d\n", span.Events().Len())
	fmt.Printf("First event name: %s\n", span.Events().At(0).Name())
	fmt.Printf("Second event name: %s\n", span.Events().At(1).Name())
	// Output:
	// Events count: 2
	// First event name: query.start
	// Second event name: query.end
}

func ExampleSpan_Status() {
	traces := NewTraces()
	resourceSpans := traces.ResourceSpans().AppendEmpty()
	scopeSpans := resourceSpans.ScopeSpans().AppendEmpty()
	span := scopeSpans.Spans().AppendEmpty()

	span.SetName("failed-operation")

	status := span.Status()
	status.SetCode(StatusCodeError)
	status.SetMessage("Connection timeout")

	fmt.Printf("Status code: %s\n", status.Code())
	fmt.Printf("Status message: %s\n", status.Message())
	// Output:
	// Status code: Error
	// Status message: Connection timeout
}
