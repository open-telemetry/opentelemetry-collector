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
package loggingexporter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/testdata"
)

func TestLoggingTracesExporterNoErrors(t *testing.T) {
	lte, err := newTracesExporter(&config.ExporterSettings{}, "Debug", zap.NewNop())
	require.NotNil(t, lte)
	assert.NoError(t, err)

	assert.NoError(t, lte.ConsumeTraces(context.Background(), pdata.NewTraces()))
	assert.NoError(t, lte.ConsumeTraces(context.Background(), testdata.GenerateTracesTwoSpansSameResourceOneDifferent()))

	assert.NoError(t, lte.Shutdown(context.Background()))
}

func TestLoggingMetricsExporterNoErrors(t *testing.T) {
	lme, err := newMetricsExporter(&config.ExporterSettings{}, "DEBUG", zap.NewNop())
	require.NotNil(t, lme)
	assert.NoError(t, err)

	assert.NoError(t, lme.ConsumeMetrics(context.Background(), pdata.NewMetrics()))
	assert.NoError(t, lme.ConsumeMetrics(context.Background(), testdata.GeneratMetricsAllTypesWithSampleDatapoints()))
	assert.NoError(t, lme.ConsumeMetrics(context.Background(), testdata.GenerateMetricsAllTypesEmptyDataPoint()))
	assert.NoError(t, lme.ConsumeMetrics(context.Background(), testdata.GenerateMetricsMetricTypeInvalid()))

	assert.NoError(t, lme.Shutdown(context.Background()))
}

func TestLoggingLogsExporterNoErrors(t *testing.T) {
	lle, err := newLogsExporter(&config.ExporterSettings{}, "debug", zap.NewNop())
	require.NotNil(t, lle)
	assert.NoError(t, err)

	assert.NoError(t, lle.ConsumeLogs(context.Background(), pdata.NewLogs()))
	assert.NoError(t, lle.ConsumeLogs(context.Background(), testdata.GenerateLogsOneEmptyResourceLogs()))
	assert.NoError(t, lle.ConsumeLogs(context.Background(), testdata.GenerateLogsNoLogRecords()))
	assert.NoError(t, lle.ConsumeLogs(context.Background(), testdata.GenerateLogsOneEmptyLogRecord()))

	assert.NoError(t, lle.Shutdown(context.Background()))
}
