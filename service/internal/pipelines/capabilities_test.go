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

package pipelines

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/testdata"
)

func TestWrapLogs(t *testing.T) {
	sink := &consumertest.LogsSink{}
	require.Equal(t, consumer.Capabilities{MutatesData: false}, sink.Capabilities())

	wrap := wrapLogs(sink, consumer.Capabilities{MutatesData: true})
	assert.Equal(t, consumer.Capabilities{MutatesData: true}, wrap.Capabilities())

	assert.NoError(t, wrap.ConsumeLogs(context.Background(), testdata.GenerateLogsOneLogRecord()))
	assert.Len(t, sink.AllLogs(), 1)
	assert.Equal(t, testdata.GenerateLogsOneLogRecord(), sink.AllLogs()[0])
}

func TestWrapMetrics(t *testing.T) {
	sink := &consumertest.MetricsSink{}
	require.Equal(t, consumer.Capabilities{MutatesData: false}, sink.Capabilities())

	wrap := wrapMetrics(sink, consumer.Capabilities{MutatesData: true})
	assert.Equal(t, consumer.Capabilities{MutatesData: true}, wrap.Capabilities())

	assert.NoError(t, wrap.ConsumeMetrics(context.Background(), testdata.GenerateMetricsOneMetric()))
	assert.Len(t, sink.AllMetrics(), 1)
	assert.Equal(t, testdata.GenerateMetricsOneMetric(), sink.AllMetrics()[0])
}

func TestWrapTraces(t *testing.T) {
	sink := &consumertest.TracesSink{}
	require.Equal(t, consumer.Capabilities{MutatesData: false}, sink.Capabilities())

	wrap := wrapTraces(sink, consumer.Capabilities{MutatesData: true})
	assert.Equal(t, consumer.Capabilities{MutatesData: true}, wrap.Capabilities())

	assert.NoError(t, wrap.ConsumeTraces(context.Background(), testdata.GenerateTracesOneSpan()))
	assert.Len(t, sink.AllTraces(), 1)
	assert.Equal(t, testdata.GenerateTracesOneSpan(), sink.AllTraces()[0])
}
