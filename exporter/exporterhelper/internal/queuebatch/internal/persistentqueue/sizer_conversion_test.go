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
		protoSizerType int32
		wantErr        bool
	}{
		{
			name:           "requests",
			sizerType:      request.SizerTypeRequests,
			protoSizerType: SizerTypeRequests,
		},
		{
			name:           "items",
			sizerType:      request.SizerTypeItems,
			protoSizerType: SizerTypeItems,
		},
		{
			name:           "bytes",
			sizerType:      request.SizerTypeBytes,
			protoSizerType: SizerTypeBytes,
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
		protoSizerType int32
		wantErr        bool
	}{
		{
			name:           "requests",
			sizerType:      request.SizerTypeRequests,
			protoSizerType: SizerTypeRequests,
		},
		{
			name:           "items",
			sizerType:      request.SizerTypeItems,
			protoSizerType: SizerTypeItems,
		},
		{
			name:           "bytes",
			sizerType:      request.SizerTypeBytes,
			protoSizerType: SizerTypeBytes,
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
