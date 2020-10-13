// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package telemetryprocessor

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/stats/view"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/collector/telemetry"
	"go.opentelemetry.io/collector/internal/data/testdata"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/translator/conventions"
)

func TestErrorPath(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	views := processor.MetricViews(configtelemetry.LevelDetailed)
	assert.NoError(t, view.Register(views...))
	defer view.Unregister(views...)

	mockP := newMockSpanProcessor()
	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	cfg := createDefaultConfig().(*Config)
	qp := newTelemetryTracesProcessor(creationParams, mockP, cfg)
	require.NoError(t, qp.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, qp.Shutdown(context.Background()))
	})

	mockP.consumeError = errors.New("consume error")

	wantBatches := 10
	for i := 0; i < wantBatches; i++ {
		td := testdata.GenerateTraceDataTwoSpansSameResource()
		require.Error(t, qp.ConsumeTraces(context.Background(), td))
	}

	obsreporttest.CheckProcessorTracesViews(t, cfg.Name(), int64(0), 20, 0)
}

func TestTraceHappyPath(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	lvl, err := telemetry.GetLevel()
	require.NoError(t, err)
	telemetry.SetLevel(configtelemetry.LevelDetailed)
	defer telemetry.SetLevel(lvl)

	views := processor.MetricViews(configtelemetry.LevelDetailed)
	assert.NoError(t, view.Register(views...))
	defer view.Unregister(views...)

	mockP := newMockSpanProcessor()
	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	cfg := createDefaultConfig().(*Config)
	qp := newTelemetryTracesProcessor(creationParams, mockP, cfg)
	require.NoError(t, qp.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, qp.Shutdown(context.Background()))
	})

	wantBatches := 10
	wantSpans := 20
	for i := 0; i < wantBatches; i++ {
		td := testdata.GenerateTraceDataTwoSpansSameResource()
		rs := td.ResourceSpans()
		for j := 0; j < rs.Len(); j++ {
			r := rs.At(j)
			r.Resource().Attributes().InsertString(conventions.AttributeServiceName, "foobar")
		}
		require.NoError(t, qp.ConsumeTraces(context.Background(), td))
	}

	mockP.checkNumBatches(t, wantBatches)
	mockP.checkNumSpans(t, wantSpans)

	droppedView, err := findViewNamed(views, processor.StatDroppedSpanCount.Name())
	require.NoError(t, err)

	data, err := view.RetrieveData(droppedView.Name)
	require.NoError(t, err)
	require.Len(t, data, 1)
	assert.Equal(t, 0.0, data[0].Data.(*view.SumData).Value)

	data, err = view.RetrieveData(processor.StatTraceBatchesDroppedCount.Name())
	require.NoError(t, err)
	assert.Equal(t, 0.0, data[0].Data.(*view.SumData).Value)
	obsreporttest.CheckProcessorTracesViews(t, cfg.Name(), int64(wantSpans), 0, 0)

	av, err := findViewNamed(views, processor.StatReceivedSpanCount.Name())
	require.NoError(t, err)

	data, err = view.RetrieveData(av.Name)
	require.NoError(t, err)

	var serviceName string
	for _, v := range data {
		for _, t := range v.Tags {
			if t.Key.Name() == "service" {
				serviceName = t.Value
			}
		}
	}
	assert.Equal(t, "foobar", serviceName)
}

func TestMetricsQueueProcessorHappyPath(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	mockP := newMockSpanProcessor()
	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	cfg := createDefaultConfig().(*Config)
	qp := newTelemetryMetricsProcessor(creationParams, mockP, cfg)
	require.NoError(t, qp.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, qp.Shutdown(context.Background()))
	})

	wantBatches := 10
	wantMetricPoints := 2 * 20
	for i := 0; i < wantBatches; i++ {
		md := testdata.GenerateMetricsTwoMetrics()
		require.NoError(t, qp.ConsumeMetrics(context.Background(), md))
	}

	mockP.checkNumBatches(t, wantBatches)
	mockP.checkNumPoints(t, wantMetricPoints)
	obsreporttest.CheckProcessorMetricsViews(t, cfg.Name(), int64(wantMetricPoints), 0, 0)
}

type mockSpanProcessor struct {
	consumeError      error
	batchCount        int64
	spanCount         int64
	metricPointsCount int64
}

var _ consumer.TraceConsumer = (*mockSpanProcessor)(nil)
var _ consumer.MetricsConsumer = (*mockSpanProcessor)(nil)

func newMockSpanProcessor() *mockSpanProcessor {
	return &mockSpanProcessor{}
}

func (p *mockSpanProcessor) ConsumeTraces(_ context.Context, td pdata.Traces) error {
	p.batchCount++
	p.spanCount += int64(td.SpanCount())
	return p.consumeError
}

func (p *mockSpanProcessor) ConsumeMetrics(_ context.Context, md pdata.Metrics) error {
	p.batchCount++
	_, mpc := md.MetricAndDataPointCount()
	p.metricPointsCount += int64(mpc)
	return p.consumeError
}

func (p *mockSpanProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: false}
}

func (p *mockSpanProcessor) checkNumBatches(t *testing.T, want int) {
	t.Helper()
	assert.EqualValues(t, want, int(p.batchCount))
}

func (p *mockSpanProcessor) checkNumSpans(t *testing.T, want int) {
	t.Helper()
	assert.EqualValues(t, want, int(p.spanCount))
}

func (p *mockSpanProcessor) checkNumPoints(t *testing.T, want int) {
	t.Helper()
	assert.EqualValues(t, want, int(p.metricPointsCount))
}

func findViewNamed(views []*view.View, name string) (*view.View, error) {
	for _, v := range views {
		if v.Name == name {
			return v, nil
		}
	}
	return nil, fmt.Errorf("view %s not found", name)
}
