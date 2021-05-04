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
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/testdata"
)

var testCfg = config.NewProcessorSettings(config.NewID(typeStr))

func TestDefaultOptions(t *testing.T) {
	bp := newBaseProcessor(config.NewID(typeStr))
	assert.True(t, bp.GetCapabilities().MutatesConsumedData)
	assert.NoError(t, bp.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, bp.Shutdown(context.Background()))
}

func TestWithOptions(t *testing.T) {
	want := errors.New("my_error")
	bp := newBaseProcessor(config.NewID(typeStr),
		WithStart(func(context.Context, component.Host) error { return want }),
		WithShutdown(func(context.Context) error { return want }),
		WithCapabilities(component.ProcessorCapabilities{MutatesConsumedData: false}))
	assert.Equal(t, want, bp.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, want, bp.Shutdown(context.Background()))
	assert.False(t, bp.GetCapabilities().MutatesConsumedData)
}

func TestNewTracesProcessor(t *testing.T) {
	me, err := NewTracesProcessor(&testCfg, consumertest.NewNop(), newTestTProcessor(nil))
	require.NoError(t, err)

	assert.NoError(t, me.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, me.ConsumeTraces(context.Background(), testdata.GenerateTraceDataEmpty()))
	assert.NoError(t, me.Shutdown(context.Background()))
}

func TestNewTracesProcessor_NilRequiredFields(t *testing.T) {
	_, err := NewTracesProcessor(&testCfg, consumertest.NewNop(), nil)
	assert.Error(t, err)

	_, err = NewTracesProcessor(&testCfg, nil, newTestTProcessor(nil))
	assert.Equal(t, componenterror.ErrNilNextConsumer, err)
}

func TestNewTracesProcessor_ProcessTraceError(t *testing.T) {
	want := errors.New("my_error")
	me, err := NewTracesProcessor(&testCfg, consumertest.NewNop(), newTestTProcessor(want))
	require.NoError(t, err)
	assert.Equal(t, want, me.ConsumeTraces(context.Background(), testdata.GenerateTraceDataEmpty()))
}

func TestNewMetricsProcessor(t *testing.T) {
	me, err := NewMetricsProcessor(&testCfg, consumertest.NewNop(), newTestMProcessor(nil))
	require.NoError(t, err)

	assert.NoError(t, me.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, me.ConsumeMetrics(context.Background(), testdata.GenerateMetricsEmpty()))
	assert.NoError(t, me.Shutdown(context.Background()))
}

func TestNewMetricsProcessor_NilRequiredFields(t *testing.T) {
	_, err := NewMetricsProcessor(&testCfg, consumertest.NewNop(), nil)
	assert.Error(t, err)

	_, err = NewMetricsProcessor(&testCfg, nil, newTestMProcessor(nil))
	assert.Equal(t, componenterror.ErrNilNextConsumer, err)
}

func TestNewMetricsProcessor_ProcessMetricsError(t *testing.T) {
	want := errors.New("my_error")
	me, err := NewMetricsProcessor(&testCfg, consumertest.NewNop(), newTestMProcessor(want))
	require.NoError(t, err)
	assert.Equal(t, want, me.ConsumeMetrics(context.Background(), testdata.GenerateMetricsEmpty()))
}

func TestNewMetricsProcessor_ProcessMetricsErrSkipProcessingData(t *testing.T) {
	me, err := NewMetricsProcessor(&testCfg, consumertest.NewNop(), newTestMProcessor(ErrSkipProcessingData))
	require.NoError(t, err)
	assert.Equal(t, nil, me.ConsumeMetrics(context.Background(), testdata.GenerateMetricsEmpty()))
}

func TestNewLogsProcessor(t *testing.T) {
	me, err := NewLogsProcessor(&testCfg, consumertest.NewNop(), newTestLProcessor(nil))
	require.NoError(t, err)

	assert.NoError(t, me.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, me.ConsumeLogs(context.Background(), testdata.GenerateLogDataEmpty()))
	assert.NoError(t, me.Shutdown(context.Background()))
}

func TestNewLogsProcessor_NilRequiredFields(t *testing.T) {
	_, err := NewLogsProcessor(&testCfg, consumertest.NewNop(), nil)
	assert.Error(t, err)

	_, err = NewLogsProcessor(&testCfg, nil, newTestLProcessor(nil))
	assert.Equal(t, componenterror.ErrNilNextConsumer, err)
}

func TestNewLogsProcessor_ProcessLogError(t *testing.T) {
	want := errors.New("my_error")
	me, err := NewLogsProcessor(&testCfg, consumertest.NewNop(), newTestLProcessor(want))
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
