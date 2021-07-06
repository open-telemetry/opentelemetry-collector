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
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/model/pdata"
)

var testTracesCfg = config.NewProcessorSettings(config.NewID(typeStr))

func TestNewTracesProcessor(t *testing.T) {
	tp, err := NewTracesProcessor(&testTracesCfg, consumertest.NewNop(), newTestTProcessor(nil))
	require.NoError(t, err)

	assert.True(t, tp.Capabilities().MutatesData)
	assert.NoError(t, tp.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, tp.ConsumeTraces(context.Background(), pdata.NewTraces()))
	assert.NoError(t, tp.Shutdown(context.Background()))
}

func TestNewTracesProcessor_WithOptions(t *testing.T) {
	want := errors.New("my_error")
	tp, err := NewTracesProcessor(&testTracesCfg, consumertest.NewNop(), newTestTProcessor(nil),
		WithStart(func(context.Context, component.Host) error { return want }),
		WithShutdown(func(context.Context) error { return want }),
		WithCapabilities(consumer.Capabilities{MutatesData: false}))
	assert.NoError(t, err)

	assert.Equal(t, want, tp.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, want, tp.Shutdown(context.Background()))
	assert.False(t, tp.Capabilities().MutatesData)
}

func TestNewTracesProcessor_NilRequiredFields(t *testing.T) {
	_, err := NewTracesProcessor(&testTracesCfg, consumertest.NewNop(), nil)
	assert.Error(t, err)

	_, err = NewTracesProcessor(&testTracesCfg, nil, newTestTProcessor(nil))
	assert.Equal(t, componenterror.ErrNilNextConsumer, err)
}

func TestNewTracesProcessor_ProcessTraceError(t *testing.T) {
	want := errors.New("my_error")
	tp, err := NewTracesProcessor(&testTracesCfg, consumertest.NewNop(), newTestTProcessor(want))
	require.NoError(t, err)
	assert.Equal(t, want, tp.ConsumeTraces(context.Background(), pdata.NewTraces()))
}

func TestNewTracesProcessor_ProcessTracesErrSkipProcessingData(t *testing.T) {
	tp, err := NewTracesProcessor(&testTracesCfg, consumertest.NewNop(), newTestTProcessor(ErrSkipProcessingData))
	require.NoError(t, err)
	assert.Equal(t, nil, tp.ConsumeTraces(context.Background(), pdata.NewTraces()))
}

func newTestTProcessor(retError error) ProcessTracesFunc {
	return func(_ context.Context, td pdata.Traces) (pdata.Traces, error) {
		return td, retError
	}
}
