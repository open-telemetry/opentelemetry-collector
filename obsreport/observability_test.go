// Copyright The OpenTelemetry Authors
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

// obsreport_test instead of just obsreport to avoid dependency cycle between
// obsreport_test and obsreporttest
package obsreport_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
)

const (
	receiverName = "fake_receiver"
	exporterName = "fake_exporter"
)

func TestTracePieplineRecordedMetrics(t *testing.T) {
	doneFn := obsreporttest.SetupRecordedMetricsTest()
	defer doneFn()

	receiverCtx := obsreport.LegacyContextWithReceiverName(context.Background(), receiverName)
	obsreport.LegacyRecordMetricsForTraceReceiver(receiverCtx, 17, 13)
	exporterCtx := obsreport.LegacyContextWithExporterName(receiverCtx, exporterName)
	obsreport.LegacyRecordMetricsForTraceExporter(exporterCtx, 27, 23)

	err := obsreporttest.CheckValueViewReceiverReceivedSpans(receiverName, 17)
	require.Nil(t, err, "When check receiver received spans")

	err = obsreporttest.CheckValueViewReceiverDroppedSpans(receiverName, 13)
	require.Nil(t, err, "When check receiver dropped spans")

	err = obsreporttest.CheckValueViewExporterReceivedSpans(receiverName, exporterName, 27)
	require.Nil(t, err, "When check exporter received spans")

	err = obsreporttest.CheckValueViewExporterDroppedSpans(receiverName, exporterName, 23)
	require.Nil(t, err, "When check exporter dropped spans")
}

func TestMetricsPieplineRecordedMetrics(t *testing.T) {
	doneFn := obsreporttest.SetupRecordedMetricsTest()
	defer doneFn()

	receiverCtx := obsreport.LegacyContextWithReceiverName(context.Background(), receiverName)
	obsreport.LegacyRecordMetricsForMetricsReceiver(receiverCtx, 17, 13)
	exporterCtx := obsreport.LegacyContextWithExporterName(receiverCtx, exporterName)
	obsreport.LegacyRecordMetricsForMetricsExporter(exporterCtx, 27, 23)

	err := obsreporttest.CheckValueViewReceiverReceivedTimeSeries(receiverName, 17)
	require.Nil(t, err, "When check receiver received timeseries")

	err = obsreporttest.CheckValueViewReceiverDroppedTimeSeries(receiverName, 13)
	require.Nil(t, err, "When check receiver dropped timeseries")

	err = obsreporttest.CheckValueViewExporterReceivedTimeSeries(receiverName, exporterName, 27)
	require.Nil(t, err, "When check exporter received timeseries")

	err = obsreporttest.CheckValueViewExporterDroppedTimeSeries(receiverName, exporterName, 23)
	require.Nil(t, err, "When check exporter dropped timeseries")
}
