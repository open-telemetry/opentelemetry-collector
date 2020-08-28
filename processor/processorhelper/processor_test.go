// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package processorhelper

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/internal/data/testdata"
	"go.opentelemetry.io/collector/internal/dataold/testdataold"
)

const testFullName = "testFullName"

var testCfg = &configmodels.ProcessorSettings{
	TypeVal: testFullName,
	NameVal: testFullName,
}

func TestWithStart(t *testing.T) {
	startCalled := false
	start := func(context.Context, component.Host) error { startCalled = true; return nil }

	bp := newBaseProcessor(testFullName, WithStart(start))
	assert.NoError(t, bp.Start(context.Background(), componenttest.NewNopHost()))
	assert.True(t, startCalled)
}

func TestWithStart_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	start := func(context.Context, component.Host) error { return want }

	bp := newBaseProcessor(testFullName, WithStart(start))
	assert.Equal(t, want, bp.Start(context.Background(), componenttest.NewNopHost()))
}

func TestWithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	bp := newBaseProcessor(testFullName, WithShutdown(shutdown))
	assert.NoError(t, bp.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestWithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	bp := newBaseProcessor(testFullName, WithShutdown(shutdownErr))
	assert.Equal(t, want, bp.Shutdown(context.Background()))
}

func TestWithCapabilities(t *testing.T) {
	bp := newBaseProcessor(testFullName)
	assert.True(t, bp.GetCapabilities().MutatesConsumedData)

	bp = newBaseProcessor(testFullName, WithCapabilities(component.ProcessorCapabilities{MutatesConsumedData: false}))
	assert.False(t, bp.GetCapabilities().MutatesConsumedData)
}

func TestNewTraceExporter(t *testing.T) {
	me, err := NewTraceProcessor(testCfg, exportertest.NewNopTraceExporter(), newTestTProcessor(nil))
	require.NoError(t, err)

	assert.NoError(t, me.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, me.ConsumeTraces(context.Background(), testdata.GenerateTraceDataEmpty()))
	assert.NoError(t, me.Shutdown(context.Background()))
}

func TestNewTraceExporter_NilRequiredFields(t *testing.T) {
	_, err := NewTraceProcessor(testCfg, exportertest.NewNopTraceExporter(), nil)
	assert.Error(t, err)

	_, err = NewTraceProcessor(testCfg, nil, newTestTProcessor(nil))
	assert.Equal(t, componenterror.ErrNilNextConsumer, err)
}

func TestNewTraceExporter_ProcessTraceError(t *testing.T) {
	want := errors.New("my_error")
	me, err := NewTraceProcessor(testCfg, exportertest.NewNopTraceExporter(), newTestTProcessor(want))
	require.NoError(t, err)
	assert.Equal(t, want, me.ConsumeTraces(context.Background(), testdata.GenerateTraceDataEmpty()))
}

func TestNewMetricsExporter(t *testing.T) {
	me, err := NewMetricsProcessor(testCfg, exportertest.NewNopMetricsExporter(), newTestMProcessor(nil))
	require.NoError(t, err)

	assert.NoError(t, me.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, me.ConsumeMetrics(context.Background(), pdatautil.MetricsFromOldInternalMetrics(testdataold.GenerateMetricDataEmpty())))
	assert.NoError(t, me.Shutdown(context.Background()))
}

func TestNewMetricsExporter_NilRequiredFields(t *testing.T) {
	_, err := NewMetricsProcessor(testCfg, exportertest.NewNopMetricsExporter(), nil)
	assert.Error(t, err)

	_, err = NewMetricsProcessor(testCfg, nil, newTestMProcessor(nil))
	assert.Equal(t, componenterror.ErrNilNextConsumer, err)
}

func TestNewMetricsExporter_ProcessMetricsError(t *testing.T) {
	want := errors.New("my_error")
	me, err := NewMetricsProcessor(testCfg, exportertest.NewNopMetricsExporter(), newTestMProcessor(want))
	require.NoError(t, err)
	assert.Equal(t, want, me.ConsumeMetrics(context.Background(), pdatautil.MetricsFromOldInternalMetrics(testdataold.GenerateMetricDataEmpty())))
}

func TestNewMetricsExporter_ProcessMetricsErrSkipProcessingData(t *testing.T) {
	me, err := NewMetricsProcessor(testCfg, exportertest.NewNopMetricsExporter(), newTestMProcessor(ErrSkipProcessingData))
	require.NoError(t, err)
	assert.Equal(t, nil, me.ConsumeMetrics(context.Background(), pdatautil.MetricsFromOldInternalMetrics(testdataold.GenerateMetricDataEmpty())))
}

func TestNewLogsExporter(t *testing.T) {
	me, err := NewLogsProcessor(testCfg, exportertest.NewNopLogsExporter(), newTestLProcessor(nil))
	require.NoError(t, err)

	assert.NoError(t, me.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, me.ConsumeLogs(context.Background(), testdata.GenerateLogDataEmpty()))
	assert.NoError(t, me.Shutdown(context.Background()))
}

func TestNewLogsExporter_NilRequiredFields(t *testing.T) {
	_, err := NewLogsProcessor(testCfg, exportertest.NewNopLogsExporter(), nil)
	assert.Error(t, err)

	_, err = NewLogsProcessor(testCfg, nil, newTestLProcessor(nil))
	assert.Equal(t, componenterror.ErrNilNextConsumer, err)
}

func TestNewLogsExporter_ProcessLogError(t *testing.T) {
	want := errors.New("my_error")
	me, err := NewLogsProcessor(testCfg, exportertest.NewNopLogsExporter(), newTestLProcessor(want))
	require.NoError(t, err)
	assert.Equal(t, want, me.ConsumeLogs(context.Background(), testdata.GenerateLogDataEmpty()))
}

type testTProcessor struct {
	retError error
}

func newTestTProcessor(retError error) TProcessor {
	return &testTProcessor{retError: retError}
}

func (ttp *testTProcessor) ProcessTraces(_ context.Context, td pdata.Traces) (pdata.Traces, error) {
	return td, ttp.retError
}

type testMProcessor struct {
	retError error
}

func newTestMProcessor(retError error) MProcessor {
	return &testMProcessor{retError: retError}
}

func (tmp *testMProcessor) ProcessMetrics(_ context.Context, md pdata.Metrics) (pdata.Metrics, error) {
	return md, tmp.retError
}

type testLProcessor struct {
	retError error
}

func newTestLProcessor(retError error) LProcessor {
	return &testLProcessor{retError: retError}
}

func (tlp *testLProcessor) ProcessLogs(_ context.Context, ld pdata.Logs) (pdata.Logs, error) {
	return ld, tlp.retError
}
