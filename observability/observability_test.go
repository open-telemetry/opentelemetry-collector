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

// observability_test instead of just observability to avoid dependency cycle between
// observability and observabilitytest
package observability_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector/observability"
	"github.com/open-telemetry/opentelemetry-collector/observability/observabilitytest"
)

const (
	receiverName = "fake_receiver"
	exporterName = "fake_exporter"
)

func TestTracePieplineRecordedMetrics(t *testing.T) {
	doneFn := observabilitytest.SetupRecordedMetricsTest()
	defer doneFn()

	receiverCtx := observability.ContextWithReceiverName(context.Background(), receiverName)
	observability.RecordMetricsForTraceReceiver(receiverCtx, 17, 13)
	exporterCtx := observability.ContextWithExporterName(receiverCtx, exporterName)
	observability.RecordMetricsForTraceExporter(exporterCtx, 27, 23)

	err := observabilitytest.CheckValueViewReceiverReceivedSpans(receiverName, 17)
	require.Nil(t, err, "When check receiver received spans")

	err = observabilitytest.CheckValueViewReceiverDroppedSpans(receiverName, 13)
	require.Nil(t, err, "When check receiver dropped spans")

	err = observabilitytest.CheckValueViewExporterReceivedSpans(receiverName, exporterName, 27)
	require.Nil(t, err, "When check exporter received spans")

	err = observabilitytest.CheckValueViewExporterDroppedSpans(receiverName, exporterName, 23)
	require.Nil(t, err, "When check exporter dropped spans")
}

func TestMetricsPieplineRecordedMetrics(t *testing.T) {
	doneFn := observabilitytest.SetupRecordedMetricsTest()
	defer doneFn()

	receiverCtx := observability.ContextWithReceiverName(context.Background(), receiverName)
	observability.RecordMetricsForMetricsReceiver(receiverCtx, 17, 13)
	exporterCtx := observability.ContextWithExporterName(receiverCtx, exporterName)
	observability.RecordMetricsForMetricsExporter(exporterCtx, 27, 23)

	err := observabilitytest.CheckValueViewReceiverReceivedTimeSeries(receiverName, 17)
	require.Nil(t, err, "When check receiver received timeseries")

	err = observabilitytest.CheckValueViewReceiverDroppedTimeSeries(receiverName, 13)
	require.Nil(t, err, "When check receiver dropped timeseries")

	err = observabilitytest.CheckValueViewExporterReceivedTimeSeries(receiverName, exporterName, 27)
	require.Nil(t, err, "When check exporter received timeseries")

	err = observabilitytest.CheckValueViewExporterDroppedTimeSeries(receiverName, exporterName, 23)
	require.Nil(t, err, "When check exporter dropped timeseries")
}
