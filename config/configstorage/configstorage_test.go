// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configstorage

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/xextension/storage"
	"go.opentelemetry.io/collector/pipeline"
)

var testID = component.MustNewID("test")

type mockHost struct {
	extensions map[component.ID]component.Component
}

func (h *mockHost) GetExtensions() map[component.ID]component.Component {
	return h.extensions
}

type mockWrongType struct {
	component.StartFunc
	component.ShutdownFunc
}

type mockStorageExtension struct {
	component.StartFunc
	component.ShutdownFunc
}

func (m *mockStorageExtension) GetClient(_ context.Context, _ component.Kind, _ component.ID, _ string) (storage.Client, error) {
	return storage.NewNopClient(), nil
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr error
	}{
		{
			name: "valid",
			config: Config{
				ID: testID,
			},
			wantErr: nil,
		},
		{
			name:    "empty_id",
			config:  Config{},
			wantErr: errors.New("storage 'id' is required"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr != nil {
				require.EqualError(t, err, tt.wantErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfig_GetStorageClient(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		config     Config
		extensions map[component.ID]component.Component
		wantErr    error
	}{
		{
			name: "found_and_valid",
			config: Config{
				ID: testID,
			},
			extensions: map[component.ID]component.Component{
				testID: &mockStorageExtension{},
			},
			wantErr: nil,
		},
		{
			name: "extension_not_found",
			config: Config{
				ID: testID,
			},
			extensions: map[component.ID]component.Component{},
			wantErr:    errNoStorageClient,
		},
		{
			name: "extension_wrong_type",
			config: Config{
				ID: testID,
			},
			extensions: map[component.ID]component.Component{
				testID: mockWrongType{},
			},
			wantErr: errWrongExtensionType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := &mockHost{extensions: tt.extensions}
			value, err := tt.config.GetStorageClient(ctx, component.KindReceiver, host, testID, pipeline.SignalTraces.String())

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.NotNil(t, value)
			}
		})
	}
}
