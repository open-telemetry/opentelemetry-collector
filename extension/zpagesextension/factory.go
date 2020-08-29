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

package zpagesextension

import (
	"context"
	"errors"
	"sync/atomic"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/extension/extensionhelper"
)

const (
	// The value of extension "type" in configuration.
	typeStr = "zpages"
)

// NewFactory creates a factory for Z-Pages extension.
func NewFactory() component.ExtensionFactory {
	return extensionhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		createExtension)
}

func createDefaultConfig() configmodels.Extension {
	return &Config{
		ExtensionSettings: configmodels.ExtensionSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		Endpoint: "localhost:55679",
	}
}

// createExtension creates the extension based on this config.
func createExtension(_ context.Context, params component.ExtensionCreateParams, cfg configmodels.Extension) (component.ServiceExtension, error) {
	config := cfg.(*Config)
	if config.Endpoint == "" {
		return nil, errors.New("\"endpoint\" is required when using the \"zpages\" extension")
	}

	// The runtime settings are global to the application, so while in principle it
	// is possible to have more than one instance, running multiple does not bring
	// any value to the service.
	// In order to avoid this issue we will allow the creation of a single
	// instance once per process while keeping the private function that allow
	// the creation of multiple instances for unit tests. Summary: only a single
	// instance can be created via the factory.
	if !atomic.CompareAndSwapInt32(&instanceState, instanceNotCreated, instanceCreated) {
		return nil, errors.New("only a single zpages extension instance can be created per process")
	}

	return newServer(*config, params.Logger), nil
}

// See comment in CreateExtension how these are used.
var instanceState int32

const (
	instanceNotCreated int32 = 0
	instanceCreated    int32 = 1
)
