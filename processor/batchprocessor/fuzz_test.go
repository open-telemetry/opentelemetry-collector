// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batchprocessor // import "go.opentelemetry.io/collector/processor/batchprocessor"

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor/processortest"
)

const longForm = "Jan 2, 2006 at 3:04pm (MST)"

func FuzzBatchTracesProcessor(f *testing.F) {
	f.Fuzz(func(t *testing.T, spanName,
		spanStartTs,
		spanEndTs,
		spanEvent0Ts,
		spanEvent1Ts,
		spanTraceState,
		spanEvent0Name,
		spanEvent1Name,
		spanAttrKey,
		spanAttrVal,
		spanStatusMessage,
		resAttrKey,
		resAttrVal string,
		spanTraceID,
		spanSpanID []byte,
		spanDroppedEventsCount,
		spanDroppedAttributesCount,
		spanEvent0DroppedAttributesCount,
		spanEvent1DroppedAttributesCount uint32,
		spanStatusCode int32) {
		td := ptrace.NewTraces()
		initResource(td.ResourceSpans().AppendEmpty().Resource(), resAttrKey, resAttrVal)
		ss := td.ResourceSpans().At(0).ScopeSpans().AppendEmpty().Spans()
		ss.EnsureCapacity(1)
		err := fillSpanFuzz(ss.AppendEmpty(),
			spanName,
			spanStartTs,
			spanEndTs,
			spanEvent0Ts,
			spanEvent1Ts,
			spanTraceState,
			spanEvent0Name,
			spanEvent1Name,
			spanAttrKey,
			spanAttrVal,
			spanStatusMessage,
			spanTraceID,
			spanSpanID,
			spanDroppedEventsCount,
			spanDroppedAttributesCount,
			spanEvent0DroppedAttributesCount,
			spanEvent1DroppedAttributesCount,
			spanStatusCode)
		if err != nil {
			return
		}
		tel := setupTestTelemetry()
		sink := new(consumertest.TracesSink)
		cfg := createDefaultConfig().(*Config)
		sendBatchSize := 20
		cfg.SendBatchSize = uint32(sendBatchSize)
		cfg.Timeout = 500 * time.Millisecond
		creationSet := tel.NewSettings()
		creationSet.MetricsLevel = configtelemetry.LevelDetailed
		batcher, err := newBatchTracesProcessor(creationSet, sink, cfg)
		if err != nil {
			panic(err)
		}
		batcher.ConsumeTraces(context.Background(), td)
	})
}

func fillSpanFuzz(span ptrace.Span,
	spanName,
	spanStartTs,
	spanEndTs,
	spanEvent0Ts,
	spanEvent1Ts,
	spanTraceState,
	spanEvent0Name,
	spanEvent1Name,
	spanAttrKey,
	spanAttrVal,
	spanStatusMessage string,
	spanTraceID,
	spanSpanID []byte,
	spanDroppedEventsCount,
	spanDroppedAttributesCount,
	spanEvent0DroppedAttributesCount,
	spanEvent1DroppedAttributesCount uint32,
	spanStatusCode int32) error {
	span.SetName(spanName)
	t1, err := time.Parse(longForm, spanStartTs)
	if err != nil {
		return err
	}
	t2, err := time.Parse(longForm, spanStartTs)
	if err != nil {
		return err
	}
	spanEvent0Timestamp, err := time.Parse(longForm, spanEvent0Ts)
	if err != nil {
		return err
	}
	spanEvent1Timestamp, err := time.Parse(longForm, spanEvent1Ts)
	if err != nil {
		return err
	}
	traceID := [16]byte{}
	for i := 0; i < len(spanTraceID); i++ {
		if i == 16 {
			break
		}
		traceID[i] = spanTraceID[i]
	}
	spanID := [8]byte{}
	for i := 0; i < len(spanSpanID); i++ {
		if i == 16 {
			break
		}
		spanID[i] = spanSpanID[i]
	}
	span.SetStartTimestamp(pcommon.NewTimestampFromTime(t1))
	span.SetEndTimestamp(pcommon.NewTimestampFromTime(t2))
	span.SetDroppedAttributesCount(spanDroppedAttributesCount)
	span.TraceState().FromRaw(spanTraceState)
	span.SetTraceID(traceID)
	span.SetSpanID(spanID)
	evs := span.Events()
	ev0 := evs.AppendEmpty()
	ev0.SetTimestamp(pcommon.NewTimestampFromTime(spanEvent0Timestamp))
	ev0.SetName(spanEvent0Name)
	ev0.Attributes().PutStr(spanAttrKey, spanAttrVal)
	ev0.SetDroppedAttributesCount(spanEvent0DroppedAttributesCount)
	ev1 := evs.AppendEmpty()
	ev1.SetTimestamp(pcommon.NewTimestampFromTime(spanEvent1Timestamp))
	ev1.SetName(spanEvent1Name)
	ev1.SetDroppedAttributesCount(spanEvent1DroppedAttributesCount)
	span.SetDroppedEventsCount(spanDroppedEventsCount)
	status := span.Status()
	status.SetCode(ptrace.StatusCodeError)
	status.SetMessage(spanStatusMessage)
	return nil
}

func initResource(r pcommon.Resource, resAttrKey, resAttrVal string) {
	r.Attributes().PutStr(resAttrKey, resAttrVal)
}

func FuzzConsumeTraces(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		u := &ptrace.JSONUnmarshaler{}
		td, err := u.UnmarshalTraces(data)
		if err != nil {
			return
		}
		sink := new(consumertest.TracesSink)
		cfg := createDefaultConfig().(*Config)
		sendBatchSize := 100
		cfg.SendBatchSize = uint32(sendBatchSize)
		cfg.Timeout = 100 * time.Millisecond

		creationSet := processortest.NewNopSettings()
		creationSet.MetricsLevel = configtelemetry.LevelDetailed
		batcher, err := newBatchTracesProcessor(creationSet, sink, cfg)
		if err != nil {
			panic(err)
		}
		batcher.ConsumeTraces(context.Background(), td)
	})
}

func FuzzConsumeMetrics(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		u := &pmetric.JSONUnmarshaler{}
		td, err := u.UnmarshalMetrics(data)
		if err != nil {
			return
		}
		cfg := Config{
			Timeout:       200 * time.Millisecond,
			SendBatchSize: 50,
		}

		sink := new(consumertest.MetricsSink)
		creationSet := processortest.NewNopSettings()
		creationSet.MetricsLevel = configtelemetry.LevelDetailed
		batcher, err := newBatchMetricsProcessor(creationSet, sink, &cfg)
		if err != nil {
			panic(err)
		}
		batcher.ConsumeMetrics(context.Background(), td)
	})
}

func FuzzConsumeLogs(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		u := &plog.JSONUnmarshaler{}
		td, err := u.UnmarshalLogs(data)
		if err != nil {
			return
		}
		cfg := Config{
			Timeout:       200 * time.Millisecond,
			SendBatchSize: 50,
		}

		sink := new(consumertest.LogsSink)
		creationSet := processortest.NewNopSettings()
		creationSet.MetricsLevel = configtelemetry.LevelDetailed
		batcher, err := newBatchLogsProcessor(creationSet, sink, &cfg)
		if err != nil {
			panic(err)
		}
		batcher.ConsumeLogs(context.Background(), td)
	})
}
