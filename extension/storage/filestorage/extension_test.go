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

package filestorage

import (
	"context"
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/extension/storage"
)

func TestExtensionIntegrity(t *testing.T) {
	ctx := context.Background()
	se := newTestExtension(t)

	type mockComponent struct {
		kind component.Kind
		name config.ComponentID
	}

	components := []mockComponent{
		{kind: component.KindReceiver, name: newTestEntity("receiver_one")},
		{kind: component.KindReceiver, name: newTestEntity("receiver_two")},
		{kind: component.KindProcessor, name: newTestEntity("processor_one")},
		{kind: component.KindProcessor, name: newTestEntity("processor_two")},
		{kind: component.KindExporter, name: newTestEntity("exporter_one")},
		{kind: component.KindExporter, name: newTestEntity("exporter_two")},
		{kind: component.KindExtension, name: newTestEntity("extension_one")},
		{kind: component.KindExtension, name: newTestEntity("extension_two")},
	}

	// Make a client for each component
	clients := make(map[config.ComponentID]storage.Client)
	for _, c := range components {
		client, err := se.GetClient(ctx, c.kind, c.name)
		require.NoError(t, err)
		clients[c.name] = client
	}

	thrashClient := func(wg *sync.WaitGroup, n config.ComponentID, c storage.Client) {
		// keys and values
		keys := []string{"a", "b", "c", "d", "e"}
		myBytes := []byte(n.Name())

		// Set my values
		for i := 0; i < len(keys); i++ {
			err := c.Set(ctx, keys[i], myBytes)
			require.NoError(t, err)
		}

		// Repeatedly thrash client
		for j := 0; j < 100; j++ {

			// Make sure my values are still mine
			for i := 0; i < len(keys); i++ {
				v, err := c.Get(ctx, keys[i])
				require.NoError(t, err)
				require.Equal(t, myBytes, v)
			}

			// Delete my values
			for i := 0; i < len(keys); i++ {
				err := c.Delete(ctx, keys[i])
				require.NoError(t, err)
			}

			// Reset my values
			for i := 0; i < len(keys); i++ {
				err := c.Set(ctx, keys[i], myBytes)
				require.NoError(t, err)
			}
		}
		wg.Done()
	}

	// Use clients concurrently
	var wg sync.WaitGroup
	for name, client := range clients {
		wg.Add(1)
		go thrashClient(&wg, name, client)
	}
	wg.Wait()
}

func TestShutdownClosesClients(t *testing.T) {
	ctx := context.Background()
	se := newTestExtension(t)

	myReceiverClient, err := se.GetClient(
		ctx,
		component.KindReceiver,
		newTestEntity("my_receiver"),
	)
	require.NoError(t, err)
	err = myReceiverClient.Set(ctx, "key", []byte("value"))
	require.NoError(t, err)

	myExporterClient, err := se.GetClient(
		ctx,
		component.KindReceiver,
		newTestEntity("my_exporter"),
	)
	require.NoError(t, err)
	err = myExporterClient.Set(ctx, "key", []byte("value"))
	require.NoError(t, err)

	// Shutdown should close clients
	require.NoError(t, se.Shutdown(ctx))

	err = myReceiverClient.Set(ctx, "key", []byte("value"))
	require.Error(t, err)
	require.Equal(t, err.Error(), "database not open")

	err = myExporterClient.Set(ctx, "key", []byte("value"))
	require.Error(t, err)
	require.Equal(t, err.Error(), "database not open")
}

func TestClientHandlesSimpleCases(t *testing.T) {
	ctx := context.Background()
	se := newTestExtension(t)

	client, err := se.GetClient(
		ctx,
		component.KindReceiver,
		newTestEntity("my_component"),
	)

	myBytes := []byte("value")
	require.NoError(t, err)

	// Set the data
	err = client.Set(ctx, "key", myBytes)
	require.NoError(t, err)

	// Set it again (nop does not error)
	err = client.Set(ctx, "key", myBytes)
	require.NoError(t, err)

	// Get actual data
	data, err := client.Get(ctx, "key")
	require.NoError(t, err)
	require.Equal(t, myBytes, data)

	// Delete the data
	err = client.Delete(ctx, "key")
	require.NoError(t, err)

	// Delete it again (nop does not error)
	err = client.Delete(ctx, "key")
	require.NoError(t, err)

	// Get missing data
	data, err = client.Get(ctx, "key")
	require.NoError(t, err)
	require.Nil(t, data)

}

func TestNewExtensionErrorsOnMissingDirectory(t *testing.T) {
	f := NewFactory()
	cfg := f.CreateDefaultConfig().(*Config)
	cfg.Directory = "/not/a/dir"

	params := component.ExtensionCreateParams{Logger: zaptest.NewLogger(t)}

	extension, err := f.CreateExtension(context.Background(), params, cfg)
	require.Error(t, err)
	require.Nil(t, extension)
}

func TestGetClientErrorsOnDeletedDirectory(t *testing.T) {
	ctx := context.Background()

	tempDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)

	f := NewFactory()
	cfg := f.CreateDefaultConfig().(*Config)
	cfg.Directory = tempDir

	params := component.ExtensionCreateParams{Logger: zaptest.NewLogger(t)}

	extension, err := f.CreateExtension(context.Background(), params, cfg)
	require.NoError(t, err)

	se, ok := extension.(storage.Extension)
	require.True(t, ok)

	// Delete the directory before getting client
	err = os.RemoveAll(tempDir)
	require.NoError(t, err)

	client, err := se.GetClient(
		ctx,
		component.KindReceiver,
		newTestEntity("my_component"),
	)

	require.Error(t, err)
	require.Nil(t, client)
}

func newTestExtension(t *testing.T) storage.Extension {
	tempDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)

	f := NewFactory()
	cfg := f.CreateDefaultConfig().(*Config)
	cfg.Directory = tempDir

	params := component.ExtensionCreateParams{Logger: zaptest.NewLogger(t)}

	extension, err := f.CreateExtension(context.Background(), params, cfg)
	require.NoError(t, err)

	se, ok := extension.(storage.Extension)
	require.True(t, ok)

	return se
}

func newTestEntity(name string) config.ComponentID {
	return config.NewIDWithName("nop", name)
}
