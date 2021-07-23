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

var testLogsCfg = config.NewProcessorSettings(config.NewID(typeStr))

func TestNewLogsProcessor(t *testing.T) {
	lp, err := NewLogsProcessor(&testLogsCfg, consumertest.NewNop(), newTestLProcessor(nil))
	require.NoError(t, err)

	assert.True(t, lp.Capabilities().MutatesData)
	assert.NoError(t, lp.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, lp.ConsumeLogs(context.Background(), pdata.NewLogs()))
	assert.NoError(t, lp.Shutdown(context.Background()))
}

func TestNewLogsProcessor_WithOptions(t *testing.T) {
	want := errors.New("my_error")
	lp, err := NewLogsProcessor(&testLogsCfg, consumertest.NewNop(), newTestLProcessor(nil),
		WithStart(func(context.Context, component.Host) error { return want }),
		WithShutdown(func(context.Context) error { return want }),
		WithCapabilities(consumer.Capabilities{MutatesData: false}))
	assert.NoError(t, err)

	assert.Equal(t, want, lp.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, want, lp.Shutdown(context.Background()))
	assert.False(t, lp.Capabilities().MutatesData)
}

func TestNewLogsProcessor_NilRequiredFields(t *testing.T) {
	_, err := NewLogsProcessor(&testLogsCfg, consumertest.NewNop(), nil)
	assert.Error(t, err)

	_, err = NewLogsProcessor(&testLogsCfg, nil, newTestLProcessor(nil))
	assert.Equal(t, componenterror.ErrNilNextConsumer, err)
}

func TestNewLogsProcessor_ProcessLogError(t *testing.T) {
	want := errors.New("my_error")
	lp, err := NewLogsProcessor(&testLogsCfg, consumertest.NewNop(), newTestLProcessor(want))
	require.NoError(t, err)
	assert.Equal(t, want, lp.ConsumeLogs(context.Background(), pdata.NewLogs()))
}

func TestNewLogsProcessor_ProcessLogsErrSkipProcessingData(t *testing.T) {
	lp, err := NewLogsProcessor(&testLogsCfg, consumertest.NewNop(), newTestLProcessor(ErrSkipProcessingData))
	require.NoError(t, err)
	assert.Equal(t, nil, lp.ConsumeLogs(context.Background(), pdata.NewLogs()))
}

func newTestLProcessor(retError error) ProcessLogsFunc {
	return func(_ context.Context, ld pdata.Logs) (pdata.Logs, error) {
		return ld, retError
	}
}
