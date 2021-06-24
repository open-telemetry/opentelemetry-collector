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

package extensionhelper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
)

const typeStr = "test"

var (
	defaultCfg           = config.NewExtensionSettings(config.NewID(typeStr))
	nopExtensionInstance = new(nopExtension)
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory(
		typeStr,
		defaultConfig,
		createExtension)
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())
	ext, err := factory.CreateExtension(context.Background(), componenttest.NewNopExtensionCreateSettings(), &defaultCfg)
	assert.NoError(t, err)
	assert.Same(t, nopExtensionInstance, ext)
}

func defaultConfig() config.Extension {
	return &defaultCfg
}

func createExtension(context.Context, component.ExtensionCreateSettings, config.Extension) (component.Extension, error) {
	return nopExtensionInstance, nil
}

type nopExtension struct {
}

func (ne *nopExtension) Start(context.Context, component.Host) error {
	return nil
}

// Shutdown stops the exporter and is invoked during shutdown.
func (ne *nopExtension) Shutdown(context.Context) error {
	return nil
}
