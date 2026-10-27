// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/pdata/internal/otlp"
)

func TestProfilesVersionCheckedBeforeDecode(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name   string
		values []string
		valid  bool
	}{
		{name: "absent", valid: otlp.ProfilesDevelopmentVersion == "1"},
		{name: "supported", values: []string{otlp.ProfilesDevelopmentVersion}, valid: true},
		{name: "unsupported", values: []string{otlp.ProfilesDevelopmentVersion + "0"}},
		{name: "repeated", values: []string{otlp.ProfilesDevelopmentVersion, otlp.ProfilesDevelopmentVersion}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := metadata.NewIncomingContext(t.Context(), metadata.MD{
				otlp.ProfilesDevelopmentVersionHeader: tt.values,
			})
			decoded := false
			decodeErr := errors.New("invalid protobuf")
			_, err := profilesServiceExportHandler(nil, ctx, func(any) error {
				decoded = true
				return decodeErr
			}, nil)
			assert.Equal(t, tt.valid, decoded)
			if tt.valid {
				require.ErrorIs(t, err, decodeErr)
			} else {
				assert.Equal(t, codes.InvalidArgument, status.Code(err))
			}
		})
	}
}
