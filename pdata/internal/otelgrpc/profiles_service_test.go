// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gootlpcollectorprofiles "go.opentelemetry.io/proto/otlp/collector/profiles/v1development"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/testing/protocmp"

	"go.opentelemetry.io/collector/pdata/internal"
)

func TestGrpcProfilesE2E(t *testing.T) {
	fake := &fakeProfilesServer{}
	cc := newServerAndClient(t, RegisterProfilesServiceServer, ProfilesServiceServer(fake))
	logClient := NewProfilesServiceClient(cc)

	resp, err := logClient.Export(context.Background(), internal.GenTestExportProfilesServiceRequest(), grpc.WaitForReady(true))
	require.NoError(t, err)
	assert.Equal(t, internal.GenTestExportProfilesServiceResponse(), resp)

	// Check that we get back what we initially sent.
	assert.Equal(t, internal.GenTestExportProfilesServiceRequest(), fake.request)
}

func TestGrpcProfilesE2EWithProtobuf(t *testing.T) {
	fake := &fakeProfilesServer{}
	cc := newServerAndClient(t, RegisterProfilesServiceServer, ProfilesServiceServer(fake))
	logClientPb := gootlpcollectorprofiles.NewProfilesServiceClient(cc)

	fakePb := &fakeProfilesPbServer{}
	ccPb := newServerAndClient(t, gootlpcollectorprofiles.RegisterProfilesServiceServer, gootlpcollectorprofiles.ProfilesServiceServer(fakePb))
	logClient := NewProfilesServiceClient(ccPb)

	// Send the internal generated request to the protobuf generated server using the pdata client.
	resp, err := logClient.Export(context.Background(), internal.GenTestExportProfilesServiceRequest(), grpc.WaitForReady(true))
	require.NoError(t, err)
	assert.Equal(t, internal.GenTestExportProfilesServiceResponse(), resp)

	// Send the received protobuf generated request to the pdata generated server using the protobuf client.
	respPb, err := logClientPb.Export(context.Background(), fakePb.request, grpc.WaitForReady(true))
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(genProfilesServiceResponsePb(), respPb, protocmp.Transform()))

	// Check that we get back what we initially sent.
	assert.Equal(t, internal.GenTestExportProfilesServiceRequest(), fake.request)
}

type fakeProfilesServer struct {
	UnimplementedProfilesServiceServer
	request *internal.ExportProfilesServiceRequest
}

func (f *fakeProfilesServer) Export(_ context.Context, req *internal.ExportProfilesServiceRequest) (*internal.ExportProfilesServiceResponse, error) {
	f.request = req
	return internal.GenTestExportProfilesServiceResponse(), nil
}

type fakeProfilesPbServer struct {
	gootlpcollectorprofiles.UnimplementedProfilesServiceServer
	request *gootlpcollectorprofiles.ExportProfilesServiceRequest
}

func (f *fakeProfilesPbServer) Export(_ context.Context, req *gootlpcollectorprofiles.ExportProfilesServiceRequest) (*gootlpcollectorprofiles.ExportProfilesServiceResponse, error) {
	f.request = req
	return genProfilesServiceResponsePb(), nil
}

func genProfilesServiceResponsePb() *gootlpcollectorprofiles.ExportProfilesServiceResponse {
	return &gootlpcollectorprofiles.ExportProfilesServiceResponse{
		PartialSuccess: &gootlpcollectorprofiles.ExportProfilesPartialSuccess{
			RejectedProfiles: 13,
			ErrorMessage:     "test_errormessage",
		},
	}
}
