// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

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

	ako := attrKeyOrganizer{attrKeys: []string{"attr_key"}}

	sender := &organizeSender{
		organizeMetrics: ako.organizeMetrics,
		organizeLogs:    ako.organizeLogs,
		organizeTraces:  ako.organizeTraces,
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
	fillResourceMetrics(metricsBatch.ResourceMetrics().AppendEmpty(), "attr_key", "1")
	err = sender.send(context.Background(), &metricsRequest{md: metricsBatch})
	require.NoError(t, err)

	err = sender.send(context.Background(), nil)
	require.EqualError(t, err, "unsupported type <nil>")
}

func TestSendNoFuncs(t *testing.T) {

	sender := &organizeSender{}
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
	fillResourceMetrics(metricsBatch.ResourceMetrics().AppendEmpty(), "attr_key", "1")
	err = sender.send(context.Background(), &metricsRequest{md: metricsBatch})
	require.NoError(t, err)

	err = sender.send(context.Background(), nil)
	require.EqualError(t, err, "unsupported type <nil>")
}

type attrKeyOrganizer struct {
	attrKeys []string
}

func (ako attrKeyOrganizer) organizeMetrics(md pmetric.Metrics) []pmetric.Metrics {
	rms := md.ResourceMetrics()
	lenRls := rms.Len()
	// If zero or one resource metrics, return early
	if lenRls <= 1 {
		return []pmetric.Metrics{md}
	}

	indicesByAttr := make(map[string][]int)
	for i := 0; i < lenRls; i++ {
		rm := rms.At(i)
		var attrVal string
		for _, k := range ako.attrKeys {
			if attributeValue, ok := rm.Resource().Attributes().Get(k); ok {
				attrVal = fmt.Sprintf("%s%s%s", attrVal, separator, attributeValue.Str())
			}
		}
		indicesByAttr[attrVal] = append(indicesByAttr[attrVal], i)
	}
	// If there is a single attribute value, return early
	if len(indicesByAttr) <= 1 {
		return []pmetric.Metrics{md}
	}

	var result []pmetric.Metrics
	for _, indices := range indicesByAttr {
		metricsForAttr := pmetric.NewMetrics()
		result = append(result, metricsForAttr)
		for _, i := range indices {
			rs := rms.At(i)
			rs.CopyTo(metricsForAttr.ResourceMetrics().AppendEmpty())
		}
	}
	return result
}
func (ako attrKeyOrganizer) organizeLogs(ld plog.Logs) []plog.Logs {
	rls := ld.ResourceLogs()
	lenRls := rls.Len()
	// If zero or one resource logs, return early
	if lenRls <= 1 {
		return []plog.Logs{ld}
	}

	indicesByAttr := make(map[string][]int)
	for i := 0; i < lenRls; i++ {
		rl := rls.At(i)
		var attrVal string
		for _, k := range ako.attrKeys {
			if attributeValue, ok := rl.Resource().Attributes().Get(k); ok {
				attrVal = fmt.Sprintf("%s%s%s", attrVal, separator, attributeValue.Str())
			}
		}
		indicesByAttr[attrVal] = append(indicesByAttr[attrVal], i)
	}
	// If there is a single attribute value, return early
	if len(indicesByAttr) <= 1 {
		return []plog.Logs{ld}
	}

	var result []plog.Logs
	for _, indices := range indicesByAttr {
		logsForAttr := plog.NewLogs()
		result = append(result, logsForAttr)
		for _, i := range indices {
			rs := rls.At(i)
			rs.CopyTo(logsForAttr.ResourceLogs().AppendEmpty())
		}
	}
	return result
}

func (ako attrKeyOrganizer) organizeTraces(td ptrace.Traces) []ptrace.Traces {
	rss := td.ResourceSpans()
	lenRls := rss.Len()
	// If zero or one resource traces, return early
	if lenRls <= 1 {
		return []ptrace.Traces{td}
	}

	indicesByAttr := make(map[string][]int)
	for i := 0; i < lenRls; i++ {
		rs := rss.At(i)
		var attrVal string
		for _, k := range ako.attrKeys {
			if attributeValue, ok := rs.Resource().Attributes().Get(k); ok {
				attrVal = fmt.Sprintf("%s%s%s", attrVal, separator, attributeValue.Str())
			}
		}
		indicesByAttr[attrVal] = append(indicesByAttr[attrVal], i)
	}
	// If there is a single attribute value, return early
	if len(indicesByAttr) <= 1 {
		return []ptrace.Traces{td}
	}

	var result []ptrace.Traces
	for _, indices := range indicesByAttr {
		tracesForAttr := ptrace.NewTraces()
		result = append(result, tracesForAttr)
		for _, i := range indices {
			rs := rss.At(i)
			rs.CopyTo(tracesForAttr.ResourceSpans().AppendEmpty())
		}
	}
	return result
}
