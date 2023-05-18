// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package capabilityconsumer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/testdata"
)

func TestLogs(t *testing.T) {
	sink := &consumertest.LogsSink{}
	require.Equal(t, consumer.Capabilities{MutatesData: false}, sink.Capabilities())

	same := NewLogs(sink, consumer.Capabilities{MutatesData: false})
	assert.Same(t, sink, same)

	wrap := NewLogs(sink, consumer.Capabilities{MutatesData: true})
	assert.Equal(t, consumer.Capabilities{MutatesData: true}, wrap.Capabilities())

	assert.NoError(t, wrap.ConsumeLogs(context.Background(), testdata.GenerateLogs(1)))
	assert.Len(t, sink.AllLogs(), 1)
	assert.Equal(t, testdata.GenerateLogs(1), sink.AllLogs()[0])
}

func TestMetrics(t *testing.T) {
	sink := &consumertest.MetricsSink{}
	require.Equal(t, consumer.Capabilities{MutatesData: false}, sink.Capabilities())

	same := NewMetrics(sink, consumer.Capabilities{MutatesData: false})
	assert.Same(t, sink, same)

	wrap := NewMetrics(sink, consumer.Capabilities{MutatesData: true})
	assert.Equal(t, consumer.Capabilities{MutatesData: true}, wrap.Capabilities())

	assert.NoError(t, wrap.ConsumeMetrics(context.Background(), testdata.GenerateMetrics(1)))
	assert.Len(t, sink.AllMetrics(), 1)
	assert.Equal(t, testdata.GenerateMetrics(1), sink.AllMetrics()[0])
}

func TestTraces(t *testing.T) {
	sink := &consumertest.TracesSink{}
	require.Equal(t, consumer.Capabilities{MutatesData: false}, sink.Capabilities())

	same := NewTraces(sink, consumer.Capabilities{MutatesData: false})
	assert.Same(t, sink, same)

	wrap := NewTraces(sink, consumer.Capabilities{MutatesData: true})
	assert.Equal(t, consumer.Capabilities{MutatesData: true}, wrap.Capabilities())

	assert.NoError(t, wrap.ConsumeTraces(context.Background(), testdata.GenerateTraces(1)))
	assert.Len(t, sink.AllTraces(), 1)
	assert.Equal(t, testdata.GenerateTraces(1), sink.AllTraces()[0])
}
