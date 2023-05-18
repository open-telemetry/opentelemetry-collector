// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processorhelper

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor/processortest"
)

var testMetricsCfg = struct{}{}

func TestNewMetricsProcessor(t *testing.T) {
	mp, err := NewMetricsProcessor(context.Background(), processortest.NewNopCreateSettings(), &testMetricsCfg, consumertest.NewNop(), newTestMProcessor(nil))
	require.NoError(t, err)

	assert.True(t, mp.Capabilities().MutatesData)
	assert.NoError(t, mp.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, mp.ConsumeMetrics(context.Background(), pmetric.NewMetrics()))
	assert.NoError(t, mp.Shutdown(context.Background()))
}

func TestNewMetricsProcessor_WithOptions(t *testing.T) {
	want := errors.New("my_error")
	mp, err := NewMetricsProcessor(context.Background(), processortest.NewNopCreateSettings(), &testMetricsCfg, consumertest.NewNop(), newTestMProcessor(nil),
		WithStart(func(context.Context, component.Host) error { return want }),
		WithShutdown(func(context.Context) error { return want }),
		WithCapabilities(consumer.Capabilities{MutatesData: false}))
	assert.NoError(t, err)

	assert.Equal(t, want, mp.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, want, mp.Shutdown(context.Background()))
	assert.False(t, mp.Capabilities().MutatesData)
}

func TestNewMetricsProcessor_NilRequiredFields(t *testing.T) {
	_, err := NewMetricsProcessor(context.Background(), processortest.NewNopCreateSettings(), &testMetricsCfg, consumertest.NewNop(), nil)
	assert.Error(t, err)

	_, err = NewMetricsProcessor(context.Background(), processortest.NewNopCreateSettings(), &testMetricsCfg, nil, newTestMProcessor(nil))
	assert.Equal(t, component.ErrNilNextConsumer, err)
}

func TestNewMetricsProcessor_ProcessMetricsError(t *testing.T) {
	want := errors.New("my_error")
	mp, err := NewMetricsProcessor(context.Background(), processortest.NewNopCreateSettings(), &testMetricsCfg, consumertest.NewNop(), newTestMProcessor(want))
	require.NoError(t, err)
	assert.Equal(t, want, mp.ConsumeMetrics(context.Background(), pmetric.NewMetrics()))
}

func TestNewMetricsProcessor_ProcessMetricsErrSkipProcessingData(t *testing.T) {
	mp, err := NewMetricsProcessor(context.Background(), processortest.NewNopCreateSettings(), &testMetricsCfg, consumertest.NewNop(), newTestMProcessor(ErrSkipProcessingData))
	require.NoError(t, err)
	assert.Equal(t, nil, mp.ConsumeMetrics(context.Background(), pmetric.NewMetrics()))
}

func newTestMProcessor(retError error) ProcessMetricsFunc {
	return func(_ context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
		return md, retError
	}
}
