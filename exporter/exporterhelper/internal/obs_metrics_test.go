// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pipeline"
)

func TestExporterObsMetricsAttributes(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	id := component.NewID(component.MustNewType("test"))
	om, err := newExporterObsMetrics(
		tt.NewTelemetrySettings(),
		id,
		pipeline.SignalTraces,
		[]attribute.KeyValue{attribute.String("transport", "grpc")},
	)
	require.NoError(t, err)
	t.Cleanup(om.Shutdown)

	om.RecordBatchSendSize(context.Background(), 5, 100)
	om.RecordSent(context.Background(), 5)

	batchMetric, err := tt.GetMetric("otelcol_exporter_queue_batch_send_size")
	require.NoError(t, err)
	batch := batchMetric.Data.(metricdata.Histogram[int64])
	require.Equal(t, attribute.NewSet(
		attribute.String(ExporterKey, id.String()),
	), batch.DataPoints[0].Attributes)

	sentMetric, err := tt.GetMetric("otelcol_exporter_sent_spans")
	require.NoError(t, err)
	sent := sentMetric.Data.(metricdata.Sum[int64])
	require.Equal(t, attribute.NewSet(
		attribute.String(ExporterKey, id.String()),
		attribute.String("transport", "grpc"),
	), sent.DataPoints[0].Attributes)
}
