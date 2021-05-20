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

package storagetest

import (
	"context"
	"testing"

	"go.uber.org/zap/zaptest"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/extension/storage"
	"go.opentelemetry.io/collector/extension/storage/filestorage"
)

type StorageHost struct {
	component.Host
	extensions map[config.ComponentID]component.Extension
}

func (h StorageHost) GetExtensions() map[config.ComponentID]component.Extension {
	return h.extensions
}

func NewStorageHost(t *testing.T, directory string, extensionNames ...string) StorageHost {
	h := StorageHost{
		Host:       componenttest.NewNopHost(),
		extensions: make(map[config.ComponentID]component.Extension),
	}

	for _, name := range extensionNames {
		h.extensions[newTestEntity(name)] = NewTestExtension(t, directory)
	}
	return h
}

func NewTestExtension(t *testing.T, directory string) storage.Extension {
	f := filestorage.NewFactory()
	cfg := f.CreateDefaultConfig().(*filestorage.Config)
	cfg.Directory = directory
	params := component.ExtensionCreateParams{Logger: zaptest.NewLogger(t)}
	extension, _ := f.CreateExtension(context.Background(), params, cfg)
	se, _ := extension.(storage.Extension)
	return se
}

func newTestEntity(name string) config.ComponentID {
	return config.NewIDWithName("nop", name)
}
