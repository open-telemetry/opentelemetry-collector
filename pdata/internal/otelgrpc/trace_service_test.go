// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gootlpcollectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/testing/protocmp"

	"go.opentelemetry.io/collector/pdata/internal"
)

func TestGrpcTraceE2E(t *testing.T) {
	fake := &fakeTraceServer{}
	cc := newServerAndClient(t, RegisterTraceServiceServer, TraceServiceServer(fake))
	logClient := NewTraceServiceClient(cc)

	resp, err := logClient.Export(context.Background(), internal.GenTestExportTraceServiceRequest(), grpc.WaitForReady(true))
	require.NoError(t, err)
	assert.Equal(t, internal.GenTestExportTraceServiceResponse(), resp)

	// Check that we get back what we initially sent.
	assert.Equal(t, internal.GenTestExportTraceServiceRequest(), fake.request)
}

func TestGrpcTraceE2EWithProtobuf(t *testing.T) {
	fake := &fakeTraceServer{}
	cc := newServerAndClient(t, RegisterTraceServiceServer, TraceServiceServer(fake))
	logClientPb := gootlpcollectortrace.NewTraceServiceClient(cc)

	fakePb := &fakeTracePbServer{}
	ccPb := newServerAndClient(t, gootlpcollectortrace.RegisterTraceServiceServer, gootlpcollectortrace.TraceServiceServer(fakePb))
	logClient := NewTraceServiceClient(ccPb)

	// Send the internal generated request to the protobuf generated server using the pdata client.
	resp, err := logClient.Export(context.Background(), internal.GenTestExportTraceServiceRequest(), grpc.WaitForReady(true))
	require.NoError(t, err)
	assert.Equal(t, internal.GenTestExportTraceServiceResponse(), resp)

	// Send the received protobuf generated request to the pdata generated server using the protobuf client.
	respPb, err := logClientPb.Export(context.Background(), fakePb.request, grpc.WaitForReady(true))
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(genTraceServiceResponsePb(), respPb, protocmp.Transform()))

	// Check that we get back what we initially sent.
	assert.Equal(t, internal.GenTestExportTraceServiceRequest(), fake.request)
}

type fakeTraceServer struct {
	UnimplementedTraceServiceServer
	request *internal.ExportTraceServiceRequest
}

func (f *fakeTraceServer) Export(_ context.Context, req *internal.ExportTraceServiceRequest) (*internal.ExportTraceServiceResponse, error) {
	f.request = req
	return internal.GenTestExportTraceServiceResponse(), nil
}

type fakeTracePbServer struct {
	gootlpcollectortrace.UnimplementedTraceServiceServer
	request *gootlpcollectortrace.ExportTraceServiceRequest
}

func (f *fakeTracePbServer) Export(_ context.Context, req *gootlpcollectortrace.ExportTraceServiceRequest) (*gootlpcollectortrace.ExportTraceServiceResponse, error) {
	f.request = req
	return genTraceServiceResponsePb(), nil
}

func genTraceServiceResponsePb() *gootlpcollectortrace.ExportTraceServiceResponse {
	return &gootlpcollectortrace.ExportTraceServiceResponse{
		PartialSuccess: &gootlpcollectortrace.ExportTracePartialSuccess{
			RejectedSpans: 13,
			ErrorMessage:  "test_errormessage",
		},
	}
}
