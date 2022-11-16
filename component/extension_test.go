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

// TODO: Move tests back to component package after config.*Settings are removed.

package component_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
)

type nopExtension struct {
	component.StartFunc
	component.ShutdownFunc
}

func TestNewExtensionFactory(t *testing.T) {
	const typeStr = "test"
	defaultCfg := config.NewExtensionSettings(component.NewID(typeStr))
	nopExtensionInstance := new(nopExtension)

	factory := component.NewExtensionFactory(
		typeStr,
		func() component.ExtensionConfig { return &defaultCfg },
		func(ctx context.Context, settings component.ExtensionCreateSettings, extension component.ExtensionConfig) (component.Extension, error) {
			return nopExtensionInstance, nil
		},
		component.StabilityLevelDevelopment)
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	assert.Equal(t, component.StabilityLevelDevelopment, factory.ExtensionStability())
	ext, err := factory.CreateExtension(context.Background(), component.ExtensionCreateSettings{}, &defaultCfg)
	assert.NoError(t, err)
	assert.Same(t, nopExtensionInstance, ext)
}
