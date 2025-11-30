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
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/consumertest"
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
	addr := otlpReceiverOnGRPCServer(t, mc)

	cc, err := grpc.NewClient(addr.String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "Failed to create the MetricsServiceClient: %v", err)
	t.Cleanup(func() {
		require.NoError(t, cc.Close())
	})

	return pmetricotlp.NewGRPCClient(cc)
}

func otlpReceiverOnGRPCServer(t *testing.T, mc consumer.Metrics) net.Addr {
	ln, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)

	t.Cleanup(func() {
		require.NoError(t, ln.Close())
	})

	set := receivertest.NewNopSettings(metadata.Type)
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
