// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configstorage

import (
	"context"
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

func TestID_MarshalText(t *testing.T) {
	tests := []struct {
		name string
		id   ID
		want string
	}{
		{
			name: "simple_type",
			id:   ID(component.MustNewID("file_storage")),
			want: "file_storage",
		},
		{
			name: "type_with_name",
			id:   ID(component.MustNewIDWithName("file_storage", "myname")),
			want: "file_storage/myname",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.id.MarshalText()
			require.NoError(t, err)
			require.Equal(t, tt.want, string(got))
		})
	}
}

func TestID_UnmarshalText(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    ID
		wantErr bool
	}{
		{
			name:  "simple_type",
			input: "file_storage",
			want:  ID(component.MustNewID("file_storage")),
		},
		{
			name:  "type_with_name",
			input: "file_storage/myname",
			want:  ID(component.MustNewIDWithName("file_storage", "myname")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var id ID
			err := id.UnmarshalText([]byte(tt.input))
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, id)
			}
		})
	}
}

func TestID_MarshalUnmarshalRoundTrip(t *testing.T) {
	original := ID(component.MustNewIDWithName("file_storage", "myname"))

	text, err := original.MarshalText()
	require.NoError(t, err)

	var restored ID
	err = restored.UnmarshalText(text)
	require.NoError(t, err)
	require.Equal(t, original, restored)
}

func TestConfig_GetStorageClient(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		config     ID
		extensions map[component.ID]component.Component
		wantErr    error
	}{
		{
			name:   "found_and_valid",
			config: ID(testID),
			extensions: map[component.ID]component.Component{
				testID: &mockStorageExtension{},
			},
			wantErr: nil,
		},
		{
			name:       "extension_not_found",
			config:     ID(testID),
			extensions: map[component.ID]component.Component{},
			wantErr:    errNoStorageClient,
		},
		{
			name:   "extension_wrong_type",
			config: ID(testID),
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
