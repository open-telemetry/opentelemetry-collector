// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package logs

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
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestExport(t *testing.T) {
	ld := testdata.GenerateLogs(1)
	req := plogotlp.NewExportRequestFromLogs(ld)

	logSink := new(consumertest.LogsSink)
	logClient := makeLogsServiceClient(t, logSink)
	resp, err := logClient.Export(context.Background(), req)
	require.NoError(t, err, "Failed to export trace: %v", err)
	require.NotNil(t, resp, "The response is missing")

	lds := logSink.AllLogs()
	require.Len(t, lds, 1)
	assert.EqualValues(t, ld, lds[0])
}

func TestExport_EmptyRequest(t *testing.T) {
	logSink := new(consumertest.LogsSink)

	logClient := makeLogsServiceClient(t, logSink)
	resp, err := logClient.Export(context.Background(), plogotlp.NewExportRequest())
	assert.NoError(t, err, "Failed to export trace: %v", err)
	assert.NotNil(t, resp, "The response is missing")
}

func TestExport_NonPermanentErrorConsumer(t *testing.T) {
	ld := testdata.GenerateLogs(1)
	req := plogotlp.NewExportRequestFromLogs(ld)

	logClient := makeLogsServiceClient(t, consumertest.NewErr(errors.New("my error")))
	resp, err := logClient.Export(context.Background(), req)
	assert.EqualError(t, err, "rpc error: code = Unavailable desc = my error")
	assert.IsType(t, status.Error(codes.Unknown, ""), err)
	assert.Equal(t, plogotlp.ExportResponse{}, resp)
}

func TestExport_PermanentErrorConsumer(t *testing.T) {
	ld := testdata.GenerateLogs(1)
	req := plogotlp.NewExportRequestFromLogs(ld)

	logClient := makeLogsServiceClient(t, consumertest.NewErr(consumererror.NewPermanent(errors.New("my error"))))
	resp, err := logClient.Export(context.Background(), req)
	assert.EqualError(t, err, "rpc error: code = Internal desc = Permanent error: my error")
	assert.IsType(t, status.Error(codes.Unknown, ""), err)
	assert.Equal(t, plogotlp.ExportResponse{}, resp)
}

func makeLogsServiceClient(t *testing.T, lc consumer.Logs) plogotlp.GRPCClient {
	addr := otlpReceiverOnGRPCServer(t, lc)
	cc, err := grpc.NewClient(addr.String(), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	require.NoError(t, err, "Failed to create the TraceServiceClient: %v", err)
	t.Cleanup(func() {
		require.NoError(t, cc.Close())
	})

	return plogotlp.NewGRPCClient(cc)
}

func otlpReceiverOnGRPCServer(t *testing.T, lc consumer.Logs) net.Addr {
	ln, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)

	t.Cleanup(func() {
		require.NoError(t, ln.Close())
	})

	set := receivertest.NewNopCreateSettings()
	set.ID = component.MustNewIDWithName("otlp", "log")
	obsreport, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             set.ID,
		Transport:              "grpc",
		ReceiverCreateSettings: set,
	})
	require.NoError(t, err)
	r := New(lc, obsreport)
	// Now run it as a gRPC server
	srv := grpc.NewServer()
	plogotlp.RegisterGRPCServer(srv, r)
	go func() {
		_ = srv.Serve(ln)
	}()

	return ln.Addr()
}
