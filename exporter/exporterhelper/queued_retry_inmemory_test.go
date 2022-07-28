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
	"errors"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
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

type mockStorageExtension struct {
	GetClientError error
}

func (mse *mockStorageExtension) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (mse *mockStorageExtension) Shutdown(_ context.Context) error {
	return nil
}

func (mse *mockStorageExtension) GetClient(_ context.Context, _ component.Kind, _ config.ComponentID, _ string) (storage.Client, error) {
	if mse.GetClientError != nil {
		return nil, mse.GetClientError
	}
	return storage.NewNopClient(), nil
}

func TestGetRetrySettings(t *testing.T) {
	getStorageClientError := errors.New("unable to create storage client")
	testCases := []struct {
		desc           string
		storage        storage.Extension
		numStorages    int
		storageID      string
		expectedError  error
		getClientError error
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
			desc:           "fail on error getting storage client from extension",
			numStorages:    1,
			storageID:      "0",
			expectedError:  getStorageClientError,
			getClientError: getStorageClientError,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			// prepare
			cfg := &QueueSettings{
				Enabled: true,
			}
			if tC.storageID != "" {
				compID := config.NewComponentIDWithName("file_storage", tC.storageID)
				cfg.StorageID = &compID
			}

			var extensions = map[config.ComponentID]component.Extension{}
			for i := 0; i < tC.numStorages; i++ {
				extensions[config.NewComponentIDWithName("file_storage", strconv.Itoa(i))] = &mockStorageExtension{GetClientError: tC.getClientError}
			}
			host := &mockHost{ext: extensions}
			ownerID := config.NewComponentID("foo_exporter")

			// execute
			client, err := cfg.toStorageClient(context.Background(), host, ownerID, config.TracesDataType)

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

func TestInvalidStorageExtensionType(t *testing.T) {
	// prepare
	cfg := &QueueSettings{
		Enabled: true,
	}
	compID := config.NewComponentIDWithName("extension", "extension")
	cfg.StorageID = &compID

	// make a test extension
	factory := componenttest.NewNopExtensionFactory()
	extConfig := factory.CreateDefaultConfig()
	settings := componenttest.NewNopExtensionCreateSettings()
	extension, err := factory.CreateExtension(context.Background(), settings, extConfig)
	assert.NoError(t, err)
	var extensions = map[config.ComponentID]component.Extension{
		compID: extension,
	}
	host := &mockHost{ext: extensions}
	ownerID := config.NewComponentID("foo_exporter")

	// execute
	client, err := cfg.toStorageClient(context.Background(), host, ownerID, config.TracesDataType)

	// we should get an error about the extension type
	assert.ErrorIs(t, err, errWrongExtensionType)
	assert.Nil(t, client)
}
