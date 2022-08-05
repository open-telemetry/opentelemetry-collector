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
// limitations under the License.

//go:build enable_unstable
// +build enable_unstable

package exporterhelper

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/extension/experimental/storage"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
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
		storageIndex   int
		expectedError  error
		getClientError error
	}{
		{
			desc:          "obtain storage extension by name",
			numStorages:   2,
			storageIndex:  0,
			expectedError: nil,
		},
		{
			desc:          "fail on not existing storage extension",
			numStorages:   2,
			storageIndex:  100,
			expectedError: errNoStorageClient,
		},
		{
			desc:          "invalid extension type",
			numStorages:   2,
			storageIndex:  100,
			expectedError: errNoStorageClient,
		},
		{
			desc:           "fail on error getting storage client from extension",
			numStorages:    1,
			storageIndex:   0,
			expectedError:  getStorageClientError,
			getClientError: getStorageClientError,
		},
	}

	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			storageID := config.NewComponentIDWithName("file_storage", strconv.Itoa(tC.storageIndex))

			var extensions = map[config.ComponentID]component.Extension{}
			for i := 0; i < tC.numStorages; i++ {
				extensions[config.NewComponentIDWithName("file_storage", strconv.Itoa(i))] = &mockStorageExtension{GetClientError: tC.getClientError}
			}
			host := &mockHost{ext: extensions}
			ownerID := config.NewComponentID("foo_exporter")

			// execute
			client, err := toStorageClient(context.Background(), storageID, host, ownerID, config.TracesDataType)

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
	storageID := config.NewComponentIDWithName("extension", "extension")

	// make a test extension
	factory := componenttest.NewNopExtensionFactory()
	extConfig := factory.CreateDefaultConfig()
	settings := componenttest.NewNopExtensionCreateSettings()
	extension, err := factory.CreateExtension(context.Background(), settings, extConfig)
	assert.NoError(t, err)
	var extensions = map[config.ComponentID]component.Extension{
		storageID: extension,
	}
	host := &mockHost{ext: extensions}
	ownerID := config.NewComponentID("foo_exporter")

	// execute
	client, err := toStorageClient(context.Background(), storageID, host, ownerID, config.TracesDataType)

	// we should get an error about the extension type
	assert.ErrorIs(t, err, errWrongExtensionType)
	assert.Nil(t, client)
}

// if requeueing is enabled, we eventually retry even if we failed at first
func TestQueuedRetry_RequeuingEnabled(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 1
	rCfg := NewDefaultRetrySettings()
	rCfg.MaxElapsedTime = time.Nanosecond // we don't want to retry at all, but requeue instead
	be := newBaseExporter(&defaultExporterCfg, componenttest.NewNopExporterCreateSettings(), fromOptions(WithRetry(rCfg), WithQueue(qCfg)), "", nopRequestUnmarshaler())
	ocs := newObservabilityConsumerSender(be.qrSender.consumerSender)
	be.qrSender.consumerSender = ocs
	be.qrSender.requeuingEnabled = true
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	traceErr := consumererror.NewTraces(errors.New("some error"), testdata.GenerateTraces(1))
	mockR := newMockRequest(context.Background(), 1, traceErr)
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, be.sender.send(mockR))
		ocs.waitGroup.Add(1) // necessary because we'll call send() again after requeueing
	})
	ocs.awaitAsyncProcessing()

	// In the newMockConcurrentExporter we count requests and items even for failed requests
	mockR.checkNumRequests(t, 2)
	ocs.checkSendItemsCount(t, 1)
	ocs.checkDroppedItemsCount(t, 1) // not actually dropped, but ocs counts each failed send here
}

// if requeueing is enabled, but the queue is full, we get an error
func TestQueuedRetry_RequeuingEnabledQueueFull(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 0
	qCfg.QueueSize = 0
	rCfg := NewDefaultRetrySettings()
	rCfg.MaxElapsedTime = time.Nanosecond // we don't want to retry at all, but requeue instead
	be := newBaseExporter(&defaultExporterCfg, componenttest.NewNopExporterCreateSettings(), fromOptions(WithRetry(rCfg), WithQueue(qCfg)), "", nopRequestUnmarshaler())
	be.qrSender.requeuingEnabled = true
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	traceErr := consumererror.NewTraces(errors.New("some error"), testdata.GenerateTraces(1))
	mockR := newMockRequest(context.Background(), 1, traceErr)

	require.Error(t, be.qrSender.consumerSender.send(mockR), "sending_queue is full")
	mockR.checkNumRequests(t, 1)
}

func TestQueuedRetryPersistenceEnabled(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	qCfg := NewDefaultQueueSettings()
	storageID := config.NewComponentIDWithName("file_storage", "storage")
	qCfg.StorageID = &storageID // enable persistence
	rCfg := NewDefaultRetrySettings()
	be := newBaseExporter(&defaultExporterCfg, tt.ToExporterCreateSettings(), fromOptions(WithRetry(rCfg), WithQueue(qCfg)), "", nopRequestUnmarshaler())

	var extensions = map[config.ComponentID]component.Extension{
		storageID: &mockStorageExtension{},
	}
	host := &mockHost{ext: extensions}

	// we start correctly with a file storage extension
	require.NoError(t, be.Start(context.Background(), host))
	require.NoError(t, be.Shutdown(context.Background()))
}

func TestQueuedRetryPersistenceEnabledStorageError(t *testing.T) {
	storageError := errors.New("could not get storage client")
	tt, err := obsreporttest.SetupTelemetry()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	qCfg := NewDefaultQueueSettings()
	storageID := config.NewComponentIDWithName("file_storage", "storage")
	qCfg.StorageID = &storageID // enable persistence
	rCfg := NewDefaultRetrySettings()
	be := newBaseExporter(&defaultExporterCfg, tt.ToExporterCreateSettings(), fromOptions(WithRetry(rCfg), WithQueue(qCfg)), "", nopRequestUnmarshaler())

	var extensions = map[config.ComponentID]component.Extension{
		storageID: &mockStorageExtension{GetClientError: storageError},
	}
	host := &mockHost{ext: extensions}

	// we fail to start if we get an error creating the storage client
	require.Error(t, be.Start(context.Background(), host), "could not get storage client")
}
