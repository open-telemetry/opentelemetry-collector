// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// oversized is a body/attribute value large enough that a single item carrying it
// cannot fit into a maxSize=100/150 batch on its own.
var oversized = strings.Repeat("x", 1000)

// --- logs ---

func newLogsWithBodies(bodies ...string) plog.Logs {
	ld := plog.NewLogs()
	sl := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty()
	for _, b := range bodies {
		sl.LogRecords().AppendEmpty().Body().SetStr(b)
	}
	return ld
}

func logBodies(reqs []request.Request) []string {
	var out []string
	for _, r := range reqs {
		rls := r.(*logsRequest).ld.ResourceLogs()
		for i := 0; i < rls.Len(); i++ {
			sls := rls.At(i).ScopeLogs()
			for j := 0; j < sls.Len(); j++ {
				lrs := sls.At(j).LogRecords()
				for k := 0; k < lrs.Len(); k++ {
					out = append(out, lrs.At(k).Body().Str())
				}
			}
		}
	}
	return out
}

func TestSplitLogsDropsOnlyOversizedRecord(t *testing.T) {
	tests := []struct {
		name         string
		bodies       []string
		wantSurvived []string
		wantDropped  int
	}{
		{"oversized_first", []string{oversized, "a", "b", "c"}, []string{"a", "b", "c"}, 1},
		{"oversized_in_middle", []string{"a", "b", oversized, "c", "d"}, []string{"a", "b", "c", "d"}, 1},
		{"oversized_last", []string{"a", "b", "c", oversized}, []string{"a", "b", "c"}, 1},
		{"multiple_oversized", []string{"a", oversized, "b", oversized, "c"}, []string{"a", "b", "c"}, 2},
		{"all_oversized", []string{oversized, oversized}, nil, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newLogsRequest(newLogsWithBodies(tt.bodies...))
			res, err := req.MergeSplit(context.Background(), 100, request.SizerTypeBytes, nil)
			require.ErrorContains(t, err, fmt.Sprintf("one log record size is greater than max size, dropping items: %d", tt.wantDropped))
			assert.Equal(t, tt.wantSurvived, logBodies(res))
			for _, r := range res {
				assert.LessOrEqual(t, r.BytesSize(), 100)
			}
		})
	}
}

func TestSplitLogsUnsplittableOverheadOnly(t *testing.T) {
	ld := plog.NewLogs()
	ld.ResourceLogs().AppendEmpty().Resource().Attributes().PutStr("big", strings.Repeat("x", 500))
	req := newLogsRequest(ld)
	require.Greater(t, req.BytesSize(), 100)
	res, err := req.MergeSplit(context.Background(), 100, request.SizerTypeBytes, nil)
	require.ErrorContains(t, err, "has no log records left to drop")
	assert.Empty(t, res)
}

// --- traces ---

func newTracesWithSpans(names ...string) ptrace.Traces {
	td := ptrace.NewTraces()
	ss := td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty()
	for _, n := range names {
		span := ss.Spans().AppendEmpty()
		span.SetName(n)
		if n == "BIG" {
			span.Attributes().PutStr("pad", oversized)
		}
	}
	return td
}

func spanNames(reqs []request.Request) []string {
	var out []string
	for _, r := range reqs {
		rss := r.(*tracesRequest).td.ResourceSpans()
		for i := 0; i < rss.Len(); i++ {
			sss := rss.At(i).ScopeSpans()
			for j := 0; j < sss.Len(); j++ {
				spans := sss.At(j).Spans()
				for k := 0; k < spans.Len(); k++ {
					out = append(out, spans.At(k).Name())
				}
			}
		}
	}
	return out
}

func TestSplitTracesDropsOnlyOversizedSpan(t *testing.T) {
	tests := []struct {
		name         string
		spans        []string
		wantSurvived []string
		wantDropped  int
	}{
		{"oversized_first", []string{"BIG", "a", "b", "c"}, []string{"a", "b", "c"}, 1},
		{"oversized_in_middle", []string{"a", "b", "BIG", "c", "d"}, []string{"a", "b", "c", "d"}, 1},
		{"oversized_last", []string{"a", "b", "c", "BIG"}, []string{"a", "b", "c"}, 1},
		{"multiple_oversized", []string{"a", "BIG", "b", "BIG", "c"}, []string{"a", "b", "c"}, 2},
		{"all_oversized", []string{"BIG", "BIG"}, nil, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newTracesRequest(newTracesWithSpans(tt.spans...))
			res, err := req.MergeSplit(context.Background(), 100, request.SizerTypeBytes, nil)
			require.ErrorContains(t, err, fmt.Sprintf("one span size is greater than max size, dropping items: %d", tt.wantDropped))
			assert.Equal(t, tt.wantSurvived, spanNames(res))
			for _, r := range res {
				assert.LessOrEqual(t, r.BytesSize(), 100)
			}
		})
	}
}

// --- metrics ---

// newGaugeMetrics builds one gauge metric per id, each with a single data point; an
// id of "BIG" gets a padding attribute large enough that the data point cannot fit
// into any batch.
func newGaugeMetrics(ids ...string) pmetric.Metrics {
	md := pmetric.NewMetrics()
	sm := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty()
	for _, id := range ids {
		m := sm.Metrics().AppendEmpty()
		m.SetName(id)
		dp := m.SetEmptyGauge().DataPoints().AppendEmpty()
		dp.SetIntValue(1)
		dp.Attributes().PutStr("id", id)
		if id == "BIG" {
			dp.Attributes().PutStr("pad", oversized)
		}
	}
	return md
}

func gaugeIDs(reqs []request.Request) []string {
	var out []string
	for _, r := range reqs {
		rms := r.(*metricsRequest).md.ResourceMetrics()
		for i := 0; i < rms.Len(); i++ {
			sms := rms.At(i).ScopeMetrics()
			for j := 0; j < sms.Len(); j++ {
				ms := sms.At(j).Metrics()
				for k := 0; k < ms.Len(); k++ {
					dps := ms.At(k).Gauge().DataPoints()
					for l := 0; l < dps.Len(); l++ {
						v, _ := dps.At(l).Attributes().Get("id")
						out = append(out, v.Str())
					}
				}
			}
		}
	}
	return out
}

func TestSplitMetricsDropsOnlyOversizedDataPoint(t *testing.T) {
	tests := []struct {
		name         string
		ids          []string
		wantSurvived []string
		wantDropped  int
	}{
		{"oversized_first", []string{"BIG", "a", "b", "c"}, []string{"a", "b", "c"}, 1},
		{"oversized_in_middle", []string{"a", "b", "BIG", "c", "d"}, []string{"a", "b", "c", "d"}, 1},
		{"oversized_last", []string{"a", "b", "c", "BIG"}, []string{"a", "b", "c"}, 1},
		{"multiple_oversized", []string{"a", "BIG", "b", "BIG", "c"}, []string{"a", "b", "c"}, 2},
		{"all_oversized", []string{"BIG", "BIG"}, nil, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newMetricsRequest(newGaugeMetrics(tt.ids...))
			res, err := req.MergeSplit(context.Background(), 150, request.SizerTypeBytes, nil)
			require.ErrorContains(t, err, fmt.Sprintf("one datapoint size is greater than max size, dropping items: %d", tt.wantDropped))
			assert.Equal(t, tt.wantSurvived, gaugeIDs(res))
			for _, r := range res {
				assert.LessOrEqual(t, r.BytesSize(), 150)
			}
		})
	}
}

// TestSplitMetricsDropsConsecutiveOversizedAcrossContainers ensures the force-first path
// keeps working when consecutive oversized data points live in separate scopes or separate
// resources. Extracting a metric's only data point leaves an empty metric behind; the guard
// must count data points, not containers, or the empty metric/scope/resource would disable
// force-first for the next oversized data point.
func TestSplitMetricsDropsConsecutiveOversizedAcrossContainers(t *testing.T) {
	tests := []struct {
		name string
		md   pmetric.Metrics
	}{
		{
			name: "separate_scopes",
			md: func() pmetric.Metrics {
				md := pmetric.NewMetrics()
				rm := md.ResourceMetrics().AppendEmpty()
				for range 2 {
					dp := rm.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptyGauge().DataPoints().AppendEmpty()
					dp.SetIntValue(1)
					dp.Attributes().PutStr("pad", oversized)
				}
				return md
			}(),
		},
		{
			name: "separate_resources",
			md: func() pmetric.Metrics {
				md := pmetric.NewMetrics()
				for range 2 {
					dp := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptyGauge().DataPoints().AppendEmpty()
					dp.SetIntValue(1)
					dp.Attributes().PutStr("pad", oversized)
				}
				return md
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := newMetricsRequest(tt.md).MergeSplit(context.Background(), 150, request.SizerTypeBytes, nil)
			require.ErrorContains(t, err, "one datapoint size is greater than max size, dropping items: 2")
			for _, r := range res {
				assert.LessOrEqual(t, r.BytesSize(), 150)
			}
		})
	}
}

// TestSplitMetricsDropsOversizedForEveryType exercises the force-first path in each
// per-type extract*DataPoints branch: a metric with [small, oversized, small] data
// points must keep the two small ones and drop the oversized one.
func TestSplitMetricsDropsOversizedForEveryType(t *testing.T) {
	pad := strings.Repeat("y", 400)
	appendDP := func(m pmetric.Metric, typ pmetric.MetricType) pcommon.Map {
		switch typ {
		case pmetric.MetricTypeGauge:
			return m.Gauge().DataPoints().AppendEmpty().Attributes()
		case pmetric.MetricTypeSum:
			return m.Sum().DataPoints().AppendEmpty().Attributes()
		case pmetric.MetricTypeHistogram:
			return m.Histogram().DataPoints().AppendEmpty().Attributes()
		case pmetric.MetricTypeExponentialHistogram:
			return m.ExponentialHistogram().DataPoints().AppendEmpty().Attributes()
		case pmetric.MetricTypeSummary:
			return m.Summary().DataPoints().AppendEmpty().Attributes()
		}
		t.Fatalf("unhandled metric type %v", typ)
		return pcommon.NewMap()
	}

	tests := []struct {
		name  string
		typ   pmetric.MetricType
		setup func(pmetric.Metric)
	}{
		{"gauge", pmetric.MetricTypeGauge, func(m pmetric.Metric) { m.SetEmptyGauge() }},
		{"sum", pmetric.MetricTypeSum, func(m pmetric.Metric) { m.SetEmptySum() }},
		{"histogram", pmetric.MetricTypeHistogram, func(m pmetric.Metric) { m.SetEmptyHistogram() }},
		{"exponential_histogram", pmetric.MetricTypeExponentialHistogram, func(m pmetric.Metric) { m.SetEmptyExponentialHistogram() }},
		{"summary", pmetric.MetricTypeSummary, func(m pmetric.Metric) { m.SetEmptySummary() }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := pmetric.NewMetrics()
			m := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
			m.SetName("test." + tt.name)
			tt.setup(m)
			appendDP(m, tt.typ).PutStr("id", "small_before")
			appendDP(m, tt.typ).PutStr("pad", pad)
			appendDP(m, tt.typ).PutStr("id", "small_after")
			require.Equal(t, 3, md.DataPointCount())

			res, err := newMetricsRequest(md).MergeSplit(context.Background(), 100, request.SizerTypeBytes, nil)
			require.ErrorContains(t, err, "one datapoint size is greater than max size, dropping items: 1")
			got := 0
			for _, r := range res {
				got += r.(*metricsRequest).md.DataPointCount()
				assert.LessOrEqual(t, r.BytesSize(), 100)
			}
			assert.Equal(t, 2, got, "the well-sized data points on either side must survive")
		})
	}
}
