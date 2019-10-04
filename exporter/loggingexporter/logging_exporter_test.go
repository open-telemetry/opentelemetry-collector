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
package loggingexporter

import (
	"context"
	"testing"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
)

func TestLoggingTraceExporterNoErrors(t *testing.T) {
	lte, err := NewTraceExporter(&configmodels.ExporterSettings{}, zap.NewNop())
	require.NotNil(t, lte)
	assert.NoError(t, err)

	td := consumerdata.TraceData{
		Spans: make([]*tracepb.Span, 7),
	}
	assert.NoError(t, lte.ConsumeTraceData(context.Background(), td))
	assert.NoError(t, lte.Shutdown())
}

func TestLoggingMetricsExporterNoErrors(t *testing.T) {
	lme, err := NewMetricsExporter(&configmodels.ExporterSettings{}, zap.NewNop())
	require.NotNil(t, lme)
	assert.NoError(t, err)

	md := consumerdata.MetricsData{
		Metrics: make([]*metricspb.Metric, 7),
	}
	assert.NoError(t, lme.ConsumeMetricsData(context.Background(), md))
	assert.NoError(t, lme.Shutdown())
}
