// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package persistentqueue

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

func TestSizerTypeToProto(t *testing.T) {
	tests := []struct {
		name           string
		sizerType      request.SizerType
		protoSizerType SizerType
		wantErr        bool
	}{
		{
			name:           "requests",
			sizerType:      request.SizerTypeRequests,
			protoSizerType: SizerType_REQUESTS,
		},
		{
			name:           "items",
			sizerType:      request.SizerTypeItems,
			protoSizerType: SizerType_ITEMS,
		},
		{
			name:           "bytes",
			sizerType:      request.SizerTypeBytes,
			protoSizerType: SizerType_BYTES,
		},
		{
			name:           "invalid",
			sizerType:      request.SizerType{},
			protoSizerType: -1,
			wantErr:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SizerTypeToProto(tt.sizerType)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.protoSizerType, got)
		})
	}
}

func TestSizerTypeFromProto(t *testing.T) {
	tests := []struct {
		name           string
		sizerType      request.SizerType
		protoSizerType SizerType
		wantErr        bool
	}{
		{
			name:           "requests",
			sizerType:      request.SizerTypeRequests,
			protoSizerType: SizerType_REQUESTS,
		},
		{
			name:           "items",
			sizerType:      request.SizerTypeItems,
			protoSizerType: SizerType_ITEMS,
		},
		{
			name:           "bytes",
			sizerType:      request.SizerTypeBytes,
			protoSizerType: SizerType_BYTES,
		},
		{
			name:           "invalid",
			sizerType:      request.SizerType{},
			protoSizerType: 3,
			wantErr:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SizerTypeFromProto(tt.protoSizerType)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.sizerType, got)
		})
	}
}

func TestSizerTypeRoundTrip(t *testing.T) {
	// Test that converting to int32 and back results in the same value
	sizerTypes := []request.SizerType{
		request.SizerTypeBytes,
		request.SizerTypeItems,
		request.SizerTypeRequests,
	}

	for _, originalType := range sizerTypes {
		t.Run(originalType.String(), func(t *testing.T) {
			// Convert to int32
			protoVal, err := SizerTypeToProto(originalType)
			require.NoError(t, err)

			// Convert back to SizerType
			convertedType, err := SizerTypeFromProto(protoVal)
			require.NoError(t, err)

			// Should be the same as original
			assert.Equal(t, originalType, convertedType)
		})
	}
}
