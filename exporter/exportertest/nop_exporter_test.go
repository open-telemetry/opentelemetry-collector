// Copyright 2019, OpenTelemetry Authors
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
package exportertest

import (
	"context"
	"errors"
	"testing"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
)

func TestNopTraceExporter_NoErrors(t *testing.T) {
	nte := NewNopTraceExporter()
	td := consumerdata.TraceData{
		Spans: make([]*tracepb.Span, 7),
	}

	err := nte.ConsumeTraceData(context.Background(), td)
	require.Nil(t, err)
}

func TestNopTraceExporter_WithErrors(t *testing.T) {
	want := errors.New("MyError")
	nte := NewNopTraceExporter(WithReturnError(want))
	td := consumerdata.TraceData{
		Spans: make([]*tracepb.Span, 7),
	}

	require.Equal(t, want, nte.ConsumeTraceData(context.Background(), td))
}

func TestNopMetricsExporter_NoErrors(t *testing.T) {
	nme := NewNopMetricsExporter()
	md := consumerdata.MetricsData{
		Metrics: make([]*metricspb.Metric, 7),
	}
	err := nme.ConsumeMetricsData(context.Background(), md)

	require.Nil(t, err)
}

func TestNopMetricsExporter_WithErrors(t *testing.T) {
	want := errors.New("MyError")
	nme := NewNopMetricsExporter(WithReturnError(want))
	md := consumerdata.MetricsData{
		Metrics: make([]*metricspb.Metric, 7),
	}

	require.Equal(t, want, nme.ConsumeMetricsData(context.Background(), md))
}
