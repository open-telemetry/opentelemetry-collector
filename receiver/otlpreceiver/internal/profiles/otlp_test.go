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
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pdata/pprofile/pprofileotlp"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/metadata"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.opentelemetry.io/collector/receiver/receivertest"
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
	assert.Equal(t, td.ResourceProfiles(), profileSink.AllProfiles()[0].ResourceProfiles())
}

func TestExport_EmptyRequest(t *testing.T) {
	profileSink := new(consumertest.ProfilesSink)
	profileClient := makeProfileServiceClient(t, profileSink)
	resp, err := profileClient.Export(context.Background(), pprofileotlp.NewExportRequest())
	require.NoError(t, err, "Failed to export profile: %v", err)
	assert.NotNil(t, resp, "The response is missing")
}

func TestExport_InvalidUTF8(t *testing.T) {
	setRejectInvalidUTF8Gate(t, true)

	td := testdata.GenerateProfiles(2)
	profileCount := td.ProfileCount()
	sampleCount := td.SampleCount()
	td.Dictionary().StringTable().Append(string([]byte{0xff}))
	req := pprofileotlp.NewExportRequestFromProfiles(td)

	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	profileSink := new(consumertest.ProfilesSink)
	profileClient := makeProfileServiceClientWithTelemetry(t, profileSink, tt)
	resp, err := profileClient.Export(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, int64(profileCount), resp.PartialSuccess().RejectedProfiles())
	assert.Equal(t, invalidUTF8Message, resp.PartialSuccess().ErrorMessage())
	assert.Empty(t, profileSink.AllProfiles())

	// The response counts rejected profiles, the receiver telemetry counts rejected samples.
	assertReceiverSum(t, tt, "otelcol_receiver_refused_profile_samples", int64(sampleCount))
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
	return makeProfileServiceClientWithTelemetry(t, tc, nil)
}

func makeProfileServiceClientWithTelemetry(t *testing.T, tc xconsumer.Profiles, tt *componenttest.Telemetry) pprofileotlp.GRPCClient {
	addr := otlpReceiverOnGRPCServer(t, tc, tt)
	cc, err := grpc.NewClient(addr.String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "Failed to create the profileServiceClient: %v", err)
	t.Cleanup(func() {
		require.NoError(t, cc.Close())
	})

	return pprofileotlp.NewGRPCClient(cc)
}

// otlpReceiverOnGRPCServer starts the receiver on a gRPC server. When tt is not nil, the
// receiver telemetry is wired to it so tests can assert the emitted metrics.
func otlpReceiverOnGRPCServer(t *testing.T, tc xconsumer.Profiles, tt *componenttest.Telemetry) net.Addr {
	ln, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)

	t.Cleanup(func() {
		require.NoError(t, ln.Close())
	})

	set := receivertest.NewNopSettings(metadata.Type)
	if tt != nil {
		set.TelemetrySettings = tt.NewTelemetrySettings()
	}
	set.ID = component.MustNewIDWithName("otlp", "profiles")
	obsreport, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             set.ID,
		Transport:              "grpc",
		ReceiverCreateSettings: set,
	})
	require.NoError(t, err)

	r := New(tc, obsreport)
	// Now run it as a gRPC server
	srv := grpc.NewServer()
	pprofileotlp.RegisterGRPCServer(srv, r)
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
