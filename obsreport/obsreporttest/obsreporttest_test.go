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

// observabilitytest_test instead of just obsreporttest to avoid dependency cycle between
// obsreporttest and other test packages.
package obsreporttest_test

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

func TestCheckValueViewTraceReceiverViews(t *testing.T) {
	doneFn := obsreporttest.SetupRecordedMetricsTest()
	defer doneFn()

	receiverCtx := obsreport.LegacyContextWithReceiverName(context.Background(), receiverName)
	obsreport.LegacyRecordMetricsForTraceReceiver(receiverCtx, 17, 13)
	// Test expected values.
	err := obsreporttest.CheckValueViewReceiverReceivedSpans(receiverName, 17)
	require.Nil(t, err, "When check receiver received spans")
	err = obsreporttest.CheckValueViewReceiverDroppedSpans(receiverName, 13)
	require.Nil(t, err, "When check receiver dropped spans")

	// Test unexpected tag values
	err = obsreporttest.CheckValueViewReceiverReceivedSpans(exporterName, 17)
	require.NotNil(t, err, "When check for unexpected tag value")
	err = obsreporttest.CheckValueViewReceiverDroppedSpans(exporterName, 13)
	require.NotNil(t, err, "When check for unexpected tag value")

	// Test unexpected recorded values
	err = obsreporttest.CheckValueViewReceiverReceivedSpans(receiverName, 13)
	require.NotNil(t, err, "When check for unexpected value")
	err = obsreporttest.CheckValueViewReceiverDroppedSpans(receiverName, 17)
	require.NotNil(t, err, "When check for unexpected value")
}

func TestCheckValueViewMetricsReceiverViews(t *testing.T) {
	doneFn := obsreporttest.SetupRecordedMetricsTest()
	defer doneFn()

	receiverCtx := obsreport.LegacyContextWithReceiverName(context.Background(), receiverName)
	obsreport.LegacyRecordMetricsForMetricsReceiver(receiverCtx, 17, 13)
	// Test expected values.
	err := obsreporttest.CheckValueViewReceiverReceivedTimeSeries(receiverName, 17)
	require.Nil(t, err, "When check receiver received timeseries")
	err = obsreporttest.CheckValueViewReceiverDroppedTimeSeries(receiverName, 13)
	require.Nil(t, err, "When check receiver dropped timeseries")

	// Test unexpected tag values
	err = obsreporttest.CheckValueViewReceiverReceivedTimeSeries(exporterName, 17)
	require.NotNil(t, err, "When check for unexpected tag value")
	err = obsreporttest.CheckValueViewReceiverDroppedTimeSeries(exporterName, 13)
	require.NotNil(t, err, "When check for unexpected tag value")

	// Test unexpected recorded values
	err = obsreporttest.CheckValueViewReceiverReceivedTimeSeries(receiverName, 13)
	require.NotNil(t, err, "When check for unexpected value")
	err = obsreporttest.CheckValueViewReceiverDroppedTimeSeries(receiverName, 17)
	require.NotNil(t, err, "When check for unexpected value")
}

func TestCheckValueViewTraceExporterViews(t *testing.T) {
	doneFn := obsreporttest.SetupRecordedMetricsTest()
	defer doneFn()

	receiverCtx := obsreport.LegacyContextWithReceiverName(context.Background(), receiverName)
	exporterCtx := obsreport.LegacyContextWithExporterName(receiverCtx, exporterName)
	obsreport.LegacyRecordMetricsForTraceExporter(exporterCtx, 17, 13)
	// Test expected values.
	err := obsreporttest.CheckValueViewExporterReceivedSpans(receiverName, exporterName, 17)
	require.Nil(t, err, "When check exporter received spans")
	err = obsreporttest.CheckValueViewExporterDroppedSpans(receiverName, exporterName, 13)
	require.Nil(t, err, "When check exporter dropped spans")

	// Test unexpected tag values
	err = obsreporttest.CheckValueViewExporterReceivedSpans(receiverName, receiverName, 17)
	require.NotNil(t, err, "When check for unexpected tag value")
	err = obsreporttest.CheckValueViewExporterDroppedSpans(receiverName, receiverName, 13)
	require.NotNil(t, err, "When check for unexpected tag value")

	// Test unexpected recorded values
	err = obsreporttest.CheckValueViewExporterReceivedSpans(receiverName, exporterName, 13)
	require.NotNil(t, err, "When check for unexpected value")
	err = obsreporttest.CheckValueViewExporterDroppedSpans(receiverName, exporterName, 17)
	require.NotNil(t, err, "When check for unexpected value")
}

func TestCheckValueViewMetricsExporterViews(t *testing.T) {
	doneFn := obsreporttest.SetupRecordedMetricsTest()
	defer doneFn()

	receiverCtx := obsreport.LegacyContextWithReceiverName(context.Background(), receiverName)
	exporterCtx := obsreport.LegacyContextWithExporterName(receiverCtx, exporterName)
	obsreport.LegacyRecordMetricsForMetricsExporter(exporterCtx, 17, 13)
	// Test expected values.
	err := obsreporttest.CheckValueViewExporterReceivedTimeSeries(receiverName, exporterName, 17)
	require.Nil(t, err, "When check exporter received timeseries")
	err = obsreporttest.CheckValueViewExporterDroppedTimeSeries(receiverName, exporterName, 13)
	require.Nil(t, err, "When check exporter dropped timeseries")

	// Test unexpected tag values
	err = obsreporttest.CheckValueViewExporterReceivedTimeSeries(receiverName, receiverName, 17)
	require.NotNil(t, err, "When check for unexpected tag value")
	err = obsreporttest.CheckValueViewExporterDroppedTimeSeries(receiverName, receiverName, 13)
	require.NotNil(t, err, "When check for unexpected tag value")

	// Test unexpected recorded values
	err = obsreporttest.CheckValueViewExporterReceivedTimeSeries(receiverName, exporterName, 13)
	require.NotNil(t, err, "When check for unexpected value")
	err = obsreporttest.CheckValueViewExporterDroppedTimeSeries(receiverName, exporterName, 17)
	require.NotNil(t, err, "When check for unexpected value")
}

func TestNoSetupCalled(t *testing.T) {
	receiverCtx := obsreport.LegacyContextWithReceiverName(context.Background(), receiverName)
	obsreport.LegacyRecordMetricsForTraceReceiver(receiverCtx, 17, 13)
	// Failed to check because views are not registered.
	err := obsreporttest.CheckValueViewReceiverReceivedSpans(receiverName, 17)
	require.NotNil(t, err, "When check with no views registered")
}
