// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testdata

import (
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

var (
	spanStartTimestamp = pcommon.NewTimestampFromTime(time.Date(2020, 2, 11, 20, 26, 12, 321, time.UTC))
	spanEventTimestamp = pcommon.NewTimestampFromTime(time.Date(2020, 2, 11, 20, 26, 13, 123, time.UTC))
	spanEndTimestamp   = pcommon.NewTimestampFromTime(time.Date(2020, 2, 11, 20, 26, 13, 789, time.UTC))
)

func GenerateTraces(spanCount int) ptrace.Traces {
	td := ptrace.NewTraces()
	initResource(td.ResourceSpans().AppendEmpty().Resource())
	ss := td.ResourceSpans().At(0).ScopeSpans().AppendEmpty().Spans()
	ss.EnsureCapacity(spanCount)
	for i := range spanCount {
		switch i % 2 {
		case 0:
			fillSpanOne(ss.AppendEmpty())
		case 1:
			fillSpanTwo(ss.AppendEmpty())
		}
	}
	return td
}

func fillSpanOne(span ptrace.Span) {
	span.SetName("operationA")
	span.SetStartTimestamp(spanStartTimestamp)
	span.SetEndTimestamp(spanEndTimestamp)
	span.SetDroppedAttributesCount(1)
	span.TraceState().FromRaw("ot=th:0") // 100% sampling
	span.SetTraceID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
	span.SetSpanID([8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18})
	evs := span.Events()
	ev0 := evs.AppendEmpty()
	ev0.SetTimestamp(spanEventTimestamp)
	ev0.SetName("event-with-attr")
	ev0.Attributes().PutStr("span-event-attr", "span-event-attr-val")
	ev0.SetDroppedAttributesCount(2)
	ev1 := evs.AppendEmpty()
	ev1.SetTimestamp(spanEventTimestamp)
	ev1.SetName("event")
	ev1.SetDroppedAttributesCount(2)
	span.SetDroppedEventsCount(1)
	status := span.Status()
	status.SetCode(ptrace.StatusCodeError)
	status.SetMessage("status-cancelled")
}

func fillSpanTwo(span ptrace.Span) {
	span.SetName("operationB")
	span.SetStartTimestamp(spanStartTimestamp)
	span.SetEndTimestamp(spanEndTimestamp)
	link0 := span.Links().AppendEmpty()
	link0.Attributes().PutStr("span-link-attr", "span-link-attr-val")
	link0.SetDroppedAttributesCount(4)
	link1 := span.Links().AppendEmpty()
	link1.SetDroppedAttributesCount(4)
	span.SetDroppedLinksCount(3)
}
