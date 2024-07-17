// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"context"
	"math/rand"
	"sort"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestSplitTracesOneResourceSpans(t *testing.T) {
	inBatch := ptrace.NewTraces()
	fillResourceSpans(inBatch.ResourceSpans().AppendEmpty(), "attr_key", "1")

	outBatches := organizeTraces([]string{"attr_key"}, inBatch)
	require.Len(t, outBatches, 1)
	assert.Equal(t, inBatch, outBatches[0])
}

func TestOriginalResourceSpansUnchanged(t *testing.T) {
	inBatch := ptrace.NewTraces()
	fillResourceSpans(inBatch.ResourceSpans().AppendEmpty(), "attr_key", "1")
	fillResourceSpans(inBatch.ResourceSpans().AppendEmpty(), "attr_key", "1")

	outBatches := organizeTraces([]string{"attr_key"}, inBatch)
	require.Len(t, outBatches, 1)
}

func TestSplitTracesSameResource(t *testing.T) {
	inBatch := ptrace.NewTraces()
	fillResourceSpans(inBatch.ResourceSpans().AppendEmpty(), "same_attr_val", "1")
	fillResourceSpans(inBatch.ResourceSpans().AppendEmpty(), "same_attr_val", "1")
	fillResourceSpans(inBatch.ResourceSpans().AppendEmpty(), "same_attr_val", "1")
	fillResourceSpans(inBatch.ResourceSpans().AppendEmpty(), "same_attr_val", "1")
	expected := ptrace.NewTraces()
	inBatch.CopyTo(expected)

	outBatches := organizeTraces([]string{"same_attr_val"}, inBatch)
	require.Len(t, outBatches, 1)
}

func TestSplitTracesIntoDifferentBatches(t *testing.T) {
	inBatch := ptrace.NewTraces()
	fillResourceSpans(inBatch.ResourceSpans().AppendEmpty(), "attr_key", "1")
	fillResourceSpans(inBatch.ResourceSpans().AppendEmpty(), "attr_key", "2")
	fillResourceSpans(inBatch.ResourceSpans().AppendEmpty(), "attr_key", "3")
	fillResourceSpans(inBatch.ResourceSpans().AppendEmpty(), "attr_key", "4")
	fillResourceSpans(inBatch.ResourceSpans().AppendEmpty(), "attr_key", "1")
	fillResourceSpans(inBatch.ResourceSpans().AppendEmpty(), "attr_key", "2")
	fillResourceSpans(inBatch.ResourceSpans().AppendEmpty(), "attr_key", "3")
	fillResourceSpans(inBatch.ResourceSpans().AppendEmpty(), "attr_key", "4")
	fillResourceSpans(inBatch.ResourceSpans().AppendEmpty(), "diff_attr_key", "1")
	expected := ptrace.NewTraces()
	inBatch.CopyTo(expected)

	outBatches := organizeTraces([]string{"attr_key"}, inBatch)
	require.Len(t, outBatches, 5)
	sortTraces(outBatches, "attr_key")
	assert.Equal(t, newTraces(expected.ResourceSpans().At(8)), outBatches[0])
	assert.Equal(t, newTraces(expected.ResourceSpans().At(0), expected.ResourceSpans().At(4)), outBatches[1])
	assert.Equal(t, newTraces(expected.ResourceSpans().At(1), expected.ResourceSpans().At(5)), outBatches[2])
	assert.Equal(t, newTraces(expected.ResourceSpans().At(2), expected.ResourceSpans().At(6)), outBatches[3])
	assert.Equal(t, newTraces(expected.ResourceSpans().At(3), expected.ResourceSpans().At(7)), outBatches[4])
}

func TestSplitTracesIntoDifferentBatchesWithMultipleKeys(t *testing.T) {
	inBatch := ptrace.NewTraces()
	fillResourceSpans(inBatch.ResourceSpans().AppendEmpty(), "attr_key", "1", "attr_key2", "1")
	fillResourceSpans(inBatch.ResourceSpans().AppendEmpty(), "attr_key", "2", "attr_key2", "2")
	fillResourceSpans(inBatch.ResourceSpans().AppendEmpty(), "attr_key", "3", "attr_key2", "3")
	fillResourceSpans(inBatch.ResourceSpans().AppendEmpty(), "attr_key", "4", "attr_key2", "4")
	fillResourceSpans(inBatch.ResourceSpans().AppendEmpty(), "attr_key", "1", "attr_key2", "1")
	fillResourceSpans(inBatch.ResourceSpans().AppendEmpty(), "attr_key", "2", "attr_key2", "2")
	fillResourceSpans(inBatch.ResourceSpans().AppendEmpty(), "attr_key", "3", "attr_key2", "3")
	fillResourceSpans(inBatch.ResourceSpans().AppendEmpty(), "attr_key", "4", "attr_key2", "4")
	fillResourceSpans(inBatch.ResourceSpans().AppendEmpty(), "attr_key", "5", "attr_key2", "6")
	fillResourceSpans(inBatch.ResourceSpans().AppendEmpty(), "diff_attr_key", "1")
	expected := ptrace.NewTraces()
	inBatch.CopyTo(expected)

	outBatches := organizeTraces([]string{"attr_key", "attr_key2"}, inBatch)
	sortTraces(outBatches, "attr_key")
	assert.Equal(t, newTraces(expected.ResourceSpans().At(9)), outBatches[0])
	assert.Equal(t, newTraces(expected.ResourceSpans().At(0), expected.ResourceSpans().At(4)), outBatches[1])
	assert.Equal(t, newTraces(expected.ResourceSpans().At(1), expected.ResourceSpans().At(5)), outBatches[2])
	assert.Equal(t, newTraces(expected.ResourceSpans().At(2), expected.ResourceSpans().At(6)), outBatches[3])
	assert.Equal(t, newTraces(expected.ResourceSpans().At(3), expected.ResourceSpans().At(7)), outBatches[4])
	assert.Equal(t, newTraces(expected.ResourceSpans().At(8)), outBatches[5])
}

func TestSplitMetricsOneResourceMetrics(t *testing.T) {
	inBatch := pmetric.NewMetrics()
	fillResourceMetrics(inBatch.ResourceMetrics().AppendEmpty(), "attr_key", "1")
	expected := pmetric.NewMetrics()
	inBatch.CopyTo(expected)

	outBatches := organizeMetrics([]string{"attr_key"}, inBatch)
	require.Len(t, outBatches, 1)
	assert.Equal(t, expected, outBatches[0])
}

func TestOriginalResourceMetricsUnchanged(t *testing.T) {
	inBatch := pmetric.NewMetrics()
	fillResourceMetrics(inBatch.ResourceMetrics().AppendEmpty(), "attr_key", "1")
	fillResourceMetrics(inBatch.ResourceMetrics().AppendEmpty(), "attr_key", "1")

	outBatches := organizeMetrics([]string{"attr_key"}, inBatch)
	require.Len(t, outBatches, 1)
	assert.Equal(t, inBatch, outBatches[0])
}

func TestSplitMetricsSameResource(t *testing.T) {
	inBatch := pmetric.NewMetrics()
	fillResourceMetrics(inBatch.ResourceMetrics().AppendEmpty(), "same_attr_val", "1")
	fillResourceMetrics(inBatch.ResourceMetrics().AppendEmpty(), "same_attr_val", "1")
	fillResourceMetrics(inBatch.ResourceMetrics().AppendEmpty(), "same_attr_val", "1")
	fillResourceMetrics(inBatch.ResourceMetrics().AppendEmpty(), "same_attr_val", "1")
	expected := pmetric.NewMetrics()
	inBatch.CopyTo(expected)

	outBatches := organizeMetrics([]string{"same_attr_val"}, inBatch)
	require.Len(t, outBatches, 1)
	assert.Equal(t, expected, outBatches[0])
}

func TestSplitMetricsIntoDifferentBatches(t *testing.T) {
	inBatch := pmetric.NewMetrics()
	fillResourceMetrics(inBatch.ResourceMetrics().AppendEmpty(), "attr_key", "1")
	fillResourceMetrics(inBatch.ResourceMetrics().AppendEmpty(), "attr_key", "2")
	fillResourceMetrics(inBatch.ResourceMetrics().AppendEmpty(), "attr_key", "3")
	fillResourceMetrics(inBatch.ResourceMetrics().AppendEmpty(), "attr_key", "4")
	fillResourceMetrics(inBatch.ResourceMetrics().AppendEmpty(), "attr_key", "1")
	fillResourceMetrics(inBatch.ResourceMetrics().AppendEmpty(), "attr_key", "2")
	fillResourceMetrics(inBatch.ResourceMetrics().AppendEmpty(), "attr_key", "3")
	fillResourceMetrics(inBatch.ResourceMetrics().AppendEmpty(), "attr_key", "4")
	fillResourceMetrics(inBatch.ResourceMetrics().AppendEmpty(), "diff_attr_key", "1")
	expected := pmetric.NewMetrics()
	inBatch.CopyTo(expected)

	outBatches := organizeMetrics([]string{"attr_key"}, inBatch)
	require.Len(t, outBatches, 5)
	sortMetrics(outBatches, "attr_key")
	assert.Equal(t, newMetrics(expected.ResourceMetrics().At(8)), outBatches[0])
	assert.Equal(t, newMetrics(expected.ResourceMetrics().At(0), expected.ResourceMetrics().At(4)), outBatches[1])
	assert.Equal(t, newMetrics(expected.ResourceMetrics().At(1), expected.ResourceMetrics().At(5)), outBatches[2])
	assert.Equal(t, newMetrics(expected.ResourceMetrics().At(2), expected.ResourceMetrics().At(6)), outBatches[3])
	assert.Equal(t, newMetrics(expected.ResourceMetrics().At(3), expected.ResourceMetrics().At(7)), outBatches[4])
}

func TestSplitMetricsIntoDifferentBatchesWithMultipleKeys(t *testing.T) {
	inBatch := pmetric.NewMetrics()
	fillResourceMetrics(inBatch.ResourceMetrics().AppendEmpty(), "attr_key", "1", "attr_key2", "1")
	fillResourceMetrics(inBatch.ResourceMetrics().AppendEmpty(), "attr_key", "2", "attr_key2", "2")
	fillResourceMetrics(inBatch.ResourceMetrics().AppendEmpty(), "attr_key", "3", "attr_key2", "3")
	fillResourceMetrics(inBatch.ResourceMetrics().AppendEmpty(), "attr_key", "4", "attr_key2", "4")
	fillResourceMetrics(inBatch.ResourceMetrics().AppendEmpty(), "attr_key", "1", "attr_key2", "1")
	fillResourceMetrics(inBatch.ResourceMetrics().AppendEmpty(), "attr_key", "2", "attr_key2", "2")
	fillResourceMetrics(inBatch.ResourceMetrics().AppendEmpty(), "attr_key", "3", "attr_key2", "3")
	fillResourceMetrics(inBatch.ResourceMetrics().AppendEmpty(), "attr_key", "4", "attr_key2", "4")
	fillResourceMetrics(inBatch.ResourceMetrics().AppendEmpty(), "attr_key", "5", "attr_key2", "6")
	fillResourceMetrics(inBatch.ResourceMetrics().AppendEmpty(), "diff_attr_key", "1")
	expected := pmetric.NewMetrics()
	inBatch.CopyTo(expected)

	outBatches := organizeMetrics([]string{"attr_key", "attr_key2"}, inBatch)
	require.Len(t, outBatches, 6)
	sortMetrics(outBatches, "attr_key")
	assert.Equal(t, newMetrics(expected.ResourceMetrics().At(9)), outBatches[0])
	assert.Equal(t, newMetrics(expected.ResourceMetrics().At(0), expected.ResourceMetrics().At(4)), outBatches[1])
	assert.Equal(t, newMetrics(expected.ResourceMetrics().At(1), expected.ResourceMetrics().At(5)), outBatches[2])
	assert.Equal(t, newMetrics(expected.ResourceMetrics().At(2), expected.ResourceMetrics().At(6)), outBatches[3])
	assert.Equal(t, newMetrics(expected.ResourceMetrics().At(3), expected.ResourceMetrics().At(7)), outBatches[4])
	assert.Equal(t, newMetrics(expected.ResourceMetrics().At(8)), outBatches[5])
}

func TestSplitLogsOneResourceLogs(t *testing.T) {
	inBatch := plog.NewLogs()
	fillResourceLogs(inBatch.ResourceLogs().AppendEmpty(), "attr_key", "1")
	expected := plog.NewLogs()
	inBatch.CopyTo(expected)

	outBatches := organizeLogs([]string{"attr_key"}, inBatch)
	require.Len(t, outBatches, 1)
	assert.Equal(t, expected, outBatches[0])
}

func TestOriginalResourceLogsUnchanged(t *testing.T) {
	inBatch := plog.NewLogs()
	fillResourceLogs(inBatch.ResourceLogs().AppendEmpty(), "attr_key", "1")
	fillResourceLogs(inBatch.ResourceLogs().AppendEmpty(), "attr_key", "1")

	outBatches := organizeLogs([]string{"attr_key"}, inBatch)
	require.Len(t, outBatches, 1)
	assert.Equal(t, inBatch, outBatches[0])
}

func TestSplitLogsSameResource(t *testing.T) {
	inBatch := plog.NewLogs()
	fillResourceLogs(inBatch.ResourceLogs().AppendEmpty(), "same_attr_val", "1")
	fillResourceLogs(inBatch.ResourceLogs().AppendEmpty(), "same_attr_val", "1")
	fillResourceLogs(inBatch.ResourceLogs().AppendEmpty(), "same_attr_val", "1")
	fillResourceLogs(inBatch.ResourceLogs().AppendEmpty(), "same_attr_val", "1")
	expected := plog.NewLogs()
	inBatch.CopyTo(expected)

	outBatches := organizeLogs([]string{"same_attr_val"}, inBatch)
	require.Len(t, outBatches, 1)
	assert.Equal(t, expected, outBatches[0])
}

func TestSplitLogsIntoDifferentBatches(t *testing.T) {
	inBatch := plog.NewLogs()
	fillResourceLogs(inBatch.ResourceLogs().AppendEmpty(), "attr_key", "1")
	fillResourceLogs(inBatch.ResourceLogs().AppendEmpty(), "attr_key", "2")
	fillResourceLogs(inBatch.ResourceLogs().AppendEmpty(), "attr_key", "3")
	fillResourceLogs(inBatch.ResourceLogs().AppendEmpty(), "attr_key", "4")
	fillResourceLogs(inBatch.ResourceLogs().AppendEmpty(), "attr_key", "1")
	fillResourceLogs(inBatch.ResourceLogs().AppendEmpty(), "attr_key", "2")
	fillResourceLogs(inBatch.ResourceLogs().AppendEmpty(), "attr_key", "3")
	fillResourceLogs(inBatch.ResourceLogs().AppendEmpty(), "attr_key", "4")
	fillResourceLogs(inBatch.ResourceLogs().AppendEmpty(), "diff_attr_key", "1")
	expected := plog.NewLogs()
	inBatch.CopyTo(expected)

	outBatches := organizeLogs([]string{"attr_key"}, inBatch)
	require.Len(t, outBatches, 5)
	sortLogs(outBatches, "attr_key")
	assert.Equal(t, newLogs(expected.ResourceLogs().At(8)), outBatches[0])
	assert.Equal(t, newLogs(expected.ResourceLogs().At(0), expected.ResourceLogs().At(4)), outBatches[1])
	assert.Equal(t, newLogs(expected.ResourceLogs().At(1), expected.ResourceLogs().At(5)), outBatches[2])
	assert.Equal(t, newLogs(expected.ResourceLogs().At(2), expected.ResourceLogs().At(6)), outBatches[3])
	assert.Equal(t, newLogs(expected.ResourceLogs().At(3), expected.ResourceLogs().At(7)), outBatches[4])
}

func TestSplitLogsIntoDifferentBatchesWithMultipleKeys(t *testing.T) {
	inBatch := plog.NewLogs()
	fillResourceLogs(inBatch.ResourceLogs().AppendEmpty(), "attr_key", "1", "attr_key2", "1")
	fillResourceLogs(inBatch.ResourceLogs().AppendEmpty(), "attr_key", "2", "attr_key2", "2")
	fillResourceLogs(inBatch.ResourceLogs().AppendEmpty(), "attr_key", "3", "attr_key2", "3")
	fillResourceLogs(inBatch.ResourceLogs().AppendEmpty(), "attr_key", "4", "attr_key2", "4")
	fillResourceLogs(inBatch.ResourceLogs().AppendEmpty(), "attr_key", "1", "attr_key2", "1")
	fillResourceLogs(inBatch.ResourceLogs().AppendEmpty(), "attr_key", "2", "attr_key2", "2")
	fillResourceLogs(inBatch.ResourceLogs().AppendEmpty(), "attr_key", "3", "attr_key2", "3")
	fillResourceLogs(inBatch.ResourceLogs().AppendEmpty(), "attr_key", "4", "attr_key2", "4")
	fillResourceLogs(inBatch.ResourceLogs().AppendEmpty(), "attr_key", "5", "attr_key2", "6")
	fillResourceLogs(inBatch.ResourceLogs().AppendEmpty(), "diff_attr_key", "1")
	expected := plog.NewLogs()
	inBatch.CopyTo(expected)

	outBatches := organizeLogs([]string{"attr_key", "attr_key2"}, inBatch)
	require.Len(t, outBatches, 6)
	sortLogs(outBatches, "attr_key")
	assert.Equal(t, newLogs(expected.ResourceLogs().At(9)), outBatches[0])
	assert.Equal(t, newLogs(expected.ResourceLogs().At(0), expected.ResourceLogs().At(4)), outBatches[1])
	assert.Equal(t, newLogs(expected.ResourceLogs().At(1), expected.ResourceLogs().At(5)), outBatches[2])
	assert.Equal(t, newLogs(expected.ResourceLogs().At(2), expected.ResourceLogs().At(6)), outBatches[3])
	assert.Equal(t, newLogs(expected.ResourceLogs().At(3), expected.ResourceLogs().At(7)), outBatches[4])
	assert.Equal(t, newLogs(expected.ResourceLogs().At(8)), outBatches[5])
}
func newTraces(rss ...ptrace.ResourceSpans) ptrace.Traces {
	td := ptrace.NewTraces()
	for _, rs := range rss {
		rs.CopyTo(td.ResourceSpans().AppendEmpty())
	}
	return td
}

func sortTraces(tds []ptrace.Traces, attrKey string) {
	sort.Slice(tds, func(i, j int) bool {
		valI := ""
		if av, ok := tds[i].ResourceSpans().At(0).Resource().Attributes().Get(attrKey); ok {
			valI = av.Str()
		}
		valJ := ""
		if av, ok := tds[j].ResourceSpans().At(0).Resource().Attributes().Get(attrKey); ok {
			valJ = av.Str()
		}
		return valI < valJ
	})
}
func fillResourceSpans(rs ptrace.ResourceSpans, kv ...string) {
	for i := 0; i < len(kv); i += 2 {
		rs.Resource().Attributes().PutStr(kv[i], kv[i+1])
	}

	rs.Resource().Attributes().PutInt("__other_key__", 123)
	ils := rs.ScopeSpans().AppendEmpty()
	firstSpan := ils.Spans().AppendEmpty()
	firstSpan.SetName("first-span")
	firstSpan.SetTraceID(pcommon.TraceID([16]byte{byte(rand.Int())}))
	secondSpan := ils.Spans().AppendEmpty()
	secondSpan.SetName("second-span")
	secondSpan.SetTraceID(pcommon.TraceID([16]byte{byte(rand.Int())}))
}

func newMetrics(rms ...pmetric.ResourceMetrics) pmetric.Metrics {
	md := pmetric.NewMetrics()
	for _, rm := range rms {
		rm.CopyTo(md.ResourceMetrics().AppendEmpty())
	}
	return md
}

func sortMetrics(tds []pmetric.Metrics, attrKey string) {
	sort.Slice(tds, func(i, j int) bool {
		valI := ""
		if av, ok := tds[i].ResourceMetrics().At(0).Resource().Attributes().Get(attrKey); ok {
			valI = av.Str()
		}
		valJ := ""
		if av, ok := tds[j].ResourceMetrics().At(0).Resource().Attributes().Get(attrKey); ok {
			valJ = av.Str()
		}
		return valI < valJ
	})
}

func fillResourceMetrics(rs pmetric.ResourceMetrics, kv ...string) {
	for i := 0; i < len(kv); i += 2 {
		rs.Resource().Attributes().PutStr(kv[i], kv[i+1])
	}
	rs.Resource().Attributes().PutInt("__other_key__", 123)
	ils := rs.ScopeMetrics().AppendEmpty()
	firstMetric := ils.Metrics().AppendEmpty()
	firstMetric.SetName("first-metric")
	firstMetric.SetEmptyGauge()
	secondMetric := ils.Metrics().AppendEmpty()
	secondMetric.SetName("second-metric")
	secondMetric.SetEmptySum()
}

func newLogs(rls ...plog.ResourceLogs) plog.Logs {
	ld := plog.NewLogs()
	for _, rl := range rls {
		rl.CopyTo(ld.ResourceLogs().AppendEmpty())
	}
	return ld
}

func sortLogs(tds []plog.Logs, attrKey string) {
	sort.Slice(tds, func(i, j int) bool {
		valI := ""
		if av, ok := tds[i].ResourceLogs().At(0).Resource().Attributes().Get(attrKey); ok {
			valI = av.Str()
		}
		valJ := ""
		if av, ok := tds[j].ResourceLogs().At(0).Resource().Attributes().Get(attrKey); ok {
			valJ = av.Str()
		}
		return valI < valJ
	})
}

func fillResourceLogs(rs plog.ResourceLogs, kv ...string) {
	for i := 0; i < len(kv); i += 2 {
		rs.Resource().Attributes().PutStr(kv[i], kv[i+1])
	}
	rs.Resource().Attributes().PutInt("__other_key__", 123)
	ils := rs.ScopeLogs().AppendEmpty()
	firstLogRecord := ils.LogRecords().AppendEmpty()
	firstLogRecord.SetFlags(plog.LogRecordFlags(rand.Int31()))
	secondLogRecord := ils.LogRecords().AppendEmpty()
	secondLogRecord.SetFlags(plog.LogRecordFlags(rand.Int31()))
}

func BenchmarkBatchPerResourceTraces(b *testing.B) {
	inBatch := ptrace.NewTraces()
	rss := inBatch.ResourceSpans()
	rss.EnsureCapacity(64)
	for i := 0; i < 64; i++ {
		fillResourceSpans(rss.AppendEmpty(), "attr_key", strconv.Itoa(i%8))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		organizeTraces([]string{"attr_key"}, inBatch)
	}
}

func BenchmarkBatchPerResourceMetrics(b *testing.B) {
	inBatch := pmetric.NewMetrics()
	inBatch.ResourceMetrics().EnsureCapacity(64)
	for i := 0; i < 64; i++ {
		fillResourceMetrics(inBatch.ResourceMetrics().AppendEmpty(), "attr_key", strconv.Itoa(i%8))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = organizeMetrics([]string{"attr_key"}, inBatch)
	}
}

func BenchmarkBatchPerResourceLogs(b *testing.B) {
	inBatch := plog.NewLogs()
	inBatch.ResourceLogs().EnsureCapacity(64)
	for i := 0; i < 64; i++ {
		fillResourceLogs(inBatch.ResourceLogs().AppendEmpty(), "attr_key", strconv.Itoa(i%8))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = organizeLogs([]string{"attr_key"}, inBatch)
	}
}

type nopSender struct {
	baseRequestSender
}

func (n nopSender) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (n nopSender) Shutdown(_ context.Context) error {
	return nil
}

func (n nopSender) send(_ context.Context, _ Request) error {
	return nil
}

func (n nopSender) setNextSender(_ requestSender) {
	panic("unsupported")
}

func TestSend(t *testing.T) {

	sender := &organizeSender{
		attrKeys: []string{"attr_key"},
	}
	sender.setNextSender(nopSender{})

	logBatch := plog.NewLogs()
	fillResourceLogs(logBatch.ResourceLogs().AppendEmpty(), "attr_key", "1")
	err := sender.send(context.Background(), &logsRequest{ld: logBatch})
	require.NoError(t, err)

	tracesBatch := ptrace.NewTraces()
	fillResourceSpans(tracesBatch.ResourceSpans().AppendEmpty(), "attr_key", "1")
	err = sender.send(context.Background(), &tracesRequest{td: tracesBatch})
	require.NoError(t, err)

	metricsBatch := pmetric.NewMetrics()
	fillResourceSpans(tracesBatch.ResourceSpans().AppendEmpty(), "attr_key", "1")
	err = sender.send(context.Background(), &metricsRequest{md: metricsBatch})
	require.NoError(t, err)

	err = sender.send(context.Background(), nil)
	require.EqualError(t, err, "unsupported type <nil>")
}
