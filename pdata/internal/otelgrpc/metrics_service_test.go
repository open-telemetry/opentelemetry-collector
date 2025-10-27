// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gootlpcollectormetrics "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/testing/protocmp"

	"go.opentelemetry.io/collector/pdata/internal"
)

func TestGrpcMetricsE2E(t *testing.T) {
	fake := &fakeMetricsServer{}
	cc := newServerAndClient(t, RegisterMetricsServiceServer, MetricsServiceServer(fake))
	logClient := NewMetricsServiceClient(cc)

	resp, err := logClient.Export(context.Background(), internal.GenTestExportMetricsServiceRequest(), grpc.WaitForReady(true))
	require.NoError(t, err)
	assert.Equal(t, internal.GenTestExportMetricsServiceResponse(), resp)

	// Check that we get back what we initially sent.
	assert.Equal(t, internal.GenTestExportMetricsServiceRequest(), fake.request)
}

func TestGrpcMetricsE2EWithProtobuf(t *testing.T) {
	fake := &fakeMetricsServer{}
	cc := newServerAndClient(t, RegisterMetricsServiceServer, MetricsServiceServer(fake))
	logClientPb := gootlpcollectormetrics.NewMetricsServiceClient(cc)

	fakePb := &fakeMetricsPbServer{}
	ccPb := newServerAndClient(t, gootlpcollectormetrics.RegisterMetricsServiceServer, gootlpcollectormetrics.MetricsServiceServer(fakePb))
	logClient := NewMetricsServiceClient(ccPb)

	// Send the internal generated request to the protobuf generated server using the pdata client.
	resp, err := logClient.Export(context.Background(), internal.GenTestExportMetricsServiceRequest(), grpc.WaitForReady(true))
	require.NoError(t, err)
	assert.Equal(t, internal.GenTestExportMetricsServiceResponse(), resp)

	// Send the received protobuf generated request to the pdata generated server using the protobuf client.
	respPb, err := logClientPb.Export(context.Background(), fakePb.request, grpc.WaitForReady(true))
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(genMetricsServiceResponsePb(), respPb, protocmp.Transform()))

	// Check that we get back what we initially sent.
	assert.Equal(t, internal.GenTestExportMetricsServiceRequest(), fake.request)
}

type fakeMetricsServer struct {
	UnimplementedMetricsServiceServer
	request *internal.ExportMetricsServiceRequest
}

func (f *fakeMetricsServer) Export(_ context.Context, req *internal.ExportMetricsServiceRequest) (*internal.ExportMetricsServiceResponse, error) {
	f.request = req
	return internal.GenTestExportMetricsServiceResponse(), nil
}

type fakeMetricsPbServer struct {
	gootlpcollectormetrics.UnimplementedMetricsServiceServer
	request *gootlpcollectormetrics.ExportMetricsServiceRequest
}

func (f *fakeMetricsPbServer) Export(_ context.Context, req *gootlpcollectormetrics.ExportMetricsServiceRequest) (*gootlpcollectormetrics.ExportMetricsServiceResponse, error) {
	f.request = req
	return genMetricsServiceResponsePb(), nil
}

func genMetricsServiceResponsePb() *gootlpcollectormetrics.ExportMetricsServiceResponse {
	return &gootlpcollectormetrics.ExportMetricsServiceResponse{
		PartialSuccess: &gootlpcollectormetrics.ExportMetricsPartialSuccess{
			RejectedDataPoints: 13,
			ErrorMessage:       "test_errormessage",
		},
	}
}
