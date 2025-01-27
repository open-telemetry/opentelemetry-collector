// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proctelemetry

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"go.opentelemetry.io/collector/service/internal/metadatatest"
)

func TestProcessTelemetry(t *testing.T) {
	tel := metadatatest.SetupTelemetry()
	require.NoError(t, RegisterProcessMetrics(tel.NewTelemetrySettings()))

	metadatatest.AssertEqualProcessUptime(t, tel.Telemetry,
		[]metricdata.DataPoint[float64]{{}}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())

	metadatatest.AssertEqualProcessRuntimeHeapAllocBytes(t, tel.Telemetry,
		[]metricdata.DataPoint[int64]{{}}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())

	metadatatest.AssertEqualProcessRuntimeTotalAllocBytes(t, tel.Telemetry,
		[]metricdata.DataPoint[int64]{{}}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())

	metadatatest.AssertEqualProcessRuntimeTotalSysMemoryBytes(t, tel.Telemetry,
		[]metricdata.DataPoint[int64]{{}}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())

	metadatatest.AssertEqualProcessCPUSeconds(t, tel.Telemetry,
		[]metricdata.DataPoint[float64]{{}}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())

	metadatatest.AssertEqualProcessMemoryRss(t, tel.Telemetry,
		[]metricdata.DataPoint[int64]{{}}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())
}
