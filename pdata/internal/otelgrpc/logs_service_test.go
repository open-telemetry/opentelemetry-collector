// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gootlpcollectorlogs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/testing/protocmp"

	"go.opentelemetry.io/collector/pdata/internal"
)

func TestGrpcLogsE2E(t *testing.T) {
	fake := &fakeLogsServer{}
	cc := newServerAndClient(t, RegisterLogsServiceServer, LogsServiceServer(fake))
	logClient := NewLogsServiceClient(cc)

	resp, err := logClient.Export(context.Background(), internal.GenTestExportLogsServiceRequest(), grpc.WaitForReady(true))
	require.NoError(t, err)
	assert.Equal(t, internal.GenTestExportLogsServiceResponse(), resp)

	// Check that we get back what we initially sent.
	assert.Equal(t, internal.GenTestExportLogsServiceRequest(), fake.request)
}

func TestGrpcLogsE2EWithProtobuf(t *testing.T) {
	fake := &fakeLogsServer{}
	cc := newServerAndClient(t, RegisterLogsServiceServer, LogsServiceServer(fake))
	logClientPb := gootlpcollectorlogs.NewLogsServiceClient(cc)

	fakePb := &fakeLogsPbServer{}
	ccPb := newServerAndClient(t, gootlpcollectorlogs.RegisterLogsServiceServer, gootlpcollectorlogs.LogsServiceServer(fakePb))
	logClient := NewLogsServiceClient(ccPb)

	// Send the internal generated request to the protobuf generated server using the pdata client.
	resp, err := logClient.Export(context.Background(), internal.GenTestExportLogsServiceRequest(), grpc.WaitForReady(true))
	require.NoError(t, err)
	assert.Equal(t, internal.GenTestExportLogsServiceResponse(), resp)

	// Send the received protobuf generated request to the pdata generated server using the protobuf client.
	respPb, err := logClientPb.Export(context.Background(), fakePb.request, grpc.WaitForReady(true))
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(genLogsServiceResponsePb(), respPb, protocmp.Transform()))

	// Check that we get back what we initially sent.
	assert.Equal(t, internal.GenTestExportLogsServiceRequest(), fake.request)
}

type fakeLogsServer struct {
	UnimplementedLogsServiceServer
	request *internal.ExportLogsServiceRequest
}

func (f *fakeLogsServer) Export(_ context.Context, req *internal.ExportLogsServiceRequest) (*internal.ExportLogsServiceResponse, error) {
	f.request = req
	return internal.GenTestExportLogsServiceResponse(), nil
}

type fakeLogsPbServer struct {
	gootlpcollectorlogs.UnimplementedLogsServiceServer
	request *gootlpcollectorlogs.ExportLogsServiceRequest
}

func (f *fakeLogsPbServer) Export(_ context.Context, req *gootlpcollectorlogs.ExportLogsServiceRequest) (*gootlpcollectorlogs.ExportLogsServiceResponse, error) {
	f.request = req
	return genLogsServiceResponsePb(), nil
}

func genLogsServiceResponsePb() *gootlpcollectorlogs.ExportLogsServiceResponse {
	return &gootlpcollectorlogs.ExportLogsServiceResponse{
		PartialSuccess: &gootlpcollectorlogs.ExportLogsPartialSuccess{
			RejectedLogRecords: 13,
			ErrorMessage:       "test_errormessage",
		},
	}
}
