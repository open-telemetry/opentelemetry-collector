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

// observabilitytest_test instead of just observabilitytest to avoid dependency cycle between
// observabilitytest and other test packages.
package observabilitytest_test

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

func TestCheckValueViewTraceReceiverViews(t *testing.T) {
	doneFn := observabilitytest.SetupRecordedMetricsTest()
	defer doneFn()

	receiverCtx := observability.ContextWithReceiverName(context.Background(), receiverName)
	observability.RecordMetricsForTraceReceiver(receiverCtx, 17, 13)
	// Test expected values.
	err := observabilitytest.CheckValueViewReceiverReceivedSpans(receiverName, 17)
	require.Nil(t, err, "When check receiver received spans")
	err = observabilitytest.CheckValueViewReceiverDroppedSpans(receiverName, 13)
	require.Nil(t, err, "When check receiver dropped spans")

	// Test unexpected tag values
	err = observabilitytest.CheckValueViewReceiverReceivedSpans(exporterName, 17)
	require.NotNil(t, err, "When check for unexpected tag value")
	err = observabilitytest.CheckValueViewReceiverDroppedSpans(exporterName, 13)
	require.NotNil(t, err, "When check for unexpected tag value")

	// Test unexpected recorded values
	err = observabilitytest.CheckValueViewReceiverReceivedSpans(receiverName, 13)
	require.NotNil(t, err, "When check for unexpected value")
	err = observabilitytest.CheckValueViewReceiverDroppedSpans(receiverName, 17)
	require.NotNil(t, err, "When check for unexpected value")
}

func TestCheckValueViewMetricsReceiverViews(t *testing.T) {
	doneFn := observabilitytest.SetupRecordedMetricsTest()
	defer doneFn()

	receiverCtx := observability.ContextWithReceiverName(context.Background(), receiverName)
	observability.RecordMetricsForMetricsReceiver(receiverCtx, 17, 13)
	// Test expected values.
	err := observabilitytest.CheckValueViewReceiverReceivedTimeSeries(receiverName, 17)
	require.Nil(t, err, "When check receiver received timeseries")
	err = observabilitytest.CheckValueViewReceiverDroppedTimeSeries(receiverName, 13)
	require.Nil(t, err, "When check receiver dropped timeseries")

	// Test unexpected tag values
	err = observabilitytest.CheckValueViewReceiverReceivedTimeSeries(exporterName, 17)
	require.NotNil(t, err, "When check for unexpected tag value")
	err = observabilitytest.CheckValueViewReceiverDroppedTimeSeries(exporterName, 13)
	require.NotNil(t, err, "When check for unexpected tag value")

	// Test unexpected recorded values
	err = observabilitytest.CheckValueViewReceiverReceivedTimeSeries(receiverName, 13)
	require.NotNil(t, err, "When check for unexpected value")
	err = observabilitytest.CheckValueViewReceiverDroppedTimeSeries(receiverName, 17)
	require.NotNil(t, err, "When check for unexpected value")
}

func TestCheckValueViewTraceExporterViews(t *testing.T) {
	doneFn := observabilitytest.SetupRecordedMetricsTest()
	defer doneFn()

	receiverCtx := observability.ContextWithReceiverName(context.Background(), receiverName)
	exporterCtx := observability.ContextWithExporterName(receiverCtx, exporterName)
	observability.RecordMetricsForTraceExporter(exporterCtx, 17, 13)
	// Test expected values.
	err := observabilitytest.CheckValueViewExporterReceivedSpans(receiverName, exporterName, 17)
	require.Nil(t, err, "When check exporter received spans")
	err = observabilitytest.CheckValueViewExporterDroppedSpans(receiverName, exporterName, 13)
	require.Nil(t, err, "When check exporter dropped spans")

	// Test unexpected tag values
	err = observabilitytest.CheckValueViewExporterReceivedSpans(receiverName, receiverName, 17)
	require.NotNil(t, err, "When check for unexpected tag value")
	err = observabilitytest.CheckValueViewExporterDroppedSpans(receiverName, receiverName, 13)
	require.NotNil(t, err, "When check for unexpected tag value")

	// Test unexpected recorded values
	err = observabilitytest.CheckValueViewExporterReceivedSpans(receiverName, exporterName, 13)
	require.NotNil(t, err, "When check for unexpected value")
	err = observabilitytest.CheckValueViewExporterDroppedSpans(receiverName, exporterName, 17)
	require.NotNil(t, err, "When check for unexpected value")
}

func TestCheckValueViewMetricsExporterViews(t *testing.T) {
	doneFn := observabilitytest.SetupRecordedMetricsTest()
	defer doneFn()

	receiverCtx := observability.ContextWithReceiverName(context.Background(), receiverName)
	exporterCtx := observability.ContextWithExporterName(receiverCtx, exporterName)
	observability.RecordMetricsForMetricsExporter(exporterCtx, 17, 13)
	// Test expected values.
	err := observabilitytest.CheckValueViewExporterReceivedTimeSeries(receiverName, exporterName, 17)
	require.Nil(t, err, "When check exporter received timeseries")
	err = observabilitytest.CheckValueViewExporterDroppedTimeSeries(receiverName, exporterName, 13)
	require.Nil(t, err, "When check exporter dropped timeseries")

	// Test unexpected tag values
	err = observabilitytest.CheckValueViewExporterReceivedTimeSeries(receiverName, receiverName, 17)
	require.NotNil(t, err, "When check for unexpected tag value")
	err = observabilitytest.CheckValueViewExporterDroppedTimeSeries(receiverName, receiverName, 13)
	require.NotNil(t, err, "When check for unexpected tag value")

	// Test unexpected recorded values
	err = observabilitytest.CheckValueViewExporterReceivedTimeSeries(receiverName, exporterName, 13)
	require.NotNil(t, err, "When check for unexpected value")
	err = observabilitytest.CheckValueViewExporterDroppedTimeSeries(receiverName, exporterName, 17)
	require.NotNil(t, err, "When check for unexpected value")
}

func TestNoSetupCalled(t *testing.T) {
	receiverCtx := observability.ContextWithReceiverName(context.Background(), receiverName)
	observability.RecordMetricsForTraceReceiver(receiverCtx, 17, 13)
	// Failed to check because views are not registered.
	err := observabilitytest.CheckValueViewReceiverReceivedSpans(receiverName, 17)
	require.NotNil(t, err, "When check with no views registered")
}
