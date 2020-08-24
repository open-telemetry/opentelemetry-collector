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
	"errors"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
)

const typeStr = "test"

var (
	defaultCfg = &configmodels.ExtensionSettings{
		TypeVal: typeStr,
		NameVal: typeStr,
	}
	nopExtensionInstance = new(nopExtension)
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory(
		typeStr,
		defaultConfig,
		createExtension)
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, defaultCfg, factory.CreateDefaultConfig())
	ext, err := factory.CreateExtension(context.Background(), component.ExtensionCreateParams{}, defaultCfg)
	assert.NoError(t, err)
	assert.Same(t, nopExtensionInstance, ext)
}

func TestNewFactory_WithConstructors(t *testing.T) {
	factory := NewFactory(
		typeStr,
		defaultConfig,
		createExtension,
		WithCustomUnmarshaler(customUnmarshaler))
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, defaultCfg, factory.CreateDefaultConfig())

	fu, ok := factory.(component.ConfigUnmarshaler)
	assert.True(t, ok)
	assert.Equal(t, errors.New("my error"), fu.Unmarshal(nil, nil))

	ext, err := factory.CreateExtension(context.Background(), component.ExtensionCreateParams{}, defaultCfg)
	assert.NoError(t, err)
	assert.Same(t, nopExtensionInstance, ext)
}

func defaultConfig() configmodels.Extension {
	return defaultCfg
}

func createExtension(context.Context, component.ExtensionCreateParams, configmodels.Extension) (component.ServiceExtension, error) {
	return nopExtensionInstance, nil
}

func customUnmarshaler(*viper.Viper, interface{}) error {
	return errors.New("my error")
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
