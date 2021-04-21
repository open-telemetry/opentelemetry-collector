// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stanza

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/storage"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/storage/filestorage"
	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
	"go.uber.org/zap/zaptest"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
)

func TestStorage(t *testing.T) {
	ctx := context.Background()
	tempDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)

	r := createReceiver(t)
	host := getHostWithStorage(t, tempDir)
	err = r.Start(ctx, host)
	require.NoError(t, err)

	myBytes := []byte("my_value")

	r.storageClient.Set(ctx, "key", myBytes)
	val, err := r.storageClient.Get(ctx, "key")
	require.NoError(t, err)
	require.Equal(t, myBytes, val)

	// Cycle the receiver
	require.NoError(t, r.Shutdown(ctx))
	require.NoError(t, host.ShutdownExtensions(ctx))

	r = createReceiver(t)
	err = r.Start(ctx, host)
	require.NoError(t, err)

	// Value has persisted
	val, err = r.storageClient.Get(ctx, "key")
	require.NoError(t, err)
	require.Equal(t, myBytes, val)

	err = r.storageClient.Delete(ctx, "key")
	require.NoError(t, err)

	// Value is gone
	val, err = r.storageClient.Get(ctx, "key")
	require.NoError(t, err)
	require.Nil(t, val)

	require.NoError(t, r.Shutdown(ctx))
}

func TestFailOnMultipleStorageExtensions(t *testing.T) {
	ctx := context.Background()
	tempDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)

	r := createReceiver(t)
	host := getHostWithMultipleStorage(t, tempDir)
	err = r.Start(ctx, host)
	require.Error(t, err)
	require.Equal(t, "storage client: multiple storage extensions found", err.Error())
}

func createReceiver(t *testing.T) *receiver {
	params := component.ReceiverCreateParams{
		Logger: zaptest.NewLogger(t),
	}
	mockConsumer := mockLogsConsumer{}

	factory := NewFactory(TestReceiverType{})

	logsReceiver, err := factory.CreateLogsReceiver(
		context.Background(),
		params,
		factory.CreateDefaultConfig(),
		&mockConsumer,
	)
	require.NoError(t, err, "receiver should successfully build")

	r, ok := logsReceiver.(*receiver)
	require.True(t, ok)
	return r
}

type hostWithStorage struct {
	component.Host
	extensions map[config.NamedEntity]component.Extension
}

func (h hostWithStorage) GetExtensions() map[config.NamedEntity]component.Extension {
	return h.extensions
}

func (h hostWithStorage) ShutdownExtensions(ctx context.Context) error {
	errs := []error{}
	for _, e := range h.extensions {
		errs = append(errs, e.Shutdown(ctx))
	}
	return multierr.Combine(errs...)
}

func getHostWithStorage(t *testing.T, directory string) hostWithStorage {
	return hostWithStorage{
		Host: componenttest.NewNopHost(),
		extensions: map[config.NamedEntity]component.Extension{
			newTestEntity("my_extension"): newTestExtension(t, directory),
		},
	}
}

func getHostWithMultipleStorage(t *testing.T, directory string) hostWithStorage {
	return hostWithStorage{
		Host: componenttest.NewNopHost(),
		extensions: map[config.NamedEntity]component.Extension{
			newTestEntity("my_extension_one"): newTestExtension(t, directory),
			newTestEntity("my_extension_two"): newTestExtension(t, directory),
		},
	}
}

func newTestEntity(name string) config.NamedEntity {
	return &config.ExporterSettings{TypeVal: "nop", NameVal: name}
}

func newTestExtension(t *testing.T, directory string) storage.Extension {
	f := filestorage.NewFactory()
	cfg := f.CreateDefaultConfig().(*filestorage.Config)
	cfg.Directory = directory
	params := component.ExtensionCreateParams{Logger: zaptest.NewLogger(t)}
	extension, _ := f.CreateExtension(context.Background(), params, cfg)
	se, _ := extension.(storage.Extension)
	return se
}

func TestPersisterImplementation(t *testing.T) {
	ctx := context.Background()
	myBytes := []byte("string")
	p := newMockPersister()

	err := p.Set(ctx, "key", myBytes)
	require.NoError(t, err)

	val, err := p.Get(ctx, "key")
	require.NoError(t, err)
	require.Equal(t, myBytes, val)

	err = p.Delete(ctx, "key")
	require.NoError(t, err)
}
