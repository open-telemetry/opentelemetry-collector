// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofileotlp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"go.opentelemetry.io/collector/pdata/internal"
)

func TestValidateDevelopmentVersion(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name   string
		values []string
		valid  bool
	}{
		{name: "absent", valid: DevelopmentVersion == "1"},
		{name: "supported", values: []string{DevelopmentVersion}, valid: true},
		{name: "unsupported", values: []string{DevelopmentVersion + "0"}},
		{name: "empty", values: []string{""}},
		{name: "zero", values: []string{"0"}},
		{name: "leading_zero", values: []string{"01"}},
		{name: "plus_sign", values: []string{"+1"}},
		{name: "whitespace", values: []string{" 1"}},
		{name: "repeated", values: []string{DevelopmentVersion, DevelopmentVersion}},
		{name: "comma_separated", values: []string{"1, 1"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateDevelopmentVersion(tt.values)
			if tt.valid {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, DevelopmentVersionHeader)
			}
		})
	}
}

func TestGRPCClientDevelopmentVersion(t *testing.T) {
	t.Parallel()
	md := metadata.Pairs(DevelopmentVersionHeader, "2", "other-header", "preserved")
	ctx := metadata.NewOutgoingContext(t.Context(), md)
	ctx = metadata.AppendToOutgoingContext(ctx, DevelopmentVersionHeader, "3")
	client := &grpcClient{rawClient: &versionCheckingClient{t: t}}
	_, err := client.Export(ctx, NewExportRequest())
	require.NoError(t, err)
	assert.Equal(t, []string{"2"}, md.Get(DevelopmentVersionHeader), "the caller's metadata must not be mutated")
	outgoing, _ := metadata.FromOutgoingContext(ctx)
	assert.Equal(t, []string{"2", "3"}, outgoing.Get(DevelopmentVersionHeader))
}

type versionCheckingClient struct {
	t *testing.T
}

func (c *versionCheckingClient) Export(ctx context.Context, _ *internal.ExportProfilesServiceRequest, _ ...grpc.CallOption) (*internal.ExportProfilesServiceResponse, error) {
	md, _ := metadata.FromOutgoingContext(ctx)
	assert.Equal(c.t, []string{DevelopmentVersion}, md.Get(DevelopmentVersionHeader))
	assert.Equal(c.t, []string{"preserved"}, md.Get("other-header"))
	return internal.NewExportProfilesServiceResponse(), nil
}
