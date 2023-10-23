// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package profiles

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/pdata/pprofile/pprofileotlp"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestExport(t *testing.T) {
	ld := testdata.GenerateLogs(1)
	req := pprofileotlp.NewExportRequestFromLogs(ld)

	profileSink := new(consumertest.LogsSink)
	profileClient := makeLogsServiceClient(t, profileSink)
	resp, err := profileClient.Export(context.Background(), req)
	require.NoError(t, err, "Failed to export trace: %v", err)
	require.NotNil(t, resp, "The response is missing")

	lds := profileSink.AllLogs()
	require.Len(t, lds, 1)
	assert.EqualValues(t, ld, lds[0])
}

func TestExport_EmptyRequest(t *testing.T) {
	profileSink := new(consumertest.LogsSink)

	profileClient := makeLogsServiceClient(t, profileSink)
	resp, err := profileClient.Export(context.Background(), pprofileotlp.NewExportRequest())
	assert.NoError(t, err, "Failed to export trace: %v", err)
	assert.NotNil(t, resp, "The response is missing")
}

func TestExport_ErrorConsumer(t *testing.T) {
	ld := testdata.GenerateLogs(1)
	req := pprofileotlp.NewExportRequestFromLogs(ld)

	profileClient := makeLogsServiceClient(t, consumertest.NewErr(errors.New("my error")))
	resp, err := profileClient.Export(context.Background(), req)
	assert.EqualError(t, err, "rpc error: code = Unknown desc = my error")
	assert.Equal(t, pprofileotlp.ExportResponse{}, resp)
}

func makeLogsServiceClient(t *testing.T, lc consumer.Logs) pprofileotlp.GRPCClient {
	addr := otlpReceiverOnGRPCServer(t, lc)
	cc, err := grpc.Dial(addr.String(), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	require.NoError(t, err, "Failed to create the TraceServiceClient: %v", err)
	t.Cleanup(func() {
		require.NoError(t, cc.Close())
	})

	return pprofileotlp.NewGRPCClient(cc)
}

func otlpReceiverOnGRPCServer(t *testing.T, lc consumer.Logs) net.Addr {
	ln, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)

	t.Cleanup(func() {
		require.NoError(t, ln.Close())
	})

	set := receivertest.NewNopCreateSettings()
	set.ID = component.NewIDWithName("otlp", "profile")
	obsrecv, err := obsreport.NewReceiver(obsreport.ReceiverSettings{
		ReceiverID:             set.ID,
		Transport:              "grpc",
		ReceiverCreateSettings: set,
	})
	require.NoError(t, err)
	r := New(lc, obsrecv)
	// Now run it as a gRPC server
	srv := grpc.NewServer()
	pprofileotlp.RegisterGRPCServer(srv, r)
	go func() {
		_ = srv.Serve(ln)
	}()

	return ln.Addr()
}
