// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/metadata"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestExport(t *testing.T) {
	md := testdata.GenerateMetrics(1)
	req := pmetricotlp.NewExportRequestFromMetrics(md)

	metricSink := new(consumertest.MetricsSink)
	metricsClient := makeMetricsServiceClient(t, metricSink)
	resp, err := metricsClient.Export(context.Background(), req)

	require.NoError(t, err, "Failed to export metrics: %v", err)
	require.NotNil(t, resp, "The response is missing")

	mds := metricSink.AllMetrics()
	require.Len(t, mds, 1)
	assert.Equal(t, md, mds[0])
}

func TestExport_EmptyRequest(t *testing.T) {
	metricSink := new(consumertest.MetricsSink)
	metricsClient := makeMetricsServiceClient(t, metricSink)
	resp, err := metricsClient.Export(context.Background(), pmetricotlp.NewExportRequest())
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestExport_InvalidUTF8(t *testing.T) {
	setRejectInvalidUTF8Gate(t, true)

	md := testdata.GenerateMetrics(1)
	dataPointCount := md.DataPointCount()
	md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Gauge().DataPoints().At(0).Attributes().PutStr("bad", string([]byte{0xff}))
	req := pmetricotlp.NewExportRequestFromMetrics(md)

	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	metricSink := new(consumertest.MetricsSink)
	metricsClient := makeMetricsServiceClientWithTelemetry(t, metricSink, tt)
	resp, err := metricsClient.Export(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.PartialSuccess().RejectedDataPoints())
	assert.Equal(t, invalidUTF8Message, resp.PartialSuccess().ErrorMessage())

	mds := metricSink.AllMetrics()
	require.Len(t, mds, 1)
	assert.Equal(t, dataPointCount-1, mds[0].DataPointCount())

	// Rejected data points must be visible in the receiver telemetry, not only in the response.
	assertReceiverSum(t, tt, "otelcol_receiver_refused_metric_points", 1)
	assertReceiverSum(t, tt, "otelcol_receiver_accepted_metric_points", int64(dataPointCount-1))
}

func TestExport_NonPermanentErrorConsumer(t *testing.T) {
	md := testdata.GenerateMetrics(1)
	req := pmetricotlp.NewExportRequestFromMetrics(md)

	metricsClient := makeMetricsServiceClient(t, consumertest.NewErr(errors.New("my error")))
	resp, err := metricsClient.Export(context.Background(), req)
	require.EqualError(t, err, "rpc error: code = Unavailable desc = my error")
	require.ErrorIs(t, err, status.Error(codes.Unavailable, "my error"))
	assert.Equal(t, pmetricotlp.ExportResponse{}, resp)
}

func TestExport_PermanentErrorConsumer(t *testing.T) {
	ld := testdata.GenerateMetrics(1)
	req := pmetricotlp.NewExportRequestFromMetrics(ld)

	metricsClient := makeMetricsServiceClient(t, consumertest.NewErr(consumererror.NewPermanent(errors.New("my error"))))
	resp, err := metricsClient.Export(context.Background(), req)
	require.EqualError(t, err, "rpc error: code = Internal desc = Permanent error: my error")
	require.ErrorIs(t, err, status.Error(codes.Internal, "Permanent error: my error"))
	assert.Equal(t, pmetricotlp.ExportResponse{}, resp)
}

func makeMetricsServiceClient(t *testing.T, mc consumer.Metrics) pmetricotlp.GRPCClient {
	return makeMetricsServiceClientWithTelemetry(t, mc, nil)
}

func makeMetricsServiceClientWithTelemetry(t *testing.T, mc consumer.Metrics, tt *componenttest.Telemetry) pmetricotlp.GRPCClient {
	addr := otlpReceiverOnGRPCServer(t, mc, tt)

	cc, err := grpc.NewClient(addr.String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "Failed to create the MetricsServiceClient: %v", err)
	t.Cleanup(func() {
		require.NoError(t, cc.Close())
	})

	return pmetricotlp.NewGRPCClient(cc)
}

// otlpReceiverOnGRPCServer starts the receiver on a gRPC server. When tt is not nil, the
// receiver telemetry is wired to it so tests can assert the emitted metrics.
func otlpReceiverOnGRPCServer(t *testing.T, mc consumer.Metrics, tt *componenttest.Telemetry) net.Addr {
	ln, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)

	t.Cleanup(func() {
		require.NoError(t, ln.Close())
	})

	set := receivertest.NewNopSettings(metadata.Type)
	if tt != nil {
		set.TelemetrySettings = tt.NewTelemetrySettings()
	}
	set.ID = component.MustNewIDWithName("otlp", "metrics")
	obsreport, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             set.ID,
		Transport:              "grpc",
		ReceiverCreateSettings: set,
	})
	require.NoError(t, err)
	r := New(mc, obsreport)
	// Now run it as a gRPC server
	srv := grpc.NewServer()
	pmetricotlp.RegisterGRPCServer(srv, r)
	go func() {
		_ = srv.Serve(ln)
	}()

	return ln.Addr()
}

func setRejectInvalidUTF8Gate(t *testing.T, enabled bool) {
	original := metadata.ReceiverOtlpRejectInvalidUTF8FeatureGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(metadata.ReceiverOtlpRejectInvalidUTF8FeatureGate.ID(), enabled))
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(metadata.ReceiverOtlpRejectInvalidUTF8FeatureGate.ID(), original))
	})
}

// assertReceiverSum checks the value of a receiver counter metric.
func assertReceiverSum(t *testing.T, tt *componenttest.Telemetry, name string, want int64) {
	t.Helper()
	m, err := tt.GetMetric(name)
	require.NoError(t, err)
	sum, ok := m.Data.(metricdata.Sum[int64])
	require.True(t, ok, "metric %s is not an int64 sum", name)
	var got int64
	for _, dp := range sum.DataPoints {
		got += dp.Value
	}
	assert.Equal(t, want, got, "metric %s", name)
}
