// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumertest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestTracesSink(t *testing.T) {
	sink := new(TracesSink)
	td := testdata.GenerateTraces(1)
	want := make([]ptrace.Traces, 0, 7)
	for i := 0; i < 7; i++ {
		require.NoError(t, sink.ConsumeTraces(context.Background(), td))
		want = append(want, td)
	}
	assert.Equal(t, want, sink.AllTraces())
	assert.Equal(t, len(want), sink.SpanCount())
	sink.Reset()
	assert.Empty(t, sink.AllTraces())
	assert.Equal(t, 0, sink.SpanCount())
}

func TestMetricsSink(t *testing.T) {
	sink := new(MetricsSink)
	md := testdata.GenerateMetrics(1)
	want := make([]pmetric.Metrics, 0, 7)
	for i := 0; i < 7; i++ {
		require.NoError(t, sink.ConsumeMetrics(context.Background(), md))
		want = append(want, md)
	}
	assert.Equal(t, want, sink.AllMetrics())
	assert.Equal(t, 2*len(want), sink.DataPointCount())
	sink.Reset()
	assert.Empty(t, sink.AllMetrics())
	assert.Equal(t, 0, sink.DataPointCount())
}

func TestLogsSink(t *testing.T) {
	sink := new(LogsSink)
	md := testdata.GenerateLogs(1)
	want := make([]plog.Logs, 0, 7)
	for i := 0; i < 7; i++ {
		require.NoError(t, sink.ConsumeLogs(context.Background(), md))
		want = append(want, md)
	}
	assert.Equal(t, want, sink.AllLogs())
	assert.Equal(t, len(want), sink.LogRecordCount())
	sink.Reset()
	assert.Empty(t, sink.AllLogs())
	assert.Equal(t, 0, sink.LogRecordCount())
}

func TestProfilesSink(t *testing.T) {
	sink := new(ProfilesSink)
	td := testdata.GenerateProfiles(1)
	want := make([]pprofile.Profiles, 0, 7)
	for i := 0; i < 7; i++ {
		require.NoError(t, sink.ConsumeProfiles(context.Background(), td))
		want = append(want, td)
	}
	assert.Equal(t, want, sink.AllProfiles())
	assert.Equal(t, len(want), sink.SampleCount())
	sink.Reset()
	assert.Empty(t, sink.AllProfiles())
	assert.Empty(t, sink.SampleCount())
}
