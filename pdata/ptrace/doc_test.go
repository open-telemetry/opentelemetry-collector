// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ptrace_test

import (
	"fmt"
	"strconv"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func ExampleNewTraces() {
	traces := ptrace.NewTraces()

	resourceSpans := traces.ResourceSpans().AppendEmpty()

	resourceSpans.Resource().Attributes().PutStr("service.name", "my-service")
	resourceSpans.Resource().Attributes().PutStr("service.version", "1.0.0")

	scopeSpans := resourceSpans.ScopeSpans().AppendEmpty()
	scopeSpans.Scope().SetName("my-instrumentation-library")
	scopeSpans.Scope().SetVersion("1.0.0")

	span := scopeSpans.Spans().AppendEmpty()
	span.SetName("my-operation")
	span.SetKind(ptrace.SpanKindServer)
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
	traces := ptrace.NewTraces()
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
	traces := ptrace.NewTraces()
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
	traces := ptrace.NewTraces()
	resourceSpans := traces.ResourceSpans().AppendEmpty()
	scopeSpans := resourceSpans.ScopeSpans().AppendEmpty()
	span := scopeSpans.Spans().AppendEmpty()

	span.SetName("failed-operation")

	status := span.Status()
	status.SetCode(ptrace.StatusCodeError)
	status.SetMessage("Connection timeout")

	fmt.Printf("Status code: %s\n", status.Code())
	fmt.Printf("Status message: %s\n", status.Message())
	// Output:
	// Status code: Error
	// Status message: Connection timeout
}

func ExampleSpanKind() {
	traces := ptrace.NewTraces()
	resourceSpans := traces.ResourceSpans().AppendEmpty()
	scopeSpans := resourceSpans.ScopeSpans().AppendEmpty()

	// Different span kinds
	kinds := []ptrace.SpanKind{
		ptrace.SpanKindUnspecified,
		ptrace.SpanKindInternal,
		ptrace.SpanKindServer,
		ptrace.SpanKindClient,
		ptrace.SpanKindProducer,
		ptrace.SpanKindConsumer,
	}

	for i, kind := range kinds {
		span := scopeSpans.Spans().AppendEmpty()
		span.SetName("operation-" + strconv.Itoa(i))
		span.SetKind(kind)
		span.SetStartTimestamp(pcommon.Timestamp(1640995200000000000))
		span.SetEndTimestamp(pcommon.Timestamp(1640995200100000000))
	}

	fmt.Printf("Total spans: %d\n", scopeSpans.Spans().Len())
	fmt.Printf("First span kind: %s\n", scopeSpans.Spans().At(0).Kind())
	fmt.Printf("Server span kind: %s\n", scopeSpans.Spans().At(2).Kind())
	fmt.Printf("Client span kind: %s\n", scopeSpans.Spans().At(3).Kind())
	// Output:
	// Total spans: 6
	// First span kind: Unspecified
	// Server span kind: Server
	// Client span kind: Client
}

func ExampleSpan_Links() {
	traces := ptrace.NewTraces()
	resourceSpans := traces.ResourceSpans().AppendEmpty()
	scopeSpans := resourceSpans.ScopeSpans().AppendEmpty()
	span := scopeSpans.Spans().AppendEmpty()

	span.SetName("memlimit-processor")
	span.SetKind(ptrace.SpanKindInternal)

	// Add links to other spans
	link1 := span.Links().AppendEmpty()
	link1.SetTraceID(pcommon.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))
	link1.SetSpanID(pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}))
	link1.TraceState().FromRaw("rojo=00f067aa0ba902b7,congo=t61rcWkgMzE")
	link1.Attributes().PutStr("link.type", "follows_from")
	link1.SetFlags(0x01)

	link2 := span.Links().AppendEmpty()
	link2.SetTraceID(pcommon.TraceID([16]byte{16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1}))
	link2.SetSpanID(pcommon.SpanID([8]byte{8, 7, 6, 5, 4, 3, 2, 1}))
	link2.Attributes().PutStr("link.type", "child_of")
	link2.SetDroppedAttributesCount(2)

	span.SetDroppedLinksCount(1)

	fmt.Printf("Links count: %d\n", span.Links().Len())
	fmt.Printf("First link trace state: %s\n", link1.TraceState().AsRaw())
	fmt.Printf("First link flags: %d\n", link1.Flags())
	fmt.Printf("Dropped links count: %d\n", span.DroppedLinksCount())
	// Output:
	// Links count: 2
	// First link trace state: rojo=00f067aa0ba902b7,congo=t61rcWkgMzE
	// First link flags: 1
	// Dropped links count: 1
}

func ExampleSpan_TraceState() {
	traces := ptrace.NewTraces()
	resourceSpans := traces.ResourceSpans().AppendEmpty()
	scopeSpans := resourceSpans.ScopeSpans().AppendEmpty()
	span := scopeSpans.Spans().AppendEmpty()

	span.SetName("traced-operation")

	// Set trace state (W3C Trace Context)
	span.TraceState().FromRaw("rojo=00f067aa0ba902b7,congo=t61rcWkgMzE")

	fmt.Printf("Trace state: %s\n", span.TraceState().AsRaw())
	// Output:
	// Trace state: rojo=00f067aa0ba902b7,congo=t61rcWkgMzE
}

func ExampleSpan_Flags() {
	traces := ptrace.NewTraces()
	resourceSpans := traces.ResourceSpans().AppendEmpty()
	scopeSpans := resourceSpans.ScopeSpans().AppendEmpty()
	span := scopeSpans.Spans().AppendEmpty()

	span.SetName("sampled-span")

	// Set trace flags (W3C Trace Context)
	span.SetFlags(0x01) // Sampled flag

	fmt.Printf("Span flags: %d\n", span.Flags())
	// Output:
	// Span flags: 1
}

func ExampleStatusCode() {
	traces := ptrace.NewTraces()
	resourceSpans := traces.ResourceSpans().AppendEmpty()
	scopeSpans := resourceSpans.ScopeSpans().AppendEmpty()

	// Different status codes
	statuses := []struct {
		code ptrace.StatusCode
		msg  string
		name string
	}{
		{ptrace.StatusCodeUnset, "", "unset"},
		{ptrace.StatusCodeOk, "Success", "ok"},
		{ptrace.StatusCodeError, "Internal server error", "error"},
	}

	for _, s := range statuses {
		span := scopeSpans.Spans().AppendEmpty()
		span.SetName("operation-" + s.name)
		status := span.Status()
		status.SetCode(s.code)
		status.SetMessage(s.msg)
	}

	fmt.Printf("Total spans: %d\n", scopeSpans.Spans().Len())
	fmt.Printf("Unset status: %s\n", scopeSpans.Spans().At(0).Status().Code())
	fmt.Printf("Ok status: %s\n", scopeSpans.Spans().At(1).Status().Code())
	fmt.Printf("Error status: %s\n", scopeSpans.Spans().At(2).Status().Code())
	fmt.Printf("Error message: %s\n", scopeSpans.Spans().At(2).Status().Message())
	// Output:
	// Total spans: 3
	// Unset status: Unset
	// Ok status: Ok
	// Error status: Error
	// Error message: Internal server error
}

func ExampleSpan_DroppedAttributesCount() {
	traces := ptrace.NewTraces()
	resourceSpans := traces.ResourceSpans().AppendEmpty()
	scopeSpans := resourceSpans.ScopeSpans().AppendEmpty()
	span := scopeSpans.Spans().AppendEmpty()

	span.SetName("span-with-dropped-data")

	// Add some attributes and events
	span.Attributes().PutStr("key1", "value1")
	span.Attributes().PutStr("key2", "value2")

	event := span.Events().AppendEmpty()
	event.SetName("event1")
	event.SetTimestamp(pcommon.Timestamp(1640995200000000000))

	// Set dropped counts
	span.SetDroppedAttributesCount(5)
	span.SetDroppedEventsCount(3)
	span.SetDroppedLinksCount(2)

	fmt.Printf("Current attributes: %d\n", span.Attributes().Len())
	fmt.Printf("Dropped attributes: %d\n", span.DroppedAttributesCount())
	fmt.Printf("Current events: %d\n", span.Events().Len())
	fmt.Printf("Dropped events: %d\n", span.DroppedEventsCount())
	fmt.Printf("Dropped links: %d\n", span.DroppedLinksCount())
	// Output:
	// Current attributes: 2
	// Dropped attributes: 5
	// Current events: 1
	// Dropped events: 3
	// Dropped links: 2
}
