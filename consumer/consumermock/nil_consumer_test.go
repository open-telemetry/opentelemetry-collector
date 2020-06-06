/*
 * Copyright The OpenTelemetry Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package consumermock

import (
	"context"
	"testing"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/internal/data"
)

func TestNopTraceConsumerOld(t *testing.T) {
	td := consumerdata.TraceData{
		Spans: make([]*tracepb.Span, 7),
	}
	require.NoError(t, Nil.ConsumeTraceData(context.Background(), td))
}

func TestNopMetricsConsumerOld(t *testing.T) {
	md := consumerdata.MetricsData{
		Metrics: make([]*metricspb.Metric, 7),
	}
	require.NoError(t, Nil.ConsumeMetricsData(context.Background(), md))
}

func TestNopTraceConsumer(t *testing.T) {
	require.NoError(t, Nil.ConsumeTraces(context.Background(), pdata.NewTraces()))
}

func TestNopMetricsConsumer(t *testing.T) {
	require.NoError(t, Nil.ConsumeMetrics(context.Background(), pdatautil.MetricsFromInternalMetrics(data.NewMetricData())))
}
