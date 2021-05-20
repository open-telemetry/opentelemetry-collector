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
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/extension/storage"
)

type localFileStorage struct {
	directory string
	timeout   time.Duration
	logger    *zap.Logger
	clients   []*fileStorageClient
}

// Ensure this storage extension implements the appropriate interface
var _ storage.Extension = (*localFileStorage)(nil)

func newLocalFileStorage(logger *zap.Logger, config *Config) (component.Extension, error) {
	info, err := os.Stat(config.Directory)
	if (err != nil && os.IsNotExist(err)) || !info.IsDir() {
		return nil, fmt.Errorf("directory must exist: %v", err)
	}

	return &localFileStorage{
		directory: filepath.Clean(config.Directory),
		timeout:   config.Timeout,
		logger:    logger,
		clients:   []*fileStorageClient{},
	}, nil
}

// Start does nothing
func (lfs *localFileStorage) Start(context.Context, component.Host) error {
	return nil
}

// Shutdown will close any open databases
func (lfs *localFileStorage) Shutdown(context.Context) error {
	for _, client := range lfs.clients {
		client.close()
	}
	// TODO clean up data files that did not have a client
	// and are older than a threshold (possibly configurable)
	return nil
}

// GetClient returns a storage client for an individual component
func (lfs *localFileStorage) GetClient(ctx context.Context, kind component.Kind, ent config.ComponentID) (storage.Client, error) {
	rawName := fmt.Sprintf("%s_%s_%s", kindString(kind), ent.Type(), ent.Name())
	// TODO sanitize rawName
	absoluteName := filepath.Join(lfs.directory, rawName)

	client, err := newClient(absoluteName, lfs.timeout)
	if err != nil {
		return nil, fmt.Errorf("create client: %v", err)
	}
	lfs.clients = append(lfs.clients, client)
	return client, nil
}

func kindString(k component.Kind) string {
	switch k {
	case component.KindReceiver:
		return "receiver"
	case component.KindProcessor:
		return "processor"
	case component.KindExporter:
		return "exporter"
	case component.KindExtension:
		return "extension"
	default:
		return "other" // not expected
	}
}
