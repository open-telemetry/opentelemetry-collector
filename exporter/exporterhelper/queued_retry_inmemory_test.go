// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License

package exporterhelper

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/extension/experimental/storage"
)

type mockHost struct {
	component.Host
	ext map[config.ComponentID]component.Extension
}

func (nh *mockHost) GetExtensions() map[config.ComponentID]component.Extension {
	return nh.ext
}

type mockStorageExtension struct{}

func (mse *mockStorageExtension) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (mse *mockStorageExtension) Shutdown(_ context.Context) error {
	return nil
}

func (mse *mockStorageExtension) GetClient(_ context.Context, _ component.Kind, _ config.ComponentID, _ string) (storage.Client, error) {
	return storage.NewNopClient(), nil
}

func TestGetRetrySettings(t *testing.T) {
	testCases := []struct {
		desc           string
		storage        storage.Extension
		numStorages    int
		storageID      string
		storageEnabled bool
		expectedError  error
	}{
		{
			desc:          "no storage selected",
			numStorages:   0,
			expectedError: errNoStorageClient,
		},
		{
			desc:          "obtain storage extension by name",
			numStorages:   2,
			storageID:     "1",
			expectedError: nil,
		},
		{
			desc:          "fail on not existing storage extension",
			numStorages:   2,
			storageID:     "100",
			expectedError: errNoStorageClient,
		},
		{
			desc:           "obtain default storage extension (deprecated)",
			numStorages:    1,
			storageEnabled: true,
			expectedError:  nil,
		},
		{
			desc:           "fail on obtaining default storage extension (deprecated)",
			numStorages:    2,
			storageEnabled: true,
			expectedError:  errMultipleStorageClients,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			// prepare
			cfg := &QueueSettings{
				Enabled:        true,
				StorageEnabled: tC.storageEnabled,
			}
			if tC.storageID != "" {
				compID := config.NewComponentIDWithName("file_storage", tC.storageID)
				cfg.StorageID = &compID
			}

			var extensions = map[config.ComponentID]component.Extension{}
			for i := 0; i < tC.numStorages; i++ {
				extensions[config.NewComponentIDWithName("file_storage", strconv.Itoa(i))] = &mockStorageExtension{}
			}
			host := &mockHost{ext: extensions}
			ownerID := config.NewComponentID("foo_exporter")

			// execute
			client, err := cfg.toStorageClient(context.Background(), zap.NewNop(), host, ownerID, config.TracesDataType)

			// verify
			if tC.expectedError != nil {
				assert.ErrorIs(t, err, tC.expectedError)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}
