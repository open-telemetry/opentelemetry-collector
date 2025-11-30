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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pdata/pprofile/pprofileotlp"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestExport(t *testing.T) {
	td := testdata.GenerateProfiles(1)
	req := pprofileotlp.NewExportRequestFromProfiles(td)

	profileSink := new(consumertest.ProfilesSink)
	profileClient := makeProfileServiceClient(t, profileSink)
	resp, err := profileClient.Export(context.Background(), req)
	require.NoError(t, err, "Failed to export profile: %v", err)
	require.NotNil(t, resp, "The response is missing")

	require.Len(t, profileSink.AllProfiles(), 1)
	assert.Equal(t, td, profileSink.AllProfiles()[0])
}

func TestExport_EmptyRequest(t *testing.T) {
	profileSink := new(consumertest.ProfilesSink)
	profileClient := makeProfileServiceClient(t, profileSink)
	resp, err := profileClient.Export(context.Background(), pprofileotlp.NewExportRequest())
	require.NoError(t, err, "Failed to export profile: %v", err)
	assert.NotNil(t, resp, "The response is missing")
}

func TestExport_NonPermanentErrorConsumer(t *testing.T) {
	td := testdata.GenerateProfiles(1)
	req := pprofileotlp.NewExportRequestFromProfiles(td)

	profileClient := makeProfileServiceClient(t, consumertest.NewErr(errors.New("my error")))
	resp, err := profileClient.Export(context.Background(), req)
	require.EqualError(t, err, "rpc error: code = Unavailable desc = my error")
	require.ErrorIs(t, err, status.Error(codes.Unavailable, "my error"))
	assert.Equal(t, pprofileotlp.ExportResponse{}, resp)
}

func TestExport_PermanentErrorConsumer(t *testing.T) {
	ld := testdata.GenerateProfiles(1)
	req := pprofileotlp.NewExportRequestFromProfiles(ld)

	profileClient := makeProfileServiceClient(t, consumertest.NewErr(consumererror.NewPermanent(errors.New("my error"))))
	resp, err := profileClient.Export(context.Background(), req)
	require.EqualError(t, err, "rpc error: code = Internal desc = Permanent error: my error")
	require.ErrorIs(t, err, status.Error(codes.Internal, "Permanent error: my error"))
	assert.Equal(t, pprofileotlp.ExportResponse{}, resp)
}

func makeProfileServiceClient(t *testing.T, tc xconsumer.Profiles) pprofileotlp.GRPCClient {
	addr := otlpReceiverOnGRPCServer(t, tc)
	cc, err := grpc.NewClient(addr.String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "Failed to create the profileServiceClient: %v", err)
	t.Cleanup(func() {
		require.NoError(t, cc.Close())
	})

	return pprofileotlp.NewGRPCClient(cc)
}

func otlpReceiverOnGRPCServer(t *testing.T, tc xconsumer.Profiles) net.Addr {
	ln, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)

	t.Cleanup(func() {
		require.NoError(t, ln.Close())
	})

	r := New(tc)
	// Now run it as a gRPC server
	srv := grpc.NewServer()
	pprofileotlp.RegisterGRPCServer(srv, r)
	go func() {
		_ = srv.Serve(ln)
	}()

	return ln.Addr()
}
