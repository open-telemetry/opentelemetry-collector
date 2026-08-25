// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package trace

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
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/metadata"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestExport(t *testing.T) {
	td := testdata.GenerateTraces(1)
	req := ptraceotlp.NewExportRequestFromTraces(td)

	traceSink := new(consumertest.TracesSink)
	traceClient := makeTraceServiceClient(t, traceSink)
	resp, err := traceClient.Export(context.Background(), req)
	require.NoError(t, err, "Failed to export trace: %v", err)
	require.NotNil(t, resp, "The response is missing")

	require.Len(t, traceSink.AllTraces(), 1)
	assert.Equal(t, td, traceSink.AllTraces()[0])
}

func TestExport_EmptyRequest(t *testing.T) {
	traceSink := new(consumertest.TracesSink)
	traceClient := makeTraceServiceClient(t, traceSink)
	resp, err := traceClient.Export(context.Background(), ptraceotlp.NewExportRequest())
	require.NoError(t, err, "Failed to export trace: %v", err)
	assert.NotNil(t, resp, "The response is missing")
}

func TestExport_NonPermanentErrorConsumer(t *testing.T) {
	td := testdata.GenerateTraces(1)
	req := ptraceotlp.NewExportRequestFromTraces(td)

	traceClient := makeTraceServiceClient(t, consumertest.NewErr(errors.New("my error")))
	resp, err := traceClient.Export(context.Background(), req)
	require.EqualError(t, err, "rpc error: code = Unavailable desc = my error")
	require.ErrorIs(t, err, status.Error(codes.Unavailable, "my error"))
	assert.Equal(t, ptraceotlp.ExportResponse{}, resp)
}

func TestExport_PermanentErrorConsumer(t *testing.T) {
	ld := testdata.GenerateTraces(1)
	req := ptraceotlp.NewExportRequestFromTraces(ld)

	traceClient := makeTraceServiceClient(t, consumertest.NewErr(consumererror.NewPermanent(errors.New("my error"))))
	resp, err := traceClient.Export(context.Background(), req)
	require.EqualError(t, err, "rpc error: code = Internal desc = Permanent error: my error")
	require.ErrorIs(t, err, status.Error(codes.Internal, "Permanent error: my error"))
	assert.Equal(t, ptraceotlp.ExportResponse{}, resp)
}

func makeTraceServiceClient(t *testing.T, tc consumer.Traces) ptraceotlp.GRPCClient {
	addr := otlpReceiverOnGRPCServer(t, tc)
	cc, err := grpc.NewClient(addr.String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "Failed to create the TraceServiceClient: %v", err)
	t.Cleanup(func() {
		require.NoError(t, cc.Close())
	})

	return ptraceotlp.NewGRPCClient(cc)
}

func otlpReceiverOnGRPCServer(t *testing.T, tc consumer.Traces) net.Addr {
	ln, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)

	t.Cleanup(func() {
		require.NoError(t, ln.Close())
	})

	set := receivertest.NewNopSettings(metadata.Type)
	set.ID = component.MustNewIDWithName("otlp", "trace")
	obsreport, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             set.ID,
		Transport:              "grpc",
		ReceiverCreateSettings: set,
	})
	require.NoError(t, err)
	r := New(tc, obsreport)
	// Now run it as a gRPC server
	srv := grpc.NewServer()
	ptraceotlp.RegisterGRPCServer(srv, r)
	go func() {
		_ = srv.Serve(ln)
	}()

	return ln.Addr()
}
